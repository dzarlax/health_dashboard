package health

import (
	"fmt"
	"math"
)

// computeAlerts generates anomaly-based health alerts. These are NOT score
// components — they flag potential issues that warrant attention.
func computeAlerts(d RawMetrics, ls LangStrings) []Alert {
	var alerts []Alert

	// Respiratory rate anomaly: >2 SD from baseline signals possible illness
	// or autonomic stress (Mishra et al. 2024, JMIR).
	if len(d.Resp) >= 9 {
		recent := avg(d.Resp[:7])
		baseline := d.Resp[7:]
		baselineAvg := avg(baseline)
		baselineSD := stddev(baseline)
		if baselineSD > 0 && math.Abs(recent-baselineAvg) > 2*baselineSD {
			alerts = append(alerts, Alert{
				Text:     ls["alert_rr_anomaly"],
				Severity: "warning",
				Metric:   "respiratory_rate",
			})
		}
	}

	// Wrist temperature anomaly: >2 SD from baseline may indicate fever,
	// inflammation, or hormonal changes (Fuller et al. 2024, Sensors).
	if len(d.WristTemp) >= 9 {
		recent := avg(d.WristTemp[:7])
		baseline := d.WristTemp[7:]
		baselineAvg := avg(baseline)
		baselineSD := stddev(baseline)
		if baselineSD > 0 && math.Abs(recent-baselineAvg) > 2*baselineSD {
			alerts = append(alerts, Alert{
				Text:     ls["alert_wrist_temp_anomaly"],
				Severity: "warning",
				Metric:   "wrist_temperature",
			})
		}
	}

	// HRV coefficient of variation: CV >15% over the 7-day window suggests
	// autonomic instability or overreaching (Plews et al. 2012, 2014).
	if len(d.HRV) >= 7 {
		recentHRV := d.HRV[:min(7, len(d.HRV))]
		recentAvg := avg(recentHRV)
		recentSD := stddev(recentHRV)
		if recentAvg > 0 {
			cv := recentSD / recentAvg * 100
			if cv > 15 {
				alerts = append(alerts, Alert{
					Text:     fmt.Sprintf(ls["alert_hrv_cv_high"], cv),
					Severity: "warning",
					Metric:   "hrv_cv",
				})
			}
		}
	}

	return alerts
}
