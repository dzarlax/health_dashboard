package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"health-receiver/internal/storage"
)

var sumMetrics = map[string]bool{
	"step_count": true, "active_energy": true, "basal_energy_burned": true,
	"apple_exercise_time": true, "apple_stand_time": true,
	"flights_climbed": true, "walking_running_distance": true,
	"time_in_daylight": true, "apple_stand_hour": true,
}

func main() {
	dbPath := getEnv("DB_PATH", "./data/health.db")
	db, err := storage.New(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	s := server.NewMCPServer("health-mcp", "1.0.0",
		server.WithToolCapabilities(true),
	)

	// ── list_metrics ────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("list_metrics",
		mcp.WithDescription("List all available health metrics with record counts and date ranges. Call this first to know what data is available."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metrics, err := db.ListMetrics()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(metrics)
	})

	// ── get_dashboard ────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("get_dashboard",
		mcp.WithDescription("Get today's health summary: steps, calories, heart rate, SpO2, HRV, sleep and more."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cards, err := db.GetDashboard()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(cards)
	})

	// ── get_metric_data ──────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("get_metric_data",
		mcp.WithDescription("Get time series data for a health metric. Returns aggregated data points. Use bucket='day' for trends, 'hour' for intraday patterns."),
		mcp.WithString("metric",
			mcp.Required(),
			mcp.Description("Metric name, e.g. heart_rate, step_count, blood_oxygen_saturation"),
		),
		mcp.WithString("from",
			mcp.Required(),
			mcp.Description("Start date in YYYY-MM-DD format"),
		),
		mcp.WithString("to",
			mcp.Required(),
			mcp.Description("End date in YYYY-MM-DD format"),
		),
		mcp.WithString("bucket",
			mcp.Description("Time bucket: minute, hour, day (default: day)"),
		),
		mcp.WithString("agg",
			mcp.Description("Aggregation: AVG, SUM, MAX, MIN (default: auto based on metric type)"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metric := req.GetString("metric", "")
		from := req.GetString("from", "")
		to := req.GetString("to", "")
		bucket := req.GetString("bucket", "day")
		aggFunc := req.GetString("agg", "")

		if aggFunc == "" {
			if sumMetrics[metric] {
				aggFunc = "SUM"
			} else {
				aggFunc = "AVG"
			}
		}

		points, err := db.GetMetricData(metric, from, to+" 23:59:59", bucket, aggFunc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(map[string]any{
			"metric": metric,
			"bucket": bucket,
			"agg":    aggFunc,
			"count":  len(points),
			"points": points,
		})
	})

	// ── summarize_metric ─────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("summarize_metric",
		mcp.WithDescription("Get statistical summary for a metric over recent days: avg, min, max, count, plus daily breakdown. Good for trend analysis."),
		mcp.WithString("metric",
			mcp.Required(),
			mcp.Description("Metric name, e.g. heart_rate, step_count"),
		),
		mcp.WithNumber("days",
			mcp.Description("Number of recent days to analyse (default: 7)"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metric := req.GetString("metric", "")
		days := req.GetInt("days", 7)
		stats, err := db.SummarizeMetric(metric, days)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(stats)
	})

	// ── sql_query ────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("sql_query",
		mcp.WithDescription(`Run a read-only SQL SELECT on the health database.
Tables:
  metric_points(id, health_record_id, received_at, metric_name, units, date, qty, source)
  health_records(id, received_at, automation_name, session_id, payload)
Dates are stored as text "YYYY-MM-DD HH:MM:SS +HHMM". Use substr(date,1,10) to get the date part.
Only SELECT statements are allowed.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("SQL SELECT query"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		if !strings.HasPrefix(strings.TrimSpace(strings.ToUpper(query)), "SELECT") {
			return mcp.NewToolResultError("only SELECT queries are allowed"), nil
		}
		rows, err := db.QueryReadOnly(query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
		}
		return jsonResult(rows)
	})

	log.SetOutput(os.Stderr) // MCP uses stdout for protocol
	if err := server.NewStdioServer(s).Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatalf("mcp server: %v", err)
	}
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
