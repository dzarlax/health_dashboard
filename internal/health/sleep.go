package health

import (
	"fmt"
	"math"
)

func scoreSleep(d RawMetrics, ls LangStrings) *BriefingSection {
	if len(d.Sleep) == 0 {
		return nil
	}
	sec := &BriefingSection{Key: "sleep", Title: ls["sec_sleep"], Icon: "moon"}
	recent := avg(d.Sleep[:min(3, len(d.Sleep))])
	baseline := avg(d.Sleep)
	pct := pctChange(recent, baseline)

	score := 0
	if recent >= 7 {
		score += 3
	} else if recent >= 6 {
		score += 2
	} else if recent >= 5 {
		score += 1
	}

	deepPct := 0.0
	if len(d.Deep) >= 3 && recent > 0 {
		recentDeep := avg(d.Deep[:min(3, len(d.Deep))])
		deepPct = recentDeep / recent * 100
		if deepPct >= 15 {
			score += 2
		} else if deepPct >= 10 {
			score += 1
		}
	}

	if len(d.Awake) >= 3 {
		recentAwake := avg(d.Awake[:min(3, len(d.Awake))])
		if recentAwake < 0.5 {
			score++
		}
	}

	// Sleep regularity: stddev of nightly duration over available window.
	// Low variability (≤0.5h SD) earns +1 point.
	// Motivated by PMC10782501 (2024): sleep regularity predicts all-cause
	// mortality more strongly than mean duration.
	sleepSD := 0.0
	if len(d.Sleep) >= 7 {
		sleepSD = stddev(d.Sleep)
		if sleepSD <= 0.5 {
			score++
		}
	}

	if score >= 5 {
		sec.Status = "good"
		sec.Summary = fmt.Sprintf(ls["sleep_summary_good"], recent)
	} else if score >= 3 {
		sec.Status = "fair"
		sec.Summary = fmt.Sprintf(ls["sleep_summary_fair"], recent)
	} else {
		sec.Status = "low"
		sec.Summary = fmt.Sprintf(ls["sleep_summary_low"], recent)
	}

	t := trend(pct, false)
	durationNote := ls["sleep_dur_stable"]
	if pct > 5 {
		durationNote = ls["sleep_dur_more"]
	} else if pct < -5 {
		durationNote = ls["sleep_dur_less"]
	}
	sec.Details = append(sec.Details, BriefingDetail{
		Label: ls["lbl_duration"], Value: fmt.Sprintf(ls["unit_hrs_night"], recent), Note: durationNote, Trend: t,
	})

	if deepPct > 0 {
		dNote := ls["sleep_deep_good"]
		if deepPct < 15 {
			dNote = ls["sleep_deep_low"]
		}
		deepTrend := "down"
		if deepPct >= 15 {
			deepTrend = "up"
		}
		sec.Details = append(sec.Details, BriefingDetail{
			Label: ls["lbl_deep_sleep"], Value: fmt.Sprintf(ls["unit_pct_total"], deepPct),
			Note: dNote, Trend: deepTrend,
		})
	}

	if len(d.REM) >= 3 && recent > 0 {
		recentRem := avg(d.REM[:min(3, len(d.REM))])
		remPct := recentRem / recent * 100
		rNote := ls["sleep_rem_good"]
		if remPct < 20 {
			rNote = ls["sleep_rem_low"]
		}
		remTrend := "stable"
		if remPct >= 20 {
			remTrend = "up"
		}
		sec.Details = append(sec.Details, BriefingDetail{
			Label: ls["lbl_rem"], Value: fmt.Sprintf(ls["unit_pct_total"], remPct),
			Note: rNote, Trend: remTrend,
		})
	}

	if len(d.Sleep) >= 7 {
		regNote := ls["sleep_reg_moderate"]
		regTrend := "stable"
		if sleepSD <= 0.5 {
			regNote = ls["sleep_reg_regular"]
			regTrend = "up"
		} else if sleepSD > 1.0 {
			regNote = ls["sleep_reg_irregular"]
			regTrend = "down"
		}
		sec.Details = append(sec.Details, BriefingDetail{
			Label: ls["lbl_sleep_regularity"], Value: fmt.Sprintf("±%.1fh", sleepSD),
			Note: regNote, Trend: regTrend,
		})
	}
	return sec
}

func computeSleepAnalysis(d RawMetrics) *SleepAnalysis {
	if len(d.Sleep) < 3 {
		return nil
	}
	sa := SleepAnalysis{}
	recentSleep := avg(d.Sleep[:min(3, len(d.Sleep))])
	if len(d.Deep) >= 3 {
		sa.DeepAvg = math.Round(avg(d.Deep[:min(3, len(d.Deep))])*100) / 100
	}
	if len(d.REM) >= 3 {
		sa.REMAvg = math.Round(avg(d.REM[:min(3, len(d.REM))])*100) / 100
	}
	if len(d.Awake) >= 3 {
		sa.AwakeAvg = math.Round(avg(d.Awake[:min(3, len(d.Awake))])*100) / 100
	}
	if recentSleep > 0 {
		sa.Efficiency = math.Round((recentSleep-sa.AwakeAvg)/recentSleep*100*10) / 10
		if sa.Efficiency < 0 {
			sa.Efficiency = 0
		}
	}
	return &sa
}
