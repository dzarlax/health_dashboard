# Health Dashboard — Scoring & Calculation Logic

All calculations run server-side in `internal/health/scoring.go → ComputeBriefing()`.
Data is fetched by `internal/storage/briefing.go → GetHealthBriefing()`.

---

## Data window

Every metric is fetched for the **last 30 days** relative to the most recent date that has any data in the DB (not today's date — data can be stale).

- **"Recent"** = average of the last **5 days** (index 0–4, most recent first)
- **"Baseline"** = average of days **6–30** (index 5 onward) — recent days are **excluded** from the baseline to prevent dilution
- **% change** = `(recent − baseline) / baseline × 100`

Minimum data requirement: a metric needs at least **7 days** of data to contribute to the Readiness score (5 recent + at least 2 baseline days). Section cards require at least **3 days** (7 for HRV/RHR in Recovery, 7 for sleep regularity).

> **Why 5 for "recent"?** Post-exercise HRV suppression lasts up to 72 hours [[1]](#ref-1). A 5-day window covers this window reliably while reducing the impact of single-day outliers (illness, alcohol, missed recording). ACWR-based recovery models use 7-day acute windows [[2]](#ref-2); 5 days is a balanced trade-off between responsiveness and noise robustness.

> **Why 7 total for Readiness?** With fewer than 7 days, baseline ≈ recent and the ratio is meaningless. Section cards use simpler absolute or point-based logic so 3 days is sufficient.

---

## Readiness Score (0–100)

Shown as the big number in the hero card.

```
readiness = HRV_score × 0.40 + RHR_score × 0.30 + Sleep_score × 0.30
```

All three sub-scores use the same **ratio model**: `score = clamp(70 + (ratio − 1) × 150, 0, 100)`

This means:
- A perfectly average day (ratio = 1.0) → **70**
- ~10% above baseline (ratio = 1.10) → **85**
- ~20% above baseline (ratio = 1.20) → **100**
- ~15% below baseline (ratio = 0.85) → **47**
- ~33% below baseline (ratio = 0.67) → **~5**

A score of 100 is genuinely exceptional, not the default.

> **Why HRV 40% / RHR 30% / Sleep 30%?** WHOOP's published methodology states HRV should be the single biggest recovery factor [[3]](#ref-3). Marco Altini's HRV4Training model uses HRV at roughly double the weight of RHR [[4]](#ref-4), implying a ~2:1 HRV:RHR ratio — matching this codebase's 40:30 ratio. Oura analysis suggests RHR explains ~29% of readiness variance [[5]](#ref-5), consistent with the 30% weight.

### HRV sub-score (0–100)

`ratio = recent_HRV / baseline_HRV`

Applied directly to the ratio model (higher HRV = better).

> Using personal baseline rather than population norms is the recommended approach for HRV interpretation: individual longitudinal changes are more meaningful than cross-sectional comparisons [[6]](#ref-6) [[7]](#ref-7).

### Resting HR sub-score (0–100)

`ratio = baseline_RHR / recent_RHR`  ← inverted because lower RHR is better

Applied to the ratio model.

### Sleep sub-score (0–100)

Blends two components equally (50/50):

**Absolute component** — based on recent average sleep duration [[8]](#ref-8) [[9]](#ref-9):

| Recent avg sleep | Score |
|---|---|
| ≥ 8.0 h | 95 |
| ≥ 7.5 h | 85 |
| ≥ 7.0 h | 75 |
| ≥ 6.5 h | 60 |
| ≥ 6.0 h | 45 |
| ≥ 5.5 h | 30 |
| < 5.5 h | 15 |

**Relative component** — `ratio = recent_sleep / baseline_sleep`, applied to the ratio model.

`sleep_score = abs_score × 0.5 + rel_score × 0.5`

### Readiness labels

| Score | Label | Tip |
|---|---|---|
| ≥ 80 | Optimal | Great day for a challenging workout |
| ≥ 50 | Fair | Moderate activity is fine; listen to your body |
| < 50 | Low | Focus on recovery, avoid intense exercise |

### Recovery % bar
`recovery_pct = readiness_score` (same value, displayed as a progress bar)

---

## Readiness History

Available at `GET /api/readiness-history?days=N` (default 30, max 365).

For each output day D, the score is computed using a **sliding 30-day window**: HRV, RHR, and sleep_total values for D and the 29 days before D, sorted most-recent-first (D is index 0). The same `computeReadiness` formula applies.

**Important**: the `valsBefore` helper in `briefing.go` sorts days by **date** descending, so index 0 is always the most recent calendar day — not the highest-value day. An earlier bug sorted by value, which inflated "recent" averages.

---

## Section cards (Recovery / Sleep / Activity / Heart & Lungs)

Each section has its own point-based score that maps to **good / fair / low**.
Baseline for section cards = average of **days 6–30** (same separation as Readiness).

### Recovery section

Inputs: HRV and Resting HR (≥ 7 days required per metric).

#### HRV threshold — dynamic ±1 SD

Instead of a fixed ±5% threshold, the HRV threshold is computed from the **standard deviation of the baseline window** (days 6+):

```
thresholdPct = stddev(HRV_baseline) / mean(HRV_baseline) × 100
```

Clamped to [3%, 15%] to handle sparse or very stable data.

| Condition | Points |
|---|---|
| HRV recent > baseline + thresholdPct | +2 |
| HRV recent within ±thresholdPct of baseline | +1 |
| HRV recent < baseline − thresholdPct | +0 |
| RHR recent < baseline − 2% | +2 |
| RHR recent within −2% to +3% of baseline | +1 |
| RHR recent > baseline + 3% | +0 |

Max possible: 4 points (if both metrics available).

| Ratio (score / max) | Status |
|---|---|
| ≥ 0.75 | good |
| ≥ 0.40 | fair |
| < 0.40 | low |

> **Why dynamic SD threshold?** A fixed 5% threshold ignores individual HRV variability. Someone with a coefficient of variation (CV) of 2% would trigger "below baseline" on normal fluctuation noise; someone with CV 20% would rarely trigger it even when meaningfully depleted. A ±1 SD threshold (as a % of baseline) adapts to each person's natural day-to-day variability, giving fewer false positives for high-CV individuals and fewer missed detections for low-CV individuals [[4]](#ref-4) [[6]](#ref-6).

**Trend arrows on detail rows:**
- HRV: up if > threshold, down if < −threshold, otherwise stable
- RHR: inverted — up (good) if < −3%, down (bad) if > +3%

---

### Sleep section

Inputs: sleep_total, sleep_deep, sleep_rem, sleep_awake (≥ 3 days required).
Baseline here is `avg(all available days)` — the sleep section uses a simple
duration-based point scale, not the ratio model.

**Duration score** [[8]](#ref-8):

| Recent avg sleep | Points |
|---|---|
| ≥ 7 h | +3 |
| ≥ 6 h | +2 |
| ≥ 5 h | +1 |
| < 5 h | +0 |

**Deep sleep score** (% of total sleep):

| Deep % | Points |
|---|---|
| ≥ 15% | +2 |
| ≥ 10% | +1 |
| < 10% | +0 |

**Awake time bonus:**

| Recent avg awake | Points |
|---|---|
| < 0.5 h | +1 |

**Sleep regularity bonus** (≥ 7 days required) [[10]](#ref-10):

| stddev(sleep_duration over window) | Points |
|---|---|
| ≤ 0.5 h | +1 |
| > 0.5 h | +0 |

Max possible: 7 points.

| Total points | Status |
|---|---|
| ≥ 5 | good |
| ≥ 3 | fair |
| < 3 | low |

> **Why sleep regularity?** A 2024 meta-analysis (PMC10782501, n ≈ 60 000) found the Sleep Regularity Index predicts all-cause and cardiovascular mortality more strongly than mean sleep duration (HR 0.97 vs 0.99 per SD improvement) [[10]](#ref-10). Irregular sleep schedules disrupt circadian rhythm independent of total duration.

**REM display** (not scored, shown for info):
- Ideal: ≥ 20% of total sleep
- Formula: `recent_rem / recent_total × 100`

**Consistency display** (shown when ≥ 7 days of data):
- Value: `±Xh` (standard deviation of sleep duration)
- Trend: up if ≤ 0.5h SD, down if > 1.0h SD, stable otherwise

---

### Activity section

Inputs: step_count (SUM), active_energy (SUM), apple_exercise_time (SUM).

**Steps score:**

| Steps recent vs baseline | Points |
|---|---|
| Within −10% or better | +2 |
| Within −30% | +1 |
| Below −30% | +0 |

**Active calories score:** same thresholds as steps.

**Exercise time score:**

| Recent avg exercise | Points |
|---|---|
| ≥ 30 min/day | +2 |
| ≥ 15 min/day | +1 |
| < 15 min/day | +0 |

Max possible: 6 points (if all three metrics available).

| Ratio | Status |
|---|---|
| ≥ 0.70 | good |
| ≥ 0.40 | fair |
| < 0.40 | low |

---

### Heart & Lungs section

Inputs: blood_oxygen_saturation (AVG), vo2_max (AVG), respiratory_rate (AVG).

**SpO₂ score** [[11]](#ref-11) [[12]](#ref-12):

| Recent avg SpO₂ | Points |
|---|---|
| ≥ 95% | +2 |
| ≥ 92% | +1 |
| < 92% | +0 |

> The 95% threshold matches the WHO action threshold [[12]](#ref-12). The 92–95% partial-credit band reflects the BTS guideline floor: below 92% is hypoxaemia for most adults, while 92–95% indicates reduced respiratory reserve [[11]](#ref-11).

**VO₂ Max score** [[13]](#ref-13) [[14]](#ref-14):

| VO₂ Max % change vs baseline | Points |
|---|---|
| > −3% | +2 |
| > −8% | +1 |
| ≤ −8% | +0 |

> VO₂ max is recognised by the AHA as a clinical vital sign [[14]](#ref-14). A 122 000-patient study found each 1-MET improvement in CRF reduces all-cause mortality risk by ~13% [[13]](#ref-13). Consumer wearable VO₂ estimates have ±10–15% error vs. lab measurement — trend tracking is more reliable than absolute values.

**Respiratory rate score** [[15]](#ref-15):

| Recent avg RR | Points |
|---|---|
| 12–20 br/min (normal) | +2 |
| 10–24 br/min (borderline) | +1 |
| Outside 10–24 | +0 |

> 12–20 br/min is the universally agreed adult resting normal (StatPearls, ALA, Cleveland Clinic) [[15]](#ref-15).

Max possible: 6 points.

| Ratio | Status |
|---|---|
| ≥ 0.70 | good |
| ≥ 0.40 | fair |
| < 0.40 | low |

---

## Overall status

Aggregates the statuses of all available sections.

| Condition | Overall |
|---|---|
| 2 or more sections are "low" | low |
| fair + low sections > good sections | fair |
| otherwise | good |

---

## Metric cards (top row)

5 cards: Steps, Sleep, HRV, Resting HR, Respiratory Rate.

Each card shows:
- **Value**: recent 5-day average
- **Trend %**: `(recent − baseline) / baseline × 100`, rounded to 1 decimal
  where baseline = `avg(all 30 days)` for display cards (not the separated baseline used for Readiness)
- **Trend label**: positive if > +3%, negative if < −3%, neutral otherwise

---

## Activity vs Recovery chart (Correlation)

Shows the last **7 days** where both step count and HRV data exist on the same calendar day.

- **Load axis (left, %)**: daily step count normalized to the week's max steps
  Formula: `steps_day / max_steps_in_week × 100`
- **HRV axis (right, ms)**: daily average HRV in milliseconds

Only days where **both** metrics have data are plotted. If fewer than 2 matching days exist, the chart is hidden.

> ⚠️ Note: this chart shows correlation visually but does not compute a statistical correlation coefficient.

---

## Weekly Insights (up to 3)

### Insight 1 — Step consistency
Counts how many of the last 7 days had steps ≥ the 30-day average.

- ≥ 5 days → positive: "You hit your average step count on N of the last 7 days"
- < 5 days → warning: "Only N of 7 days above your average steps"

Requires: ≥ 7 days of step data.

### Insight 2 — HRV resilience after hard days
For each day in the overlapping steps/HRV window:
- A day is "high activity" if steps > 1.2 × 30-day average
- The **following day's** HRV is checked against the 30-day HRV average
- If following-day HRV < 95% of average → counted as "HRV dropped"

Array indexing: both slices are most-recent-first. Index `i` = today, `i+1` = yesterday.
Code checks `steps[i+1]` (yesterday was high-activity) → checks `hrv[i]` (today's HRV dropped).

Requires: ≥ 2 high-activity days in the window.

- If > half of high-activity days caused an HRV drop → warning
- Otherwise → positive

> Post-exercise HRV suppression typically lasts 48–72 hours and is a documented marker of autonomic stress [[1]](#ref-1). Chronic suppression (multiple high-activity days with persistent HRV drop) is associated with overreaching syndrome [[16]](#ref-16).

### Insight 3 — Sleep on active vs rest days
Splits the last 30 days into "active" (steps > average) and "rest" days, compares average sleep duration.

- Active sleep avg > rest sleep avg + 0.5 h → positive
- Rest sleep avg > active sleep avg + 0.5 h → warning
- Within 0.5 h → no insight generated

Requires: ≥ 7 days of both steps and sleep data, and at least 1 day in each category.

### Insight 4 — Overtraining warning
Fires after all three insights above are evaluated, before the 3-insight cap.

- Condition: `Activity status == "good"` AND `Readiness score < 50`
- Message: "Your activity is high despite signs of exhaustion. Risk of overtraining is elevated." (type: warning)

---

## Sleep Analysis panel

Shows averages over the last **3 nights**:

| Field | Calculation |
|---|---|
| Deep sleep | avg(sleep_deep last 3 days) — in hours |
| REM sleep | avg(sleep_rem last 3 days) — in hours |
| Awake time | avg(sleep_awake last 3 days) — in hours |
| Efficiency | `(total_sleep − awake) / total_sleep × 100` — green if ≥ 85% |

---

## Sleep deduplication

When two wearables (e.g. Apple Watch + RingConn) both record the same night, raw `SUM` would double-count. All sleep queries use a **MAX-per-source** pattern:

```sql
SELECT MAX(source_sum)
FROM (
    SELECT date, source, SUM(qty) AS source_sum
    FROM metric_points WHERE metric_name = 'sleep_total' ...
    GROUP BY date, source
)
GROUP BY date
```

This picks the device that recorded the longest sleep for each night. Applied in: `GetHealthBriefing`, `GetMetricData`, `GetDashboard`, `GetSleepSummary` (MCP tool), `GetReadinessHistory`.

---

## Known limitations

1. **Readiness requires 7 days minimum** — with fewer days, baseline is too thin to be meaningful. Score defaults to 70 (neutral) if a component has insufficient data.

2. **Sleep section uses full-window baseline** — unlike Readiness, the Sleep section card compares recent to `avg(all 30 days)` rather than `avg(days 6–30)`. This is intentional: the point scale is absolute-duration-based, so baseline dilution matters less.

3. **Recovery % = Readiness score** — they are literally the same number, displayed twice (score and progress bar).

4. **Correlation chart is visual only** — no statistical significance is computed.

5. **No absolute step threshold** — the step goal is the personal 30-day average, not a fixed 10,000. Intentional.

6. **Exercise time** — `apple_exercise_time` counts minutes in the Apple Watch Exercise ring, not total movement time.

7. **Readiness History `valsBefore` sorts by date** — an earlier version sorted by metric value, which put the highest-HRV days first and inflated "recent" averages. Fixed: now sorts by calendar date descending.

8. **Consumer SpO₂ accuracy** — consumer pulse oximeters overestimate SpO₂ by 2–3% in people with darker skin pigmentation [[17]](#ref-17). The 92% threshold may represent true saturation of ~89–90% for affected users.

9. **Consumer VO₂ max accuracy** — wearable VO₂ max estimates have ±10–15% error vs. laboratory testing. Trend direction is more reliable than absolute values.

10. **Sleep regularity** — requires ≥ 7 days of sleep data. With fewer days, no regularity bonus or display is shown.

---

## References

<a id="ref-1"></a>[1] Zulfiqar U et al. (2010). Relation of high heart rate variability to healthy longevity. *American Journal of Cardiology*, 105(8), 1181–1185. — Post-exercise HRV suppression, 72h recovery window. [PubMed](https://pubmed.ncbi.nlm.nih.gov/20381670/)

<a id="ref-2"></a>[2] Gabbett TJ (2016). The training—injury prevention paradox: should athletes be training smarter and harder? *British Journal of Sports Medicine*, 50(5), 273–280. — Acute:Chronic Workload Ratio, 7-day acute window. [PMC7047972](https://pmc.ncbi.nlm.nih.gov/articles/PMC7047972/)

<a id="ref-3"></a>[3] WHOOP Inc. HRV & Recovery Score Methodology. — "HRV will always be the single biggest factor in your recovery score." [whoop.com](https://www.whoop.com/us/en/thelocker/episode364-hrv-cv-explained/)

<a id="ref-4"></a>[4] Altini M (2021). On Heart Rate Variability (HRV) and readiness. *Medium / HRV4Training*. — Personal baseline approach; HRV weighted ~2× RHR. [hrv4training.com](https://www.hrv4training.com/blog2/daily-score-baseline-and-normal-range-an-overview)

<a id="ref-5"></a>[5] Oura Ring (2023). Readiness Score documentation. — RHR as a major readiness contributor. [ouraring.com](https://ouraring.com/blog/readiness-score/)

<a id="ref-6"></a>[6] Beattie K et al. (2024). Heart Rate Variability Applications in Strength and Conditioning: A Narrative Review. *PMC*. — Individual baseline comparisons preferred over population norms. [PMC11204851](https://pmc.ncbi.nlm.nih.gov/articles/PMC11204851/)

<a id="ref-7"></a>[7] Gisselman AS et al. (2026). Monitoring Training Adaptation and Recovery Using HRV via Mobile Devices. *MDPI Sensors*, 26(1). — RMSSD as primary autonomic recovery signal. [MDPI](https://www.mdpi.com/1424-8220/26/1/3)

<a id="ref-8"></a>[8] Watson NF et al. (2015). Recommended Amount of Sleep for a Healthy Adult. *Sleep*, 38(6), 843–844. — AASM/SRS joint consensus: ≥7 hours for adults. [PMC4513271](https://pmc.ncbi.nlm.nih.gov/articles/PMC4513271/)

<a id="ref-9"></a>[9] Van Dongen HPA et al. (2003). The Cumulative Cost of Additional Wakefulness. *Sleep*, 26(2), 117–126. — Cognitive deficits at 6h vs 8h sleep over 14 days. [PubMed](https://pubmed.ncbi.nlm.nih.gov/12603781/)

<a id="ref-10"></a>[10] Huang T et al. (2024). Sleep regularity and major health outcomes: a systematic review and meta-analysis. *Sleep Health*, 10(1), 7–14. — Sleep regularity index predicts all-cause & CV mortality more strongly than mean duration (HR 0.97 vs 0.99). [PMC10782501](https://pmc.ncbi.nlm.nih.gov/articles/PMC10782501/)

<a id="ref-11"></a>[11] British Thoracic Society (2017). BTS Guideline for oxygen use in adults in healthcare and emergency settings. — SpO₂ <92% is clinically significant; 94–98% target range for most adults. [BTS](https://www.brit-thoracic.org.uk/quality-improvement/guidelines/emergency-oxygen/)

<a id="ref-12"></a>[12] WHO (2011). Pulse Oximetry Training Manual. — Action threshold SpO₂ < 95%. [WHO PDF](https://cdn.who.int/media/docs/default-source/patient-safety/pulse-oximetry/who-ps-pulse-oxymetry-training-manual-en.pdf)

<a id="ref-13"></a>[13] Kokkinos P et al. (2018). Exercise Capacity and Mortality in Older Men. *JACC*, 71(9), 964–975. — 122 000-patient study: each MET improvement ↓ all-cause mortality ~13%. [JACC](https://www.jacc.org/doi/10.1016/j.jacc.2018.06.045)

<a id="ref-14"></a>[14] Ross R et al. (2016). Importance of Assessing Cardiorespiratory Fitness in Clinical Practice. *Circulation*, 134(24), e653–e699. — AHA recommendation to treat VO₂ max as clinical vital sign. [AHA](https://www.ahajournals.org/doi/10.1161/CIR.0000000000000461)

<a id="ref-15"></a>[15] Physiology, Respiratory Rate. *StatPearls*, NCBI Bookshelf. — Normal adult RR = 12–20 br/min. [NBK537306](https://www.ncbi.nlm.nih.gov/books/NBK537306/)

<a id="ref-16"></a>[16] Meeusen R et al. (2013). Prevention, Diagnosis, and Treatment of the Overtraining Syndrome. *Medicine & Science in Sports & Exercise*, 45(1), 186–205. — HRV as overtraining/overreaching biomarker; parasympathetic withdrawal pattern. [PubMed](https://pubmed.ncbi.nlm.nih.gov/23247672/)

<a id="ref-17"></a>[17] Sjoding MW et al. (2020). Racial Bias in Pulse Oximetry Measurement. *NEJM*, 383(25), 2477–2478. — Consumer oximeters overestimate SpO₂ by 2–3% in darker skin tones. [NEJM](https://www.nejm.org/doi/10.1056/NEJMc2029240)
