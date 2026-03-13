package storage

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// aggFuncFor returns the aggregation function name for a metric.
// SUM metrics accumulate within a period; all others are averaged.
func aggFuncFor(metric string) string {
	if SumMetrics[metric] {
		return "SUM"
	}
	return "AVG"
}

// combineFuncFor returns the SQL aggregate to combine per-source pre-computed
// values when merging sources at query time.
//   - sleep SUM metrics: MAX(per-source sum) avoids double-counting two wearables
//   - other SUM metrics: SUM — each source is independent (HealthKit deduplicates)
//   - AVG metrics: AVG
func combineFuncFor(metric string) string {
	if SumMetrics[metric] {
		if strings.HasPrefix(metric, "sleep_") {
			return "MAX"
		}
		return "SUM"
	}
	return "AVG"
}

// SumMetrics is the canonical set of metrics that should be SUMmed within a bucket.
// Exported so the MCP server can use the same classification without duplication.
var SumMetrics = map[string]bool{
	"step_count": true, "active_energy": true, "basal_energy_burned": true,
	"apple_exercise_time": true, "apple_stand_time": true,
	"flights_climbed": true, "walking_running_distance": true,
	"time_in_daylight": true, "apple_stand_hour": true,
	// sleep phases are SUM'd per source, then MAX'd across sources
	"sleep_total": true, "sleep_deep": true, "sleep_rem": true,
	"sleep_core": true, "sleep_awake": true,
}

func (s *DB) listMetricNames() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT metric_name FROM metric_points ORDER BY metric_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		rows.Scan(&m)
		out = append(out, m)
	}
	return out, rows.Err()
}

// InvalidateRecentAggregates deletes pre-aggregated rows for the last `hours`
// hours from both minute_metrics and hourly_metrics. Safe to call from a goroutine.
func (s *DB) InvalidateRecentAggregates(hours int) {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour).Format("2006-01-02 15:04")
	for _, tbl := range []string{"minute_metrics", "hourly_metrics"} {
		col := "minute"
		if tbl == "hourly_metrics" {
			col = "hour"
		}
		if _, err := s.db.Exec(
			fmt.Sprintf("DELETE FROM %s WHERE %s >= ?", tbl, col), cutoff,
		); err != nil {
			log.Printf("invalidate %s: %v", tbl, err)
		}
	}
}

// BackfillAggregates (re)builds minute_metrics and hourly_metrics for all
// available data, cascading: metric_points → minute_metrics → hourly_metrics.
// If force=true the tables are truncated first; otherwise only rows missing
// from the cache are computed.
func (s *DB) BackfillAggregates(force bool) error {
	if force {
		for _, tbl := range []string{"minute_metrics", "hourly_metrics"} {
			if _, err := s.db.Exec("DELETE FROM " + tbl); err != nil {
				return fmt.Errorf("clear %s: %w", tbl, err)
			}
		}
		log.Println("minute_metrics and hourly_metrics cleared")
	}

	metrics, err := s.listMetricNames()
	if err != nil {
		return fmt.Errorf("list metrics: %w", err)
	}

	log.Printf("backfill aggregates: %d metrics", len(metrics))

	for _, m := range metrics {
		agg := aggFuncFor(m)
		if err := s.buildMinuteMetric(m, agg, force); err != nil {
			log.Printf("  minute %s: %v", m, err)
		}
		if err := s.buildHourlyMetric(m, agg, force); err != nil {
			log.Printf("  hourly %s: %v", m, err)
		}
	}

	// Level 3: hourly_metrics → daily_scores metric columns.
	if err := s.BuildDailyMetrics(force); err != nil {
		return fmt.Errorf("daily metrics: %w", err)
	}

	log.Println("backfill aggregates done")
	return nil
}

// BuildDailyMetrics fills the metric columns of daily_scores from hourly_metrics.
// This is Level 3 of the cascade: hourly → daily.
// Existing readiness/score_version columns are not touched.
func (s *DB) BuildDailyMetrics(force bool) error {
	type spec struct {
		col   string
		name  string
		sleep bool // needs MAX(source_sum) dedup across devices
	}
	specs := []spec{
		{"hrv_avg", "heart_rate_variability", false},
		{"rhr_avg", "resting_heart_rate", false},
		{"sleep_total", "sleep_total", true},
		{"sleep_deep", "sleep_deep", true},
		{"sleep_rem", "sleep_rem", true},
		{"sleep_core", "sleep_core", true},
		{"sleep_awake", "sleep_awake", true},
		{"steps", "step_count", false},
		{"calories", "active_energy", false},
		{"exercise_min", "apple_exercise_time", false},
		{"spo2_avg", "blood_oxygen_saturation", false},
		{"vo2_avg", "vo2_max", false},
		{"resp_avg", "respiratory_rate", false},
	}

	for _, sp := range specs {
		if err := s.buildDailyMetricCol(sp.col, sp.name, sp.sleep, force); err != nil {
			log.Printf("  daily %s (%s): %v", sp.col, sp.name, err)
		}
	}
	log.Printf("daily metrics filled (%d columns)", len(specs))
	return nil
}

