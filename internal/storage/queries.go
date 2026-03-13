package storage

import (
	"fmt"
	"strings"
)

// MetricSummary is returned by ListMetrics.
type MetricSummary struct {
	Name  string
	Units string
	Count int
	Min   string
	Max   string
}

// DataPoint is a single time-bucketed value returned by metric data queries.
type DataPoint struct {
	Date string  `json:"date"`
	Qty  float64 `json:"qty"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
}

// SourceDataPoints groups DataPoints by device source.
type SourceDataPoints struct {
	Source string      `json:"source"`
	Points []DataPoint `json:"points"`
}

// CardData is a single metric card value for the dashboard.
type CardData struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Prev   float64 `json:"prev"` // previous day value for trend indicator
	Unit   string  `json:"unit"`
	Date   string  `json:"date"`
}

// DashboardResponse is returned by GetDashboard.
type DashboardResponse struct {
	Date        string     `json:"date"`
	LastUpdated string     `json:"last_updated"`
	Cards       []CardData `json:"cards"`
}

// LatestValue is the most recent value for a single metric, used by GetLatestMetricValues.
type LatestValue struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Unit   string  `json:"unit"`
	Date   string  `json:"date"`
}

// GetLatestMetricValues returns the latest non-zero daily average for every metric in the DB.
func (s *DB) GetLatestMetricValues() ([]LatestValue, error) {
	rows, err := s.db.Query(`
		WITH latest AS (
			SELECT metric_name, MAX(substr(date,1,10)) AS max_date
			FROM metric_points
			WHERE qty > 0
			GROUP BY metric_name
		)
		SELECT mp.metric_name, mp.units, l.max_date, AVG(mp.qty) AS value
		FROM metric_points mp
		JOIN latest l ON mp.metric_name = l.metric_name
			AND substr(mp.date,1,10) = l.max_date
		WHERE mp.qty > 0
		GROUP BY mp.metric_name
		ORDER BY mp.metric_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LatestValue
	for rows.Next() {
		var v LatestValue
		if err := rows.Scan(&v.Metric, &v.Unit, &v.Date, &v.Value); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// MetricStats is returned by SummarizeMetric.
type MetricStats struct {
	Metric string      `json:"metric"`
	Units  string      `json:"units"`
	From   string      `json:"from"`
	To     string      `json:"to"`
	Count  int         `json:"count"`
	Avg    float64     `json:"avg"`
	Min    float64     `json:"min"`
	Max    float64     `json:"max"`
	Daily  []DataPoint `json:"daily"`
}

func (s *DB) ListMetrics() ([]MetricSummary, error) {
	rows, err := s.db.Query(`
		SELECT metric_name, units, COUNT(*) as cnt, MIN(date), MAX(date)
		FROM metric_points
		WHERE qty > 0
		GROUP BY metric_name, units
		ORDER BY cnt DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MetricSummary
	for rows.Next() {
		var m MetricSummary
		if err := rows.Scan(&m.Name, &m.Units, &m.Count, &m.Min, &m.Max); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *DB) GetMetricData(metric, from, to, bucket, aggFunc string) ([]DataPoint, error) {
	bucketExpr := bucketExpression(bucket)

	if aggFunc != "SUM" && aggFunc != "MAX" && aggFunc != "MIN" {
		aggFunc = "AVG"
	}

	// For sleep metrics, MAX across sources prevents double-counting when two
	// devices (e.g. Apple Watch + RingConn) each independently record the same
	// night. For other SUM metrics (steps, calories), HealthKit deduplicates at
	// the sample level, so flat SUM is correct.
	var query string
	if aggFunc == "SUM" && strings.HasPrefix(metric, "sleep_") {
		query = `SELECT bucket, MAX(source_sum), MIN(source_min), MAX(source_max)
			FROM (
				SELECT ` + bucketExpr + ` AS bucket, source, SUM(qty) AS source_sum, MIN(qty) AS source_min, MAX(qty) AS source_max
				FROM metric_points
				WHERE metric_name = ? AND date >= ? AND date <= ? AND qty > 0
				GROUP BY bucket, source
			)
			GROUP BY bucket
			ORDER BY bucket`
	} else {
		query = "SELECT " + bucketExpr + " as bucket, " + aggFunc + `(qty), MIN(qty), MAX(qty)
			FROM metric_points
			WHERE metric_name = ?
			  AND date >= ?
			  AND date <= ?
			  AND qty > 0
			GROUP BY bucket
			ORDER BY bucket`
	}

	rows, err := s.db.Query(query, metric, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DataPoint
	for rows.Next() {
		var p DataPoint
		if err := rows.Scan(&p.Date, &p.Qty, &p.Min, &p.Max); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *DB) GetMetricDataBySource(metric, from, to, bucket, aggFunc string) ([]SourceDataPoints, error) {
	bucketExpr := bucketExpression(bucket)

	if aggFunc != "SUM" && aggFunc != "MAX" && aggFunc != "MIN" {
		aggFunc = "AVG"
	}

	// Normalize source: HealthKit pipe-joins contributing device names (e.g.
	// "Apple Watch|iPhone"). We take only the first component as the display
	// source so that "Watch", "Watch|Ring", "Watch|iPhone" all merge under "Watch".
	normSource := `SUBSTR(source, 1, INSTR(source || '|', '|') - 1)`
	query := "SELECT " + bucketExpr + " as bucket, " + normSource + " as src, " + aggFunc + `(qty), MIN(qty), MAX(qty)
		FROM metric_points
		WHERE metric_name = ? AND date >= ? AND date <= ? AND qty > 0
		GROUP BY bucket, src
		ORDER BY bucket, src`

	rows, err := s.db.Query(query, metric, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sourceMap := make(map[string][]DataPoint)
	var sourceOrder []string
	seen := make(map[string]bool)
	for rows.Next() {
		var bkt, src string
		var qty, mn, mx float64
		if err := rows.Scan(&bkt, &src, &qty, &mn, &mx); err != nil {
			return nil, err
		}
		if !seen[src] {
			seen[src] = true
			sourceOrder = append(sourceOrder, src)
		}
		sourceMap[src] = append(sourceMap[src], DataPoint{Date: bkt, Qty: qty, Min: mn, Max: mx})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []SourceDataPoints
	for _, src := range sourceOrder {
		result = append(result, SourceDataPoints{Source: src, Points: sourceMap[src]})
	}
	return result, nil
}

func (s *DB) GetDashboard() (*DashboardResponse, error) {
	var today string
	if err := s.db.QueryRow(`SELECT MAX(substr(date,1,10)) FROM metric_points WHERE qty > 0`).Scan(&today); err != nil || today == "" {
		return &DashboardResponse{}, nil
	}

	var yesterday string
	s.db.QueryRow(`SELECT MAX(substr(date,1,10)) FROM metric_points WHERE qty > 0 AND substr(date,1,10) < ?`, today).Scan(&yesterday)

	var lastUpdated string
	s.db.QueryRow(`SELECT MAX(received_at) FROM health_records`).Scan(&lastUpdated)

	type spec struct {
		metric string
		agg    string
	}
	cards := []spec{
		{"step_count", "SUM"},
		{"active_energy", "SUM"},
		{"basal_energy_burned", "SUM"},
		{"heart_rate", "AVG"},
		{"resting_heart_rate", "AVG"},
		{"heart_rate_variability", "AVG"},
		{"blood_oxygen_saturation", "AVG"},
		{"respiratory_rate", "AVG"},
		{"sleep_total", "SUM"},
		{"apple_exercise_time", "SUM"},
		{"walking_running_distance", "SUM"},
		{"apple_sleeping_wrist_temperature", "AVG"},
	}

	queryDay := func(metric, agg, day string) (float64, string) {
		var val float64
		var unit string
		// Sleep metrics need MAX-per-source to avoid double-counting from two devices.
		// Cumulative metrics (steps etc.) use flat SUM — HealthKit already deduplicates.
		if agg == "SUM" && strings.HasPrefix(metric, "sleep_") {
			s.db.QueryRow(`
				SELECT MAX(source_sum), units FROM (
					SELECT source, SUM(qty) AS source_sum, units
					FROM metric_points
					WHERE metric_name=? AND substr(date,1,10)=? AND qty>0
					GROUP BY source
				)`, metric, day,
			).Scan(&val, &unit)
		} else {
			s.db.QueryRow(
				`SELECT `+agg+`(qty), units FROM metric_points WHERE metric_name=? AND substr(date,1,10)=? AND qty>0`,
				metric, day,
			).Scan(&val, &unit)
		}
		return val, unit
	}

	var result []CardData
	for _, c := range cards {
		val, unit := queryDay(c.metric, c.agg, today)
		if val == 0 {
			continue
		}
		prev, _ := queryDay(c.metric, c.agg, yesterday)
		result = append(result, CardData{
			Metric: c.metric, Value: val, Prev: prev,
			Unit: unit, Date: today,
		})
	}
	return &DashboardResponse{Date: today, LastUpdated: lastUpdated, Cards: result}, nil
}

func (s *DB) SummarizeMetric(metric string, days int) (*MetricStats, error) {
	if days <= 0 {
		days = 7
	}
	from := fmt.Sprintf("date('now','-%d days')", days-1)

	var stats MetricStats
	var units string
	err := s.db.QueryRow(`
		SELECT metric_name, units, COUNT(*), AVG(qty), MIN(qty), MAX(qty),
		       MIN(date), MAX(date)
		FROM metric_points
		WHERE metric_name = ? AND date >= `+from+` AND qty > 0`,
		metric,
	).Scan(&stats.Metric, &units, &stats.Count, &stats.Avg, &stats.Min, &stats.Max, &stats.From, &stats.To)
	if err != nil {
		return nil, err
	}
	stats.Units = units

	daily, err := s.GetMetricData(metric, stats.From[:10], stats.To[:10]+" 23:59:59", "day", "AVG")
	if err == nil {
		stats.Daily = daily
	}
	return &stats, nil
}

// SleepNight holds per-night sleep phase totals, deduplicated across devices.
type SleepNight struct {
	Date  string  `json:"date"`
	Total float64 `json:"total"`
	Deep  float64 `json:"deep"`
	REM   float64 `json:"rem"`
	Core  float64 `json:"core"`
	Awake float64 `json:"awake"`
}

// GetSleepSummary returns per-night sleep breakdown for the date range.
// For each phase it picks MAX(source_sum) across devices to avoid double-counting
// when two wearables (e.g. Apple Watch + RingConn) both record the same night.
func (s *DB) GetSleepSummary(from, to string) ([]SleepNight, error) {
	rows, err := s.db.Query(`
		SELECT d,
			MAX(CASE WHEN metric_name='sleep_total' THEN source_max ELSE 0 END),
			MAX(CASE WHEN metric_name='sleep_deep'  THEN source_max ELSE 0 END),
			MAX(CASE WHEN metric_name='sleep_rem'   THEN source_max ELSE 0 END),
			MAX(CASE WHEN metric_name='sleep_core'  THEN source_max ELSE 0 END),
			MAX(CASE WHEN metric_name='sleep_awake' THEN source_max ELSE 0 END)
		FROM (
			SELECT d, metric_name, MAX(source_sum) AS source_max
			FROM (
				SELECT substr(date,1,10) AS d, metric_name,
					SUBSTR(source, 1, INSTR(source||'|','|')-1) AS src,
					SUM(qty) AS source_sum
				FROM metric_points
				WHERE metric_name IN ('sleep_total','sleep_deep','sleep_rem','sleep_core','sleep_awake')
				  AND substr(date,1,10) >= ? AND substr(date,1,10) <= ? AND qty > 0
				GROUP BY d, metric_name, src
			)
			GROUP BY d, metric_name
		)
		GROUP BY d
		ORDER BY d`,
		from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SleepNight
	for rows.Next() {
		var n SleepNight
		if err := rows.Scan(&n.Date, &n.Total, &n.Deep, &n.REM, &n.Core, &n.Awake); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// QueryReadOnly executes an arbitrary SELECT and returns results as []map[string]any.
func (s *DB) QueryReadOnly(query string) ([]map[string]any, error) {
	q := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(q, "SELECT") {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(cols))
		for i, c := range cols {
			row[c] = vals[i]
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// bucketExpression returns the SQL expression that truncates a date column
// to the requested time bucket.
func bucketExpression(bucket string) string {
	switch bucket {
	case "hour":
		return "substr(date, 1, 13) || ':00'"
	case "day":
		return "substr(date, 1, 10)"
	default: // minute
		return "substr(date, 1, 16)"
	}
}
