package storage

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db *sql.DB
}

func New(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	s := &DB{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *DB) migrate() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS health_records (
		id                     INTEGER PRIMARY KEY AUTOINCREMENT,
		received_at            DATETIME DEFAULT CURRENT_TIMESTAMP,
		automation_name        TEXT,
		automation_id          TEXT,
		automation_aggregation TEXT,
		automation_period      TEXT,
		session_id             TEXT,
		content_type           TEXT,
		payload                TEXT
	)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE TABLE IF NOT EXISTS metric_points (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		health_record_id INTEGER NOT NULL REFERENCES health_records(id),
		received_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
		metric_name      TEXT NOT NULL,
		units            TEXT,
		date             TEXT NOT NULL,
		qty              REAL,
		source           TEXT,
		UNIQUE(metric_name, date, source)
	)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_metric_points_name_date ON metric_points(metric_name, date)`)
	return err
}

type Record struct {
	AutomationName        string
	AutomationID          string
	AutomationAggregation string
	AutomationPeriod      string
	SessionID             string
	ContentType           string
	Payload               string
}

type MetricPoint struct {
	MetricName string
	Units      string
	Date       string
	Qty        float64
	Source     string
}

func (s *DB) Insert(r Record, points []MetricPoint) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO health_records
		(automation_name, automation_id, automation_aggregation, automation_period, session_id, content_type, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.AutomationName, r.AutomationID, r.AutomationAggregation,
		r.AutomationPeriod, r.SessionID, r.ContentType, r.Payload,
	)
	if err != nil {
		return 0, err
	}
	recordID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO metric_points
		(health_record_id, metric_name, units, date, qty, source)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, p := range points {
		if _, err := stmt.Exec(recordID, p.MetricName, p.Units, p.Date, p.Qty, p.Source); err != nil {
			return 0, fmt.Errorf("insert point %s/%s: %w", p.MetricName, p.Date, err)
		}
	}

	return recordID, tx.Commit()
}

type MetricSummary struct {
	Name  string
	Units string
	Count int
	Min   string
	Max   string
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

type DataPoint struct {
	Date string  `json:"date"`
	Qty  float64 `json:"qty"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
}

func (s *DB) GetMetricData(metric, from, to, bucket, aggFunc string) ([]DataPoint, error) {
	var bucketExpr string
	switch bucket {
	case "hour":
		bucketExpr = "substr(date, 1, 13) || ':00'"
	case "day":
		bucketExpr = "substr(date, 1, 10)"
	default: // minute
		bucketExpr = "substr(date, 1, 16)"
	}

	if aggFunc != "SUM" && aggFunc != "MAX" && aggFunc != "MIN" {
		aggFunc = "AVG"
	}

	query := "SELECT " + bucketExpr + " as bucket, " + aggFunc + `(qty), MIN(qty), MAX(qty)
		FROM metric_points
		WHERE metric_name = ?
		  AND date >= ?
		  AND date <= ?
		  AND qty > 0
		GROUP BY bucket
		ORDER BY bucket`

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

type CardData struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Prev   float64 `json:"prev"` // previous day value for trend indicator
	Unit   string  `json:"unit"`
	Date   string  `json:"date"`
}

type DashboardResponse struct {
	Date        string     `json:"date"`
	LastUpdated string     `json:"last_updated"`
	Cards       []CardData `json:"cards"`
}

func (s *DB) GetDashboard() (*DashboardResponse, error) {
	var today string
	if err := s.db.QueryRow(`SELECT MAX(substr(date,1,10)) FROM metric_points WHERE qty > 0`).Scan(&today); err != nil || today == "" {
		return &DashboardResponse{}, nil
	}

	// Previous day with data
	var yesterday string
	s.db.QueryRow(`SELECT MAX(substr(date,1,10)) FROM metric_points WHERE qty > 0 AND substr(date,1,10) < ?`, today).Scan(&yesterday)

	// Timestamp of the most recent data receipt
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
		s.db.QueryRow(
			`SELECT `+agg+`(qty), units FROM metric_points WHERE metric_name=? AND substr(date,1,10)=? AND qty>0`,
			metric, day,
		).Scan(&val, &unit)
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

func (s *DB) Close() error {
	return s.db.Close()
}
