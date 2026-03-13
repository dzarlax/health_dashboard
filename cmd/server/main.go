package main

import (
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"health-receiver/internal/handler"
	"health-receiver/internal/mcpserver"
	"health-receiver/internal/storage"
	"health-receiver/internal/ui"
)

func main() {
	dbPath := getEnv("DB_PATH", "/app/data/health.db")
	addr := getEnv("ADDR", ":8080")
	apiKey     := os.Getenv("API_KEY")
	uiPassword := os.Getenv("UI_PASSWORD")
	baseURL    := getEnv("BASE_URL", "http://localhost"+addr)

	db, err := storage.New(dbPath)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer db.Close()

	// Start the backfill scheduler. It runs an initial incremental backfill
	// shortly after startup and then re-runs whenever new data arrives,
	// debouncing rapid successive syncs into a single pass.
	sched := newBackfillScheduler(db, 2*time.Minute)
	sched.scheduleAfter(10 * time.Second) // warm caches soon after startup

	onNewData := func() {
		db.InvalidateRecentAggregates(6)
		db.InvalidateRecentScores(3)
		sched.schedule() // debounced: triggers backfill 2 min after last sync
	}

	// forceRunning prevents concurrent force-rebuild runs.
	var forceRunning int32
	backfillFn := func(force bool) {
		if !force {
			sched.schedule()
			return
		}
		if !atomic.CompareAndSwapInt32(&forceRunning, 0, 1) {
			log.Println("force backfill already running, skipping")
			return
		}
		go func() {
			defer atomic.StoreInt32(&forceRunning, 0)
			log.Println("force backfill: starting full rebuild…")
			db.BackfillAggregates(true)
			db.BackfillScores(true)
			log.Println("force backfill: done")
		}()
	}

	mux := http.NewServeMux()
	handler.New(db, apiKey, onNewData).Register(mux)
	ui.New(db, uiPassword, backfillFn).Register(mux)
	mcpserver.Register(mux, db, baseURL, apiKey)

	logged := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		mux.ServeHTTP(w, r)
	})

	log.Printf("listening on %s", addr)
	log.Printf("MCP endpoint: %s/mcp", baseURL)
	if err := http.ListenAndServe(addr, logged); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// backfillScheduler debounces backfill triggers so that multiple POST /health
// requests within `delay` collapse into a single backfill run.
type backfillScheduler struct {
	db      *storage.DB
	delay   time.Duration
	trigger chan struct{}
}

func newBackfillScheduler(db *storage.DB, delay time.Duration) *backfillScheduler {
	s := &backfillScheduler{
		db:      db,
		delay:   delay,
		trigger: make(chan struct{}, 1),
	}
	go s.run()
	return s
}

// schedule queues a backfill. If one is already queued, this is a no-op.
func (s *backfillScheduler) schedule() {
	select {
	case s.trigger <- struct{}{}:
	default: // already queued
	}
}

// scheduleAfter queues a backfill to start after the given duration.
func (s *backfillScheduler) scheduleAfter(d time.Duration) {
	go func() {
		time.Sleep(d)
		s.schedule()
	}()
}

func (s *backfillScheduler) run() {
	for range s.trigger {
		time.Sleep(s.delay)
		// Drain any additional triggers that arrived during the delay.
		for len(s.trigger) > 0 {
			<-s.trigger
		}
		log.Println("scheduler: running incremental backfill…")
		s.db.RunIncrementalBackfill()
		log.Println("scheduler: done")
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
