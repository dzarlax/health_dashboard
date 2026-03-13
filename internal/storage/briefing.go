package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"health-receiver/internal/health"
)

// GetHealthBriefing fetches raw metric time series from the DB and delegates
// all scoring and insight computation to the health package.
// lang selects the output language ("en", "ru", "sr").
func (s *DB) GetHealthBriefing(lang string) (*health.BriefingResponse, error) {
	var lastDate string
	if err := s.db.QueryRow(`SELECT MAX(substr(date,1,10)) FROM metric_points WHERE qty > 0`).Scan(&lastDate); err != nil || lastDate == "" {
		return &health.BriefingResponse{Greeting: "Welcome! No health data yet."}, nil
	}

	// getDailyValues returns per-day aggregated values.
	// Sleep metrics use MAX across sources (two devices record the same night
	// independently). All other metrics use flat aggregation (HealthKit
	// already deduplicates cumulative metrics at sample level via pipe-joined sources).
	getDailyValues := func(metric string, days int, agg string) []float64 {
		var rows *sql.Rows
		var err error
		if agg == "SUM" && strings.HasPrefix(metric, "sleep_") {
			rows, err = s.db.Query(`
				SELECT MAX(source_sum)
				FROM (
					SELECT substr(date,1,10) AS d, source, SUM(qty) AS source_sum
					FROM metric_points
					WHERE metric_name = ? AND substr(date,1,10) >= ? AND qty > 0
					GROUP BY d, source
				)
				GROUP BY d
				ORDER BY d DESC
				LIMIT ?`,
				metric, subtractDays(lastDate, days), days)
		} else {
			rows, err = s.db.Query(`
				SELECT `+agg+`(qty)
				FROM metric_points
				WHERE metric_name = ? AND substr(date,1,10) >= ? AND qty > 0
				GROUP BY substr(date,1,10)
				ORDER BY substr(date,1,10) DESC
				LIMIT ?`,
				metric, subtractDays(lastDate, days), days)
		}
		if err != nil {
			return nil
		}
		defer rows.Close()
		var vals []float64
		for rows.Next() {
			var v float64
			if err := rows.Scan(&v); err == nil {
				vals = append(vals, v)
			}
		}
		return vals
	}

	getDailyWithDates := func(metric string, days int, agg string) []health.DatedValue {
		var rows *sql.Rows
		var err error
		if agg == "SUM" && strings.HasPrefix(metric, "sleep_") {
			rows, err = s.db.Query(`
				SELECT d, MAX(source_sum)
				FROM (
					SELECT substr(date,1,10) AS d, source, SUM(qty) AS source_sum
					FROM metric_points
					WHERE metric_name = ? AND substr(date,1,10) >= ? AND qty > 0
					GROUP BY d, source
				)
				GROUP BY d
				ORDER BY d DESC
				LIMIT ?`,
				metric, subtractDays(lastDate, days), days)
		} else {
			rows, err = s.db.Query(`
				SELECT substr(date,1,10), `+agg+`(qty)
				FROM metric_points
				WHERE metric_name = ? AND substr(date,1,10) >= ? AND qty > 0
				GROUP BY substr(date,1,10)
				ORDER BY substr(date,1,10) DESC
				LIMIT ?`,
				metric, subtractDays(lastDate, days), days)
		}
		if err != nil {
			return nil
		}
		defer rows.Close()
		var out []health.DatedValue
		for rows.Next() {
			var dv health.DatedValue
			if err := rows.Scan(&dv.Date, &dv.Val); err == nil {
				out = append(out, dv)
			}
		}
		return out
	}

	data := health.RawMetrics{
		LastDate:       lastDate,
		HRV:            getDailyValues("heart_rate_variability", 30, "AVG"),
		RHR:            getDailyValues("resting_heart_rate", 30, "AVG"),
		Sleep:          getDailyValues("sleep_total", 30, "SUM"),
		Deep:           getDailyValues("sleep_deep", 30, "SUM"),
		REM:            getDailyValues("sleep_rem", 30, "SUM"),
		Awake:          getDailyValues("sleep_awake", 30, "SUM"),
		Steps:          getDailyValues("step_count", 30, "SUM"),
		Cal:            getDailyValues("active_energy", 30, "SUM"),
		Exercise:       getDailyValues("apple_exercise_time", 30, "SUM"),
		SpO2:           getDailyValues("blood_oxygen_saturation", 30, "AVG"),
		VO2:            getDailyValues("vo2_max", 30, "AVG"),
		Resp:           getDailyValues("respiratory_rate", 30, "AVG"),
		StepsWithDates: getDailyWithDates("step_count", 7, "SUM"),
		HRVWithDates:   getDailyWithDates("heart_rate_variability", 7, "AVG"),
	}

	resp := health.ComputeBriefing(data, lang)

	// Attach per-source sleep breakdown for the most recent night.
	if resp.Sleep != nil {
		sleepRows, qErr := s.db.Query(`
			SELECT SUBSTR(source, 1, INSTR(source || '|', '|') - 1) AS src,
				SUM(CASE WHEN metric_name='sleep_total' THEN qty ELSE 0 END),
				SUM(CASE WHEN metric_name='sleep_deep'  THEN qty ELSE 0 END),
				SUM(CASE WHEN metric_name='sleep_rem'   THEN qty ELSE 0 END),
				SUM(CASE WHEN metric_name='sleep_core'  THEN qty ELSE 0 END),
				SUM(CASE WHEN metric_name='sleep_awake' THEN qty ELSE 0 END)
			FROM metric_points
			WHERE metric_name IN ('sleep_total','sleep_deep','sleep_rem','sleep_core','sleep_awake')
			  AND substr(date,1,10) = ? AND qty > 0
			GROUP BY src
			ORDER BY SUM(CASE WHEN metric_name='sleep_total' THEN qty ELSE 0 END) DESC`,
			lastDate)
		if qErr == nil {
			defer sleepRows.Close()
			for sleepRows.Next() {
				var ss health.SleepSourceSummary
				if sErr := sleepRows.Scan(&ss.Source, &ss.Total, &ss.Deep, &ss.REM, &ss.Core, &ss.Awake); sErr == nil && ss.Total > 0 {
					resp.Sleep.Sources = append(resp.Sleep.Sources, ss)
				}
			}
		}
	}
	return resp, nil
}

