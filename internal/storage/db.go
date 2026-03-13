package storage

import (
	"database/sql"
	"fmt"

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
	if err != nil {
		return err
	}
	if err := s.migrateDailyScores(); err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE TABLE IF NOT EXISTS minute_metrics (
		metric_name TEXT NOT NULL,
		minute      TEXT NOT NULL,
		source      TEXT NOT NULL DEFAULT '',
		avg_val     REAL NOT NULL,
		min_val     REAL NOT NULL,
		max_val     REAL NOT NULL,
		PRIMARY KEY (metric_name, minute, source)
	)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE TABLE IF NOT EXISTS hourly_metrics (
		metric_name TEXT NOT NULL,
		hour        TEXT NOT NULL,
		source      TEXT NOT NULL DEFAULT '',
		avg_val     REAL NOT NULL,
		min_val     REAL NOT NULL,
		max_val     REAL NOT NULL,
		PRIMARY KEY (metric_name, hour, source)
	)`)
	return err
}

// migrateDailyScores creates or upgrades the daily_scores table.
// v1 had NOT NULL on readiness; v2 adds 13 metric columns and makes all
// computed columns nullable so metric and readiness fills can happen
// independently. Safe to call on every startup.
func (s *DB) migrateDailyScores() error {
	// Check whether the table needs creating or upgrading.
	var hasMetrics int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('daily_scores') WHERE name='hrv_avg'`).Scan(&hasMetrics)

	if hasMetrics == 0 {
		// Create new table or upgrade existing one via rename-copy-drop.
		s.db.Exec(`CREATE TABLE IF NOT EXISTS daily_scores_new (
			date          TEXT NOT NULL PRIMARY KEY,
			readiness     INTEGER,
			score_version INTEGER,
			computed_at   TEXT NOT NULL DEFAULT (datetime('now')),
			hrv_avg       REAL,
			rhr_avg       REAL,
			sleep_total   REAL,
			sleep_deep    REAL,
			sleep_rem     REAL,
			sleep_core    REAL,
			sleep_awake   REAL,
			steps         REAL,
			calories      REAL,
			exercise_min  REAL,
			spo2_avg      REAL,
			vo2_avg       REAL,
			resp_avg      REAL
		)`)
		// Copy any existing rows (readiness scores survive the upgrade).
		s.db.Exec(`INSERT OR IGNORE INTO daily_scores_new (date, readiness, score_version, computed_at)
			SELECT date, readiness, score_version, computed_at FROM daily_scores`)
		s.db.Exec(`DROP TABLE IF EXISTS daily_scores`)
		if _, err := s.db.Exec(`ALTER TABLE daily_scores_new RENAME TO daily_scores`); err != nil {
			return fmt.Errorf("migrate daily_scores: %w", err)
		}
	}
	return nil
}

func (s *DB) Close() error {
	return s.db.Close()
}

// Record is the raw payload received from Health Auto Export.
type Record struct {
	AutomationName        string
	AutomationID          string
	AutomationAggregation string
	AutomationPeriod      string
	SessionID             string
	ContentType           string
	Payload               string
}

// MetricPoint is a single parsed data point stored in metric_points.
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
