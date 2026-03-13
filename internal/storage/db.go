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
	return err
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
