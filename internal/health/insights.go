package health

import (
	"fmt"
	"math"
)

func buildCorrelation(d RawMetrics) []CorrelationPoint {
	maxSteps := 1.0
	for _, s := range d.StepsWithDates {
		if s.Val > maxSteps {
			maxSteps = s.Val
		}
	}
	hrvByDate := make(map[string]float64)
	for _, h := range d.HRVWithDates {
		hrvByDate[h.Date] = h.Val
	}
	var out []CorrelationPoint
	for _, s := range d.StepsWithDates {
		if h, ok := hrvByDate[s.Date]; ok {
			out = append(out, CorrelationPoint{
				Date: s.Date,
				Load: math.Round(s.Val/maxSteps*100*10) / 10,
				HRV:  math.Round(h*10) / 10,
			})
		}
	}
	return out
}

func computeInsights(d RawMetrics, activitySec *BriefingSection, readinessScore int, ls LangStrings) []Insight {
	var insights []Insight

	if len(d.Steps) >= 7 {
		stepsAvg := avg(d.Steps)
		aboveCount := 0
		checkDays := min(7, len(d.Steps))
		for i := 0; i < checkDays; i++ {
			if d.Steps[i] >= stepsAvg {
				aboveCount++
			}
		}
		if aboveCount >= 5 {
			insights = append(insights, Insight{
				Text: fmt.Sprintf(ls["insight_steps_good"], aboveCount), Type: "positive",
			})
		} else {
			insights = append(insights, Insight{
				Text: fmt.Sprintf(ls["insight_steps_low"], aboveCount), Type: "warning",
			})
		}
	}

	if len(d.Steps) >= 3 && len(d.HRV) >= 3 {
		stepsAvg := avg(d.Steps)
		highActivityLowHRV := 0
		highActivityDays := 0
		checkLen := min(len(d.Steps), len(d.HRV)) - 1
		for i := 0; i < checkLen; i++ {
			if d.Steps[i+1] > stepsAvg*1.2 {
				highActivityDays++
				if d.HRV[i] < avg(d.HRV)*0.95 {
					highActivityLowHRV++
				}
			}
		}
		if highActivityDays >= 2 && highActivityLowHRV > highActivityDays/2 {
			insights = append(insights, Insight{Text: ls["insight_hrv_drop"], Type: "warning"})
		} else if highActivityDays >= 2 {
			insights = append(insights, Insight{Text: ls["insight_hrv_resilient"], Type: "positive"})
		}
	}

	if len(d.Steps) >= 7 && len(d.Sleep) >= 7 {
		stepsAvg := avg(d.Steps)
		var sleepOnActive, sleepOnRest []float64
		checkLen := min(len(d.Steps), len(d.Sleep))
		for i := 0; i < checkLen; i++ {
			if d.Steps[i] > stepsAvg {
				sleepOnActive = append(sleepOnActive, d.Sleep[i])
			} else {
				sleepOnRest = append(sleepOnRest, d.Sleep[i])
			}
		}
		if len(sleepOnActive) > 0 && len(sleepOnRest) > 0 {
			activeSleepAvg := avg(sleepOnActive)
			restSleepAvg := avg(sleepOnRest)
			if activeSleepAvg > restSleepAvg+0.5 {
				insights = append(insights, Insight{
					Text: fmt.Sprintf(ls["insight_sleep_active"], activeSleepAvg, restSleepAvg), Type: "positive",
				})
			} else if restSleepAvg > activeSleepAvg+0.5 {
				insights = append(insights, Insight{
					Text: fmt.Sprintf(ls["insight_sleep_rest"], restSleepAvg, activeSleepAvg), Type: "warning",
				})
			}
		}
	}

	if activitySec != nil && activitySec.Status == "good" && readinessScore < 50 {
		insights = append(insights, Insight{Text: ls["insight_overtrain"], Type: "warning"})
	}

	if len(insights) > 3 {
		insights = insights[:3]
	}
	return insights
}
