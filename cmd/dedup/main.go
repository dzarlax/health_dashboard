// cmd/dedup: rebuilds metric_points with a UNIQUE constraint and removes duplicates.
// Safe: runs inside a transaction, rolls back on any error.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := getEnv("DB_PATH", "./data/health.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Count before
	var before int
	db.QueryRow("SELECT COUNT(*) FROM metric_points").Scan(&before)
	fmt.Printf("before: %d rows\n", before)

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()

	steps := []string{
		// 1. New table with UNIQUE constraint
		`CREATE TABLE metric_points_new (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			health_record_id INTEGER NOT NULL REFERENCES health_records(id),
			received_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			metric_name      TEXT NOT NULL,
			units            TEXT,
			date             TEXT NOT NULL,
			qty              REAL,
			source           TEXT,
			UNIQUE(metric_name, date, source)
		)`,

		// 2. Copy unique rows (MIN(id) keeps the earliest)
		`INSERT INTO metric_points_new
			(id, health_record_id, received_at, metric_name, units, date, qty, source)
		SELECT MIN(id), MIN(health_record_id), MIN(received_at), metric_name, units, date, AVG(qty), source
		FROM metric_points
		GROUP BY metric_name, date, source`,

		// 3. Drop old table
		`DROP TABLE metric_points`,

		// 4. Rename
		`ALTER TABLE metric_points_new RENAME TO metric_points`,

		// 5. Restore index
		`CREATE INDEX idx_metric_points_name_date ON metric_points(metric_name, date)`,
	}

	for _, s := range steps {
		if _, err := tx.Exec(s); err != nil {
			log.Fatalf("step failed:\n%s\nerror: %v", s, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}

	var after int
	db.QueryRow("SELECT COUNT(*) FROM metric_points").Scan(&after)
	fmt.Printf("after:  %d rows\n", after)
	fmt.Printf("removed %d duplicates\n", before-after)
	fmt.Println("UNIQUE constraint is now active — future imports are protected.")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