func (s *DB) buildDailyMetricCol(col, metric string, isSleep, force bool) error {
	var fromClause string
	if !force {
		// Only fill dates that don't have this column set yet.
		var lastFilled string
		s.db.QueryRow(
			`SELECT MAX(date) FROM daily_scores WHERE `+col+` IS NOT NULL`,
		).Scan(&lastFilled)
		if lastFilled != "" {
			fromClause = fmt.Sprintf("AND substr(hour,1,10) > '%s'", lastFilled)
		}
	}

	var query string
	if isSleep {
		// For sleep metrics two devices each record the same night independently.
		// Per source: SUM hours across day. Across sources: MAX to avoid double-count.
		query = fmt.Sprintf(`
			SELECT day, MAX(src_sum) FROM (
				SELECT substr(hour,1,10) AS day, source, SUM(avg_val) AS src_sum
				FROM hourly_metrics
				WHERE metric_name = ? %s
				GROUP BY day, source
			)
			GROUP BY day`, fromClause)
	} else {
		agg := aggFuncFor(metric)
		query = fmt.Sprintf(`
			SELECT substr(hour,1,10), %s(avg_val)
			FROM hourly_metrics
			WHERE metric_name = ? %s
			GROUP BY substr(hour,1,10)`, agg, fromClause)
	}

	rows, err := s.db.Query(query, metric)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var date string
		var val float64
		if rows.Scan(&date, &val) != nil {
			continue
		}
		s.db.Exec(fmt.Sprintf(`
			INSERT INTO daily_scores (date, %s, computed_at)
			VALUES (?, ?, datetime('now'))
			ON CONFLICT(date) DO UPDATE SET %s = excluded.%s`, col, col, col),
			date, val)
	}
	return rows.Err()
}

// buildMinuteMetric fills minute_metrics for one metric by reading metric_points.
// Level 1 of the cascade.
func (s *DB) buildMinuteMetric(metric, agg string, force bool) error {
	// Find the range to (re)compute: if not force, only fill missing minutes.
	var fromClause string
	if !force {
		var lastCached string
		s.db.QueryRow(
			`SELECT MAX(minute) FROM minute_metrics WHERE metric_name = ?`, metric,
		).Scan(&lastCached)
		if lastCached != "" {
			fromClause = fmt.Sprintf("AND substr(date,1,16) > '%s'", lastCached)
		}
	}

	_, err := s.db.Exec(fmt.Sprintf(`
		INSERT OR REPLACE INTO minute_metrics (metric_name, minute, source, avg_val, min_val, max_val)
		SELECT metric_name,
		       substr(date, 1, 16) AS minute,
		       source,
		       %s(qty), MIN(qty), MAX(qty)
		FROM metric_points
		WHERE metric_name = ? AND qty > 0 %s
		GROUP BY metric_name, minute, source
	`, agg, fromClause), metric)
	return err
}

// buildHourlyMetric fills hourly_metrics for one metric by reading minute_metrics.
// Level 2 of the cascade — never touches metric_points.
func (s *DB) buildHourlyMetric(metric, agg string, force bool) error {
	var fromClause string
	if !force {
		var lastCached string
		s.db.QueryRow(
			`SELECT MAX(hour) FROM hourly_metrics WHERE metric_name = ?`, metric,
		).Scan(&lastCached)
		if lastCached != "" {
			fromClause = fmt.Sprintf("AND substr(minute,1,13) > '%s'", lastCached)
		}
	}

	// Combine per-source minute values into hourly per-source values.
	// For SUM metrics: SUM the per-minute sums. For AVG: AVG the per-minute avgs.
	_, err := s.db.Exec(fmt.Sprintf(`
		INSERT OR REPLACE INTO hourly_metrics (metric_name, hour, source, avg_val, min_val, max_val)
		SELECT metric_name,
		       substr(minute, 1, 13) || ':00' AS hour,
		       source,
		       %s(avg_val), MIN(min_val), MAX(max_val)
		FROM minute_metrics
		WHERE metric_name = ? %s
		GROUP BY metric_name, hour, source
	`, agg, fromClause), metric)
	return err
}