// GetReadinessHistory computes a historical readiness score for each of the
// last `outputDays` days using a sliding 30-day baseline window.
// For each output day D the function uses HRV/RHR/sleep data from D-29..D
// (most-recent-first) and calls health.ComputeReadinessScore.
func (s *DB) GetReadinessHistory(outputDays int) ([]health.ReadinessPoint, error) {
	window := 30
	total := outputDays + window

	// Fetch date-keyed maps for the full look-back period.
	fetch := func(metric, agg string, isSleep bool) (map[string]float64, error) {
		var rows *sql.Rows
		var err error
		if isSleep {
			rows, err = s.db.Query(`
				SELECT d, MAX(source_sum)
				FROM (
					SELECT substr(date,1,10) AS d, source, SUM(qty) AS source_sum
					FROM metric_points
					WHERE metric_name = ?
					  AND qty > 0
					  AND substr(date,1,10) >= date('now', ? || ' days')
					GROUP BY d, source
				)
				GROUP BY d`,
				metric, fmt.Sprintf("-%d", total))
		} else {
			rows, err = s.db.Query(`
				SELECT substr(date,1,10), `+agg+`(qty)
				FROM metric_points
				WHERE metric_name = ?
				  AND qty > 0
				  AND substr(date,1,10) >= date('now', ? || ' days')
				GROUP BY substr(date,1,10)`,
				metric, fmt.Sprintf("-%d", total))
		}
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		m := make(map[string]float64)
		for rows.Next() {
			var d string
			var v float64
			if rows.Scan(&d, &v) == nil {
				m[d] = v
			}
		}
		return m, nil
	}

	hrvMap, err := fetch("heart_rate_variability", "AVG", false)
	if err != nil {
		return nil, err
	}
	rhrMap, err := fetch("resting_heart_rate", "AVG", false)
	if err != nil {
		return nil, err
	}
	sleepMap, err := fetch("sleep_total", "SUM", true)
	if err != nil {
		return nil, err
	}

	// Build a sorted list of all days we have any data for.
	dateSet := make(map[string]bool)
	for d := range hrvMap {
		dateSet[d] = true
	}
	for d := range rhrMap {
		dateSet[d] = true
	}
	for d := range sleepMap {
		dateSet[d] = true
	}
	allDates := make([]string, 0, len(dateSet))
	for d := range dateSet {
		allDates = append(allDates, d)
	}
	// sort ascending
	for i := 0; i < len(allDates)-1; i++ {
		for j := i + 1; j < len(allDates); j++ {
			if allDates[i] > allDates[j] {
				allDates[i], allDates[j] = allDates[j], allDates[i]
			}
		}
	}

	// For each output day (last outputDays dates) compute score using 30-day window.
	if len(allDates) > outputDays {
		allDates = allDates[len(allDates)-outputDays:]
	}

	// valsBefore returns values for all dates <= anchor, sorted by DATE descending
	// (most recent first) so that vals[:3] is the last 3 days, vals[3:] is the
	// historical baseline. Sorting by value (as before) was a bug: it put the
	// best HRV days first, artificially inflating the "recent" average.
	valsBefore := func(m map[string]float64, anchor string) []float64 {
		type dateval struct{ d string; v float64 }
		var pairs []dateval
		for d, v := range m {
			if d <= anchor {
				pairs = append(pairs, dateval{d, v})
			}
		}
		// sort by date descending
		for i := 0; i < len(pairs)-1; i++ {
			for j := i + 1; j < len(pairs); j++ {
				if pairs[i].d < pairs[j].d {
					pairs[i], pairs[j] = pairs[j], pairs[i]
				}
			}
		}
		if len(pairs) > window {
			pairs = pairs[:window]
		}
		out := make([]float64, len(pairs))
		for i, p := range pairs {
			out[i] = p.v
		}
		return out
	}

	out := make([]health.ReadinessPoint, 0, len(allDates))
	for _, d := range allDates {
		hrv := valsBefore(hrvMap, d)
		rhr := valsBefore(rhrMap, d)
		sleep := valsBefore(sleepMap, d)
		score := health.ComputeReadinessScore(hrv, rhr, sleep)
		out = append(out, health.ReadinessPoint{Date: d, Score: score})
	}
	return out, nil
}

func subtractDays(dateStr string, days int) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.AddDate(0, 0, -days).Format("2006-01-02")
}
