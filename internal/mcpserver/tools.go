package mcpserver

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"health-receiver/internal/storage"
)

func registerMetricTools(s *server.MCPServer, db *storage.DB) {
	s.AddTool(mcp.NewTool("list_metrics",
		mcp.WithDescription("List all available health metrics with record counts and date ranges. Call this first to discover what data is available."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metrics, err := db.ListMetrics()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(metrics)
	})

	s.AddTool(mcp.NewTool("get_dashboard",
		mcp.WithDescription("Get today's health summary: steps, calories, heart rate, SpO2, HRV, sleep and more. Includes trend vs yesterday."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cards, err := db.GetDashboard()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(cards)
	})

	s.AddTool(mcp.NewTool("get_metric_data",
		mcp.WithDescription("Get time series data for a single health metric. Use bucket='day' for trends, 'hour' for intraday patterns, 'minute' for raw resolution."),
		mcp.WithString("metric", mcp.Required(), mcp.Description("Metric name, e.g. heart_rate, step_count, sleep_total")),
		mcp.WithString("from", mcp.Required(), mcp.Description("Start date YYYY-MM-DD")),
		mcp.WithString("to", mcp.Required(), mcp.Description("End date YYYY-MM-DD")),
		mcp.WithString("bucket", mcp.Description("minute, hour, day (default: day)")),
		mcp.WithString("agg", mcp.Description("AVG, SUM, MAX, MIN (default: auto based on metric type)")),
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
		return jsonResult(map[string]any{"metric": metric, "bucket": bucket, "agg": aggFunc, "count": len(points), "points": points})
	})

	s.AddTool(mcp.NewTool("summarize_metric",
		mcp.WithDescription("Statistical summary for a metric over recent days: avg, min, max, count + daily breakdown. Good for quick trend analysis."),
		mcp.WithString("metric", mcp.Required(), mcp.Description("Metric name, e.g. heart_rate, step_count")),
		mcp.WithNumber("days", mcp.Description("Number of recent days (default: 7)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metric := req.GetString("metric", "")
		days := req.GetInt("days", 7)
		stats, err := db.SummarizeMetric(metric, days)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(stats)
	})

	s.AddTool(mcp.NewTool("sql_query",
		mcp.WithDescription(`Run a read-only SQL SELECT on the health database. Use when other tools don't cover your query.

Schema:
  metric_points(
    id               INTEGER,
    health_record_id INTEGER,   -- FK to health_records.id
    metric_name      TEXT,      -- e.g. 'heart_rate', 'step_count', 'sleep_total'
    units            TEXT,      -- e.g. 'count/min', 'count', 'kcal', 'hr'
    date             TEXT,      -- "YYYY-MM-DD HH:MM:SS +HHMM" — use substr(date,1,10) for day
    qty              REAL,      -- the measured value
    source           TEXT       -- data source app/device
  )
  health_records(
    id               INTEGER,
    received_at      DATETIME,  -- when the server received the payload
    automation_name  TEXT,
    session_id       TEXT
  )

Key metrics: heart_rate, resting_heart_rate, heart_rate_variability, blood_oxygen_saturation,
  step_count, active_energy, basal_energy_burned, walking_running_distance, apple_exercise_time,
  sleep_total, sleep_deep, sleep_rem, sleep_core, sleep_awake,
  respiratory_rate, apple_sleeping_wrist_temperature, vo2_max

Notes:
  - Always filter qty > 0 to exclude zero-value placeholders
  - Date comparison: use substr(date,1,10) >= '2026-01-01' (NOT strftime — TZ offset breaks it)
  - SUM metrics: step_count, active_energy, basal_energy_burned, apple_exercise_time,
      apple_stand_time, flights_climbed, walking_running_distance, time_in_daylight
  - All others use AVG`),
		mcp.WithString("query", mcp.Required(), mcp.Description("SQL SELECT query")),
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
}

func registerAnalysisTools(s *server.MCPServer, db *storage.DB) {
	s.AddTool(mcp.NewTool("compare_periods",
		mcp.WithDescription("Compare a metric between two date ranges. Useful for before/after analysis or week-over-week comparisons."),
		mcp.WithString("metric", mcp.Required(), mcp.Description("Metric name, e.g. heart_rate, step_count")),
		mcp.WithString("period_a_from", mcp.Required(), mcp.Description("Period A start date YYYY-MM-DD")),
		mcp.WithString("period_a_to", mcp.Required(), mcp.Description("Period A end date YYYY-MM-DD")),
		mcp.WithString("period_b_from", mcp.Required(), mcp.Description("Period B start date YYYY-MM-DD")),
		mcp.WithString("period_b_to", mcp.Required(), mcp.Description("Period B end date YYYY-MM-DD")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metric := req.GetString("metric", "")
		aFromStr := req.GetString("period_a_from", "")
		aToStr := req.GetString("period_a_to", "")
		bFromStr := req.GetString("period_b_from", "")
		bToStr := req.GetString("period_b_to", "")

		agg := "AVG"
		if sumMetrics[metric] {
			agg = "SUM"
		}

		type periodStats struct {
			From       string  `json:"from"`
			To         string  `json:"to"`
			Value      float64 `json:"value"`
			Min        float64 `json:"min"`
			Max        float64 `json:"max"`
			DataPoints int     `json:"data_points"`
		}

		queryPeriod := func(from, to string) (periodStats, error) {
			rows, err := db.QueryReadOnly(fmt.Sprintf(
				`SELECT %s(qty), MIN(qty), MAX(qty), COUNT(*) FROM metric_points
				 WHERE metric_name = '%s' AND substr(date,1,10) >= '%s' AND substr(date,1,10) <= '%s' AND qty > 0`,
				agg, metric, from, to,
			))
			if err != nil || len(rows) == 0 {
				return periodStats{From: from, To: to}, err
			}
			r := rows[0]
			ps := periodStats{From: from, To: to}
			if v, ok := r[agg+"(qty)"].(float64); ok {
				ps.Value = v
			}
			if v, ok := r["MIN(qty)"].(float64); ok {
				ps.Min = v
			}
			if v, ok := r["MAX(qty)"].(float64); ok {
				ps.Max = v
			}
			if v, ok := r["COUNT(*)"].(int64); ok {
				ps.DataPoints = int(v)
			}
			return ps, nil
		}

		a, err := queryPeriod(aFromStr, aToStr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, err := queryPeriod(bFromStr, bToStr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var changePct *float64
		if a.Value != 0 {
			pct := (b.Value - a.Value) / a.Value * 100
			changePct = &pct
		}

		return jsonResult(map[string]any{
			"metric": metric, "agg": agg,
			"period_a": a, "period_b": b, "change_pct": changePct,
		})
	})

	s.AddTool(mcp.NewTool("get_sleep_summary",
		mcp.WithDescription("Get sleep breakdown by phase (deep, REM, core, awake, total) per night for a date range. Values are deduplicated across devices (e.g. Apple Watch + RingConn)."),
		mcp.WithString("from", mcp.Required(), mcp.Description("Start date YYYY-MM-DD")),
		mcp.WithString("to", mcp.Required(), mcp.Description("End date YYYY-MM-DD")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		from := req.GetString("from", "")
		to := req.GetString("to", "")
		nights, err := db.GetSleepSummary(from, to)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(map[string]any{"from": from, "to": to, "nights": nights})
	})

	s.AddTool(mcp.NewTool("find_anomalies",
		mcp.WithDescription("Find days where a metric was statistically unusual (beyond mean ± sigma threshold). Useful for spotting illness, overtraining, or unusually good days."),
		mcp.WithString("metric", mcp.Required(), mcp.Description("Metric name, e.g. heart_rate, hrv, step_count")),
		mcp.WithString("from", mcp.Required(), mcp.Description("Start date YYYY-MM-DD")),
		mcp.WithString("to", mcp.Required(), mcp.Description("End date YYYY-MM-DD")),
		mcp.WithNumber("sigma", mcp.Description("Threshold in standard deviations (default: 2.0)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metric := req.GetString("metric", "")
		from := req.GetString("from", "")
		to := req.GetString("to", "")
		sigma := req.GetFloat("sigma", 2.0)

		agg := "AVG"
		if sumMetrics[metric] {
			agg = "SUM"
		}
		points, err := db.GetMetricData(metric, from, to+" 23:59:59", "day", agg)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if len(points) < 3 {
			return mcp.NewToolResultText("not enough data points to detect anomalies"), nil
		}

		var sum float64
		for _, p := range points {
			sum += p.Qty
		}
		mean := sum / float64(len(points))

		var variance float64
		for _, p := range points {
			d := p.Qty - mean
			variance += d * d
		}
		stddev := math.Sqrt(variance / float64(len(points)))

		type anomaly struct {
			Date   string  `json:"date"`
			Value  float64 `json:"value"`
			ZScore float64 `json:"z_score"`
			Type   string  `json:"type"`
		}
		var anomalies []anomaly
		for _, p := range points {
			z := (p.Qty - mean) / stddev
			if math.Abs(z) >= sigma {
				t := "high"
				if z < 0 {
					t = "low"
				}
				anomalies = append(anomalies, anomaly{Date: p.Date, Value: p.Qty, ZScore: math.Round(z*100) / 100, Type: t})
			}
		}

		return jsonResult(map[string]any{
			"metric": metric, "from": from, "to": to,
			"mean": math.Round(mean*100) / 100, "stddev": math.Round(stddev*100) / 100,
			"sigma": sigma, "anomalies": anomalies,
		})
	})

	s.AddTool(mcp.NewTool("get_weekly_summary",
		mcp.WithDescription("Weekly aggregates for one or more metrics. Good for spotting week-over-week trends."),
		mcp.WithString("metrics", mcp.Required(), mcp.Description("Comma-separated metric names, e.g. 'step_count,heart_rate,sleep_total'")),
		mcp.WithString("from", mcp.Required(), mcp.Description("Start date YYYY-MM-DD")),
		mcp.WithString("to", mcp.Required(), mcp.Description("End date YYYY-MM-DD")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metricsStr := req.GetString("metrics", "")
		from := req.GetString("from", "")
		to := req.GetString("to", "")

		metricList := strings.Split(metricsStr, ",")
		result := map[string]any{}

		for _, metric := range metricList {
			metric = strings.TrimSpace(metric)
			if metric == "" {
				continue
			}
			agg := "AVG"
			if sumMetrics[metric] {
				agg = "SUM"
			}
			rows, err := db.QueryReadOnly(fmt.Sprintf(
				`SELECT strftime('%%Y-W%%W', substr(date,1,10)) as week,
				        %s(qty) as value, COUNT(*) as data_points
				 FROM metric_points
				 WHERE metric_name = '%s' AND substr(date,1,10) >= '%s' AND substr(date,1,10) <= '%s' AND qty > 0
				 GROUP BY week ORDER BY week`,
				agg, metric, from, to,
			))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("%s: %v", metric, err)), nil
			}
			result[metric] = map[string]any{"agg": agg, "weeks": rows}
		}
		return jsonResult(map[string]any{"from": from, "to": to, "data": result})
	})

	s.AddTool(mcp.NewTool("get_personal_records",
		mcp.WithDescription("Best (max) and worst (min) values ever recorded for each metric, with the date they occurred."),
		mcp.WithString("metric", mcp.Description("Filter to a single metric. If omitted, returns records for all metrics.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		metric := req.GetString("metric", "")
		whereMetric := ""
		if metric != "" {
			whereMetric = fmt.Sprintf("AND m.metric_name = '%s'", metric)
		}
		rows, err := db.QueryReadOnly(fmt.Sprintf(
			`SELECT m.metric_name, m.units,
			        mx.max_qty as best_value,  substr(mxd.date,1,10) as best_date,
			        mn.min_qty as worst_value, substr(mnd.date,1,10) as worst_date
			 FROM (SELECT DISTINCT metric_name FROM metric_points WHERE qty > 0) m
			 JOIN (SELECT metric_name, MAX(qty) as max_qty FROM metric_points WHERE qty > 0 GROUP BY metric_name) mx
			   ON m.metric_name = mx.metric_name
			 JOIN (SELECT metric_name, MIN(qty) as min_qty FROM metric_points WHERE qty > 0 GROUP BY metric_name) mn
			   ON m.metric_name = mn.metric_name
			 JOIN metric_points mxd ON mxd.metric_name = m.metric_name AND mxd.qty = mx.max_qty
			 JOIN metric_points mnd ON mnd.metric_name = m.metric_name AND mnd.qty = mn.min_qty
			 WHERE 1=1 %s
			 GROUP BY m.metric_name
			 ORDER BY m.metric_name`, whereMetric,
		))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(rows)
	})
}
