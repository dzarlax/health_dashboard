package health

func scoreCardio(d RawMetrics, ls LangStrings) *BriefingSection {
	if len(d.SpO2) == 0 && len(d.VO2) == 0 && len(d.Resp) == 0 {
		return nil
	}
	sec := &BriefingSection{Key: "cardio", Title: ls["sec_cardio"], Icon: "heart"}
	score, maxScore := 0, 0

	if len(d.SpO2) >= 3 {
		recent := avg(d.SpO2[:min(3, len(d.SpO2))])
		maxScore += 2
		if recent >= 95 {
			score += 2
		} else if recent >= 92 {
			score += 1
		}
		note := ls["spo2_note_good"]
		if recent < 95 {
			note = ls["spo2_note_low"]
		}
		spo2Trend := "down"
		if recent >= 95 {
			spo2Trend = "up"
		}
		sec.Details = append(sec.Details, BriefingDetail{
			Label: ls["lbl_blood_o2"], Value: fmtFloat(recent, 1) + "%",
			Note: note, Trend: spo2Trend,
		})
	}

	if len(d.VO2) >= 3 {
		recent := avg(d.VO2[:min(3, len(d.VO2))])
		baseline := avg(d.VO2)
		pct := pctChange(recent, baseline)
		maxScore += 2
		if pct > -3 {
			score += 2
		} else if pct > -8 {
			score += 1
		}
		note := ls["vo2_note_stable"]
		if pct > 3 {
			note = ls["vo2_note_good"]
		} else if pct < -5 {
			note = ls["vo2_note_decline"]
		}
		sec.Details = append(sec.Details, BriefingDetail{
			Label: ls["lbl_vo2"], Value: fmtFloat(recent, 1) + " ml/kg/min",
			Note: note, Trend: trend(pct, false),
		})
	}

	if len(d.Resp) >= 3 {
		recent := avg(d.Resp[:min(3, len(d.Resp))])
		maxScore += 2
		if recent >= 12 && recent <= 20 {
			score += 2
		} else if recent >= 10 && recent <= 24 {
			score += 1
		}
		note := ls["resp_note_normal"]
		if recent < 12 || recent > 20 {
			note = ls["resp_note_outside"]
		}
		respTrend := "down"
		if recent >= 12 && recent <= 20 {
			respTrend = "up"
		}
		sec.Details = append(sec.Details, BriefingDetail{
			Label: ls["lbl_resp"], Value: fmtFloat(recent, 1) + " br/min",
			Note: note, Trend: respTrend,
		})
	}

	if maxScore > 0 {
		ratio := float64(score) / float64(maxScore)
		if ratio >= 0.7 {
			sec.Status = "good"
			sec.Summary = ls["cardio_summary_good"]
		} else if ratio >= 0.4 {
			sec.Status = "fair"
			sec.Summary = ls["cardio_summary_fair"]
		} else {
			sec.Status = "low"
			sec.Summary = ls["cardio_summary_low"]
		}
	}
	return sec
}
