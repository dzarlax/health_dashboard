package ui

const jsSection = `
// ---- Section detail view ----
var SECTION_META = {
  recovery: {
    charts: [
      { id: 'sc-hrv',   metric: 'heart_rate_variability', agg: 'AVG', labelKey: 'metric_heart_rate_variability', unit: 'ms',          color: '#e11d48', type: 'line' },
      { id: 'sc-rhr',   metric: 'resting_heart_rate',     agg: 'AVG', labelKey: 'metric_resting_heart_rate',     unit: 'bpm',         color: '#f97316', type: 'line' },
      { id: 'sc-ready', metric: 'readiness',               agg: '',    labelKey: 'trend_readiness',               unit: '%',           color: '#0ea5e9', type: 'line', virtual: true }
    ],
    explainKeys: ['explain_hrv', 'explain_rhr', 'explain_readiness_score']
  },
  sleep: {
    charts: [
      { id: 'sc-sleep', stacked: true, labelKey: 'sec_sleep', unit: 'h' }
    ],
    explainKeys: ['explain_sleep_deep', 'explain_sleep_rem', 'explain_sleep_reg']
  },
  activity: {
    charts: [
      { id: 'sc-steps', metric: 'step_count',          agg: 'SUM', labelKey: 'metric_step_count',          unit: '',     color: '#059669', type: 'bar' },
      { id: 'sc-cal',   metric: 'active_energy',       agg: 'SUM', labelKey: 'metric_active_energy',       unit: 'kcal', color: '#d97706', type: 'bar' },
      { id: 'sc-ex',    metric: 'apple_exercise_time', agg: 'SUM', labelKey: 'metric_apple_exercise_time', unit: 'min',  color: '#2563eb', type: 'bar' }
    ],
    explainKeys: ['explain_steps', 'explain_exercise']
  },
  cardio: {
    charts: [
      { id: 'sc-spo2', metric: 'blood_oxygen_saturation', agg: 'AVG', labelKey: 'metric_blood_oxygen_saturation', unit: '%',           color: '#06b6d4', type: 'line', refRange: { min: 92, max: 100 } },
      { id: 'sc-vo2',  metric: 'vo2_max',                  agg: 'AVG', labelKey: 'metric_vo2_max',                  unit: 'ml/kg/min',  color: '#8b5cf6', type: 'line' },
      { id: 'sc-resp', metric: 'respiratory_rate',          agg: 'AVG', labelKey: 'metric_respiratory_rate',         unit: 'br/min',    color: '#0ea5e9', type: 'line', refRange: { min: 12, max: 20 } }
    ],
    explainKeys: ['explain_spo2', 'explain_vo2', 'explain_resp']
  }
};

// Localised explanations for the "How it works" cards.
var SECTION_EXPLAIN_DATA = {
  en: {
    explain_hrv: {
      title: 'Heart Rate Variability (HRV)',
      body: 'RMSSD — the variation between heartbeats — is the primary recovery biomarker. Higher HRV signals a well-recovered nervous system. Your score uses a dynamic \u00b11\u202fSD threshold relative to your personal 30-day baseline, adapting to your natural day-to-day variability rather than a fixed cutoff.'
    },
    explain_rhr: {
      title: 'Resting Heart Rate',
      body: 'RHR reflects cardiovascular efficiency. A lower-than-usual RHR suggests solid recovery; an elevated RHR often signals stress, illness, or incomplete recovery. Five-day trends are more meaningful than a single reading \u2014 a one-off spike after poor sleep is normal.'
    },
    explain_readiness_score: {
      title: 'Readiness Score',
      body: 'Computed as HRV\u202f\u00d7\u202f40\u202f% + Resting HR\u202f\u00d7\u202f30\u202f% + Sleep\u202f\u00d7\u202f30\u202f%. Each component compares your 5-day recent average to your personal 30-day baseline. Score 70 = a normal day. Above 80 is genuinely good. 100 is exceptional, not the default.'
    },
    explain_sleep_deep: {
      title: 'Deep (Slow-Wave) Sleep',
      body: 'The most physically restorative phase \u2014 growth hormone is released, muscles repair, and immune function strengthens. Aim for \u226515\u202f% of total sleep. Deep sleep occurs mainly in the first half of the night and decreases with age, alcohol, and late-night exercise.'
    },
    explain_sleep_rem: {
      title: 'REM Sleep',
      body: 'REM supports memory consolidation and emotional regulation. Healthy adults spend ~20\u201325\u202f% of sleep in REM. Alcohol and sleep deprivation suppress REM disproportionately \u2014 even a single late night can cut REM by 30\u201340\u202f%.'
    },
    explain_sleep_reg: {
      title: 'Sleep Regularity',
      body: 'Consistent sleep and wake times predict health outcomes independently of duration. A 2024 meta-analysis (n\u202f\u2248\u202f60\u202f000) found sleep regularity is a stronger predictor of all-cause mortality than mean sleep duration. The \u00b1Xh value shows your standard deviation in nightly sleep length.'
    },
    explain_steps: {
      title: 'Steps Goal',
      body: 'Your step target is your personal 30-day average \u2014 not a fixed 10\u202f000. This adapts to your lifestyle. Staying within 10\u202f% of your baseline indicates consistent activity. A sharp drop signals reduced activity that may affect recovery and fitness.'
    },
    explain_exercise: {
      title: 'Exercise Time',
      body: 'WHO recommends 150\u2013300\u202fmin/week of moderate aerobic activity (~30\u202fmin/day). Apple Exercise ring counts active minutes above brisk-walk intensity \u2014 not total movement time. Vigorous activity counts double toward this goal.'
    },
    explain_spo2: {
      title: 'Blood Oxygen (SpO\u2082)',
      body: 'Normal resting SpO\u2082 is 95\u2013100\u202f%. Below 92\u202f% indicates reduced respiratory reserve (BTS guidelines). Consumer wearables may overestimate SpO\u2082 by 2\u20133\u202f%, especially in people with darker skin tones \u2014 the displayed value may represent a lower true saturation.'
    },
    explain_vo2: {
      title: 'VO\u2082 Max',
      body: 'The gold standard for cardiorespiratory fitness. The AHA recommends it as a clinical vital sign. Each 1-MET improvement reduces all-cause mortality risk by ~13\u202f%. Wearable estimates carry \u00b110\u201315\u202f% error vs. lab testing \u2014 track your personal trend rather than the absolute number.'
    },
    explain_resp: {
      title: 'Respiratory Rate',
      body: 'Normal adult resting rate is 12\u201320\u202fbr/min. Persistent elevation (>20) may signal respiratory illness, autonomic stress, or overtraining. RR often rises 1\u20132 days before other markers during infection, making it an early warning signal.'
    }
  },
  ru: {
    explain_hrv: {
      title: '\u0412\u0430\u0440\u0438\u0430\u0431\u0435\u043b\u044c\u043d\u043e\u0441\u0442\u044c \u0441\u0435\u0440\u0434\u0435\u0447\u043d\u043e\u0433\u043e \u0440\u0438\u0442\u043c\u0430 (\u0412\u0421\u0420)',
      body: 'RMSSD \u2014 \u0432\u0430\u0440\u0438\u0430\u0446\u0438\u044f \u0438\u043d\u0442\u0435\u0440\u0432\u0430\u043b\u043e\u0432 \u043c\u0435\u0436\u0434\u0443 \u0443\u0434\u0430\u0440\u0430\u043c\u0438 \u0441\u0435\u0440\u0434\u0446\u0430 \u2014 \u043e\u0441\u043d\u043e\u0432\u043d\u043e\u0439 \u0431\u0438\u043e\u043c\u0430\u0440\u043a\u0435\u0440 \u0432\u043e\u0441\u0441\u0442\u0430\u043d\u043e\u0432\u043b\u0435\u043d\u0438\u044f. \u0412\u044b\u0441\u043e\u043a\u0430\u044f \u0412\u0421\u0420 \u0441\u0438\u0433\u043d\u0430\u043b\u0438\u0437\u0438\u0440\u0443\u0435\u0442 \u043e \u0445\u043e\u0440\u043e\u0448\u043e \u0432\u043e\u0441\u0441\u0442\u0430\u043d\u043e\u0432\u043b\u0435\u043d\u043d\u043e\u0439 \u043d\u0435\u0440\u0432\u043d\u043e\u0439 \u0441\u0438\u0441\u0442\u0435\u043c\u0435. \u041e\u0446\u0435\u043d\u043a\u0430 \u0438\u0441\u043f\u043e\u043b\u044c\u0437\u0443\u0435\u0442 \u0434\u0438\u043d\u0430\u043c\u0438\u0447\u0435\u0441\u043a\u0438\u0439 \u043f\u043e\u0440\u043e\u0433 \u00b11\u202f\u0421\u041e \u043e\u0442\u043d\u043e\u0441\u0438\u0442\u0435\u043b\u044c\u043d\u043e \u0432\u0430\u0448\u0435\u0433\u043e 30-\u0434\u043d\u0435\u0432\u043d\u043e\u0433\u043e \u0431\u0430\u0437\u043e\u0432\u043e\u0433\u043e \u0443\u0440\u043e\u0432\u043d\u044f, \u0430\u0434\u0430\u043f\u0442\u0438\u0440\u0443\u044f\u0441\u044c \u043a \u0432\u0430\u0448\u0435\u0439 \u0435\u0441\u0442\u0435\u0441\u0442\u0432\u0435\u043d\u043d\u043e\u0439 \u0432\u0430\u0440\u0438\u0430\u0442\u0438\u0432\u043d\u043e\u0441\u0442\u0438.'
    },
    explain_rhr: {
      title: '\u041f\u0443\u043b\u044c\u0441 \u0432 \u043f\u043e\u043a\u043e\u0435',
      body: '\u0427\u0421\u0421 \u043f\u043e\u043a\u043e\u044f \u043e\u0442\u0440\u0430\u0436\u0430\u0435\u0442 \u044d\u0444\u0444\u0435\u043a\u0442\u0438\u0432\u043d\u043e\u0441\u0442\u044c \u0441\u0435\u0440\u0434\u0435\u0447\u043d\u043e-\u0441\u043e\u0441\u0443\u0434\u0438\u0441\u0442\u043e\u0439 \u0441\u0438\u0441\u0442\u0435\u043c\u044b. \u041f\u0443\u043b\u044c\u0441 \u043d\u0438\u0436\u0435 \u043e\u0431\u044b\u0447\u043d\u043e\u0433\u043e \u0443\u043a\u0430\u0437\u044b\u0432\u0430\u0435\u0442 \u043d\u0430 \u0445\u043e\u0440\u043e\u0448\u0435\u0435 \u0432\u043e\u0441\u0441\u0442\u0430\u043d\u043e\u0432\u043b\u0435\u043d\u0438\u0435; \u043f\u043e\u0432\u044b\u0448\u0435\u043d\u043d\u044b\u0439 \u043f\u0443\u043b\u044c\u0441 \u0447\u0430\u0441\u0442\u043e \u0441\u0438\u0433\u043d\u0430\u043b\u0438\u0437\u0438\u0440\u0443\u0435\u0442 \u043e \u0441\u0442\u0440\u0435\u0441\u0441\u0435, \u0431\u043e\u043b\u0435\u0437\u043d\u0438 \u0438\u043b\u0438 \u043d\u0435\u043f\u043e\u043b\u043d\u043e\u043c \u0432\u043e\u0441\u0441\u0442\u0430\u043d\u043e\u0432\u043b\u0435\u043d\u0438\u0438. \u0422\u0435\u043d\u0434\u0435\u043d\u0446\u0438\u0438 \u0437\u0430 5+ \u0434\u043d\u0435\u0439 \u0431\u043e\u043b\u0435\u0435 \u0438\u043d\u0444\u043e\u0440\u043c\u0430\u0442\u0438\u0432\u043d\u044b, \u0447\u0435\u043c \u043e\u0442\u0434\u0435\u043b\u044c\u043d\u044b\u0435 \u0438\u0437\u043c\u0435\u0440\u0435\u043d\u0438\u044f.'
    },
    explain_readiness_score: {
      title: '\u041e\u0446\u0435\u043d\u043a\u0430 \u0433\u043e\u0442\u043e\u0432\u043d\u043e\u0441\u0442\u0438',
      body: '\u0420\u0430\u0441\u0441\u0447\u0438\u0442\u044b\u0432\u0430\u0435\u0442\u0441\u044f \u043a\u0430\u043a \u0412\u0421\u0420\u202f\u00d7\u202f40\u202f% + \u0427\u0421\u0421\u202f\u043f\u043e\u043a\u043e\u044f\u202f\u00d7\u202f30\u202f% + \u0421\u043e\u043d\u202f\u00d7\u202f30\u202f%. \u041a\u0430\u0436\u0434\u044b\u0439 \u043a\u043e\u043c\u043f\u043e\u043d\u0435\u043d\u0442 \u0441\u0440\u0430\u0432\u043d\u0438\u0432\u0430\u0435\u0442 \u0441\u0440\u0435\u0434\u043d\u0435\u0435 \u0437\u0430 \u043f\u043e\u0441\u043b\u0435\u0434\u043d\u0438\u0435 5 \u0434\u043d\u0435\u0439 \u0441 \u0432\u0430\u0448\u0438\u043c 30-\u0434\u043d\u0435\u0432\u043d\u044b\u043c \u0431\u0430\u0437\u043e\u0432\u044b\u043c \u0443\u0440\u043e\u0432\u043d\u0435\u043c. \u041e\u0446\u0435\u043d\u043a\u0430 70 = \u043e\u0431\u044b\u0447\u043d\u044b\u0439 \u0434\u0435\u043d\u044c. \u0412\u044b\u0448\u0435 80 \u2014 \u0434\u0435\u0439\u0441\u0442\u0432\u0438\u0442\u0435\u043b\u044c\u043d\u043e \u0445\u043e\u0440\u043e\u0448\u043e. 100 \u2014 \u0438\u0441\u043a\u043b\u044e\u0447\u0438\u0442\u0435\u043b\u044c\u043d\u044b\u0439 \u0440\u0435\u0437\u0443\u043b\u044c\u0442\u0430\u0442.'
    },
    explain_sleep_deep: {
      title: '\u0413\u043b\u0443\u0431\u043e\u043a\u0438\u0439 (\u043c\u0435\u0434\u043b\u0435\u043d\u043d\u043e\u0432\u043e\u043b\u043d\u043e\u0432\u044b\u0439) \u0441\u043e\u043d',
      body: '\u041d\u0430\u0438\u0431\u043e\u043b\u0435\u0435 \u0444\u0438\u0437\u0438\u0447\u0435\u0441\u043a\u0438 \u0432\u043e\u0441\u0441\u0442\u0430\u043d\u043e\u0432\u0438\u0442\u0435\u043b\u044c\u043d\u0430\u044f \u0444\u0430\u0437\u0430 \u2014 \u0432\u044b\u0434\u0435\u043b\u044f\u0435\u0442\u0441\u044f \u0433\u043e\u0440\u043c\u043e\u043d \u0440\u043e\u0441\u0442\u0430, \u0432\u043e\u0441\u0441\u0442\u0430\u043d\u0430\u0432\u043b\u0438\u0432\u0430\u044e\u0442\u0441\u044f \u043c\u044b\u0448\u0446\u044b, \u0443\u043a\u0440\u0435\u043f\u043b\u044f\u0435\u0442\u0441\u044f \u0438\u043c\u043c\u0443\u043d\u0438\u0442\u0435\u0442. \u0421\u0442\u0440\u0435\u043c\u0438\u0442\u0435\u0441\u044c \u043a \u226515\u202f% \u043e\u0442 \u043e\u0431\u0449\u0435\u0433\u043e \u0432\u0440\u0435\u043c\u0435\u043d\u0438 \u0441\u043d\u0430. \u0413\u043b\u0443\u0431\u043e\u043a\u0438\u0439 \u0441\u043e\u043d \u043f\u0440\u0435\u043e\u0431\u043b\u0430\u0434\u0430\u0435\u0442 \u0432 \u043f\u0435\u0440\u0432\u043e\u0439 \u043f\u043e\u043b\u043e\u0432\u0438\u043d\u0435 \u043d\u043e\u0447\u0438 \u0438 \u0443\u043c\u0435\u043d\u044c\u0448\u0430\u0435\u0442\u0441\u044f \u0441 \u0432\u043e\u0437\u0440\u0430\u0441\u0442\u043e\u043c, \u0430\u043b\u043a\u043e\u0433\u043e\u043b\u0435\u043c \u0438 \u043f\u043e\u0437\u0434\u043d\u0438\u043c\u0438 \u0442\u0440\u0435\u043d\u0438\u0440\u043e\u0432\u043a\u0430\u043c\u0438.'
    },
    explain_sleep_rem: {
      title: 'REM-\u0441\u043e\u043d',
      body: 'REM \u043f\u043e\u0434\u0434\u0435\u0440\u0436\u0438\u0432\u0430\u0435\u0442 \u043a\u043e\u043d\u0441\u043e\u043b\u0438\u0434\u0430\u0446\u0438\u044e \u043f\u0430\u043c\u044f\u0442\u0438 \u0438 \u044d\u043c\u043e\u0446\u0438\u043e\u043d\u0430\u043b\u044c\u043d\u0443\u044e \u0440\u0435\u0433\u0443\u043b\u044f\u0446\u0438\u044e. \u0417\u0434\u043e\u0440\u043e\u0432\u044b\u0435 \u0432\u0437\u0440\u043e\u0441\u043b\u044b\u0435 \u043f\u0440\u043e\u0432\u043e\u0434\u044f\u0442 ~20\u201325\u202f% \u0441\u043d\u0430 \u0432 REM. \u0410\u043b\u043a\u043e\u0433\u043e\u043b\u044c \u0438 \u043d\u0435\u0434\u043e\u0441\u044b\u043f\u0430\u043d\u0438\u0435 \u043d\u0435\u043f\u0440\u043e\u043f\u043e\u0440\u0446\u0438\u043e\u043d\u0430\u043b\u044c\u043d\u043e \u043f\u043e\u0434\u0430\u0432\u043b\u044f\u044e\u0442 REM \u2014 \u0434\u0430\u0436\u0435 \u043e\u0434\u043d\u0430 \u043f\u043e\u0437\u0434\u043d\u044f\u044f \u043d\u043e\u0447\u044c \u043c\u043e\u0436\u0435\u0442 \u0441\u043e\u043a\u0440\u0430\u0442\u0438\u0442\u044c REM \u043d\u0430 30\u201340\u202f%.'
    },
    explain_sleep_reg: {
      title: '\u0420\u0435\u0433\u0443\u043b\u044f\u0440\u043d\u043e\u0441\u0442\u044c \u0441\u043d\u0430',
      body: '\u041f\u043e\u0441\u0442\u043e\u044f\u043d\u043d\u043e\u0435 \u0432\u0440\u0435\u043c\u044f \u043e\u0442\u0445\u043e\u0434\u0430 \u043a\u043e \u0441\u043d\u0443 \u0438 \u043f\u0440\u043e\u0431\u0443\u0436\u0434\u0435\u043d\u0438\u044f \u043f\u0440\u0435\u0434\u0441\u043a\u0430\u0437\u044b\u0432\u0430\u0435\u0442 \u0441\u043e\u0441\u0442\u043e\u044f\u043d\u0438\u0435 \u0437\u0434\u043e\u0440\u043e\u0432\u044c\u044f \u043d\u0435\u0437\u0430\u0432\u0438\u0441\u0438\u043c\u043e \u043e\u0442 \u043f\u0440\u043e\u0434\u043e\u043b\u0436\u0438\u0442\u0435\u043b\u044c\u043d\u043e\u0441\u0442\u0438. \u041c\u0435\u0442\u0430-\u0430\u043d\u0430\u043b\u0438\u0437 2024\u202f\u0433. (n\u202f\u2248\u202f60\u202f000) \u043f\u043e\u043a\u0430\u0437\u0430\u043b, \u0447\u0442\u043e \u0440\u0435\u0433\u0443\u043b\u044f\u0440\u043d\u043e\u0441\u0442\u044c \u0441\u043d\u0430 \u2014 \u0431\u043e\u043b\u0435\u0435 \u0441\u0438\u043b\u044c\u043d\u044b\u0439 \u043f\u0440\u0435\u0434\u0438\u043a\u0442\u043e\u0440 \u0441\u043c\u0435\u0440\u0442\u043d\u043e\u0441\u0442\u0438, \u0447\u0435\u043c \u0441\u0440\u0435\u0434\u043d\u044f\u044f \u043f\u0440\u043e\u0434\u043e\u043b\u0436\u0438\u0442\u0435\u043b\u044c\u043d\u043e\u0441\u0442\u044c. \u0417\u043d\u0430\u0447\u0435\u043d\u0438\u0435 \u00b1X\u0447 \u2014 \u0432\u0430\u0448\u0435 \u0441\u0442\u0430\u043d\u0434\u0430\u0440\u0442\u043d\u043e\u0435 \u043e\u0442\u043a\u043b\u043e\u043d\u0435\u043d\u0438\u0435 \u043f\u0440\u043e\u0434\u043e\u043b\u0436\u0438\u0442\u0435\u043b\u044c\u043d\u043e\u0441\u0442\u0438 \u043d\u043e\u0447\u043d\u043e\u0433\u043e \u0441\u043d\u0430.'
    },
    explain_steps: {
      title: '\u0426\u0435\u043b\u044c \u043f\u043e \u0448\u0430\u0433\u0430\u043c',
      body: '\u0412\u0430\u0448\u0430 \u0446\u0435\u043b\u044c \u2014 \u044d\u0442\u043e \u043b\u0438\u0447\u043d\u044b\u0439 30-\u0434\u043d\u0435\u0432\u043d\u044b\u0439 \u0441\u0440\u0435\u0434\u043d\u0438\u0439 \u043f\u043e\u043a\u0430\u0437\u0430\u0442\u0435\u043b\u044c, \u0430 \u043d\u0435 \u0444\u0438\u043a\u0441\u0438\u0440\u043e\u0432\u0430\u043d\u043d\u044b\u0435 10\u202f000. \u041e\u0442\u043a\u043b\u043e\u043d\u0435\u043d\u0438\u0435 \u0432 \u043f\u0440\u0435\u0434\u0435\u043b\u0430\u0445 10\u202f% \u043e\u0442 \u0431\u0430\u0437\u043e\u0432\u043e\u0433\u043e \u0443\u0440\u043e\u0432\u043d\u044f \u0443\u043a\u0430\u0437\u044b\u0432\u0430\u0435\u0442 \u043d\u0430 \u0441\u0442\u0430\u0431\u0438\u043b\u044c\u043d\u0443\u044e \u0430\u043a\u0442\u0438\u0432\u043d\u043e\u0441\u0442\u044c. \u0420\u0435\u0437\u043a\u043e\u0435 \u043f\u0430\u0434\u0435\u043d\u0438\u0435 \u0441\u0438\u0433\u043d\u0430\u043b\u0438\u0437\u0438\u0440\u0443\u0435\u0442 \u0441\u043d\u0438\u0436\u0435\u043d\u0438\u0435 \u0430\u043a\u0442\u0438\u0432\u043d\u043e\u0441\u0442\u0438, \u043a\u043e\u0442\u043e\u0440\u043e\u0435 \u043c\u043e\u0436\u0435\u0442 \u0432\u043b\u0438\u044f\u0442\u044c \u043d\u0430 \u0432\u043e\u0441\u0441\u0442\u0430\u043d\u043e\u0432\u043b\u0435\u043d\u0438\u0435 \u0438 \u0444\u043e\u0440\u043c\u0443.'
    },
    explain_exercise: {
      title: '\u0412\u0440\u0435\u043c\u044f \u0443\u043f\u0440\u0430\u0436\u043d\u0435\u043d\u0438\u0439',
      body: '\u0412\u041e\u0417 \u0440\u0435\u043a\u043e\u043c\u0435\u043d\u0434\u0443\u0435\u0442 150\u2013300\u202f\u043c\u0438\u043d/\u043d\u0435\u0434\u0435\u043b\u044e \u0443\u043c\u0435\u0440\u0435\u043d\u043d\u043e\u0439 \u0430\u044d\u0440\u043e\u0431\u043d\u043e\u0439 \u0430\u043a\u0442\u0438\u0432\u043d\u043e\u0441\u0442\u0438 (~30\u202f\u043c\u0438\u043d/\u0434\u0435\u043d\u044c). \u041a\u043e\u043b\u044c\u0446\u043e \u0443\u043f\u0440\u0430\u0436\u043d\u0435\u043d\u0438\u0439 Apple \u0441\u0447\u0438\u0442\u0430\u0435\u0442 \u0430\u043a\u0442\u0438\u0432\u043d\u044b\u0435 \u043c\u0438\u043d\u0443\u0442\u044b \u0432\u044b\u0448\u0435 \u0438\u043d\u0442\u0435\u043d\u0441\u0438\u0432\u043d\u043e\u0441\u0442\u0438 \u0431\u044b\u0441\u0442\u0440\u043e\u0439 \u0445\u043e\u0434\u044c\u0431\u044b \u2014 \u043d\u0435 \u043e\u0431\u0449\u0435\u0435 \u0432\u0440\u0435\u043c\u044f \u0434\u0432\u0438\u0436\u0435\u043d\u0438\u044f. \u0418\u043d\u0442\u0435\u043d\u0441\u0438\u0432\u043d\u0430\u044f \u0430\u043a\u0442\u0438\u0432\u043d\u043e\u0441\u0442\u044c \u0437\u0430\u0441\u0447\u0438\u0442\u044b\u0432\u0430\u0435\u0442\u0441\u044f \u0432\u0434\u0432\u043e\u0439\u043d\u0435.'
    },
    explain_spo2: {
      title: '\u041a\u0438\u0441\u043b\u043e\u0440\u043e\u0434 \u0432 \u043a\u0440\u043e\u0432\u0438 (SpO\u2082)',
      body: '\u041d\u043e\u0440\u043c\u0430\u043b\u044c\u043d\u044b\u0439 SpO\u2082 \u0432 \u043f\u043e\u043a\u043e\u0435 \u2014 95\u2013100\u202f%. \u041d\u0438\u0436\u0435 92\u202f% \u0443\u043a\u0430\u0437\u044b\u0432\u0430\u0435\u0442 \u043d\u0430 \u0441\u043d\u0438\u0436\u0435\u043d\u043d\u044b\u0439 \u0434\u044b\u0445\u0430\u0442\u0435\u043b\u044c\u043d\u044b\u0439 \u0440\u0435\u0437\u0435\u0440\u0432 (\u0440\u0435\u043a\u043e\u043c\u0435\u043d\u0434\u0430\u0446\u0438\u0438 BTS). \u041f\u043e\u0442\u0440\u0435\u0431\u0438\u0442\u0435\u043b\u044c\u0441\u043a\u0438\u0435 \u0433\u0430\u0434\u0436\u0435\u0442\u044b \u043c\u043e\u0433\u0443\u0442 \u0437\u0430\u0432\u044b\u0448\u0430\u0442\u044c SpO\u2082 \u043d\u0430 2\u20133\u202f%, \u043e\u0441\u043e\u0431\u0435\u043d\u043d\u043e \u0443 \u043b\u044e\u0434\u0435\u0439 \u0441 \u0442\u0451\u043c\u043d\u044b\u043c \u0442\u043e\u043d\u043e\u043c \u043a\u043e\u0436\u0438.'
    },
    explain_vo2: {
      title: 'VO\u2082 Max',
      body: '\u0417\u043e\u043b\u043e\u0442\u043e\u0439 \u0441\u0442\u0430\u043d\u0434\u0430\u0440\u0442 \u043a\u0430\u0440\u0434\u0438\u043e\u0440\u0435\u0441\u043f\u0438\u0440\u0430\u0442\u043e\u0440\u043d\u043e\u0439 \u043f\u043e\u0434\u0433\u043e\u0442\u043e\u0432\u043a\u0438. AHA \u0440\u0435\u043a\u043e\u043c\u0435\u043d\u0434\u0443\u0435\u0442 \u0435\u0433\u043e \u043a\u0430\u043a \u043a\u043b\u0438\u043d\u0438\u0447\u0435\u0441\u043a\u0438\u0439 \u0436\u0438\u0437\u043d\u0435\u043d\u043d\u044b\u0439 \u043f\u043e\u043a\u0430\u0437\u0430\u0442\u0435\u043b\u044c. \u041a\u0430\u0436\u0434\u043e\u0435 \u0443\u043b\u0443\u0447\u0448\u0435\u043d\u0438\u0435 \u043d\u0430 1 \u041c\u0415\u0422 \u0441\u043d\u0438\u0436\u0430\u0435\u0442 \u0440\u0438\u0441\u043a \u0441\u043c\u0435\u0440\u0442\u043d\u043e\u0441\u0442\u0438 \u043d\u0430 ~13\u202f%. \u041e\u0446\u0435\u043d\u043a\u0438 \u043d\u043e\u0441\u0438\u043c\u044b\u0445 \u0443\u0441\u0442\u0440\u043e\u0439\u0441\u0442\u0432 \u0438\u043c\u0435\u044e\u0442 \u043f\u043e\u0433\u0440\u0435\u0448\u043d\u043e\u0441\u0442\u044c \u00b110\u201315\u202f% \u2014 \u043e\u0442\u0441\u043b\u0435\u0436\u0438\u0432\u0430\u0439\u0442\u0435 \u043f\u0435\u0440\u0441\u043e\u043d\u0430\u043b\u044c\u043d\u044b\u0439 \u0442\u0440\u0435\u043d\u0434, \u0430 \u043d\u0435 \u0430\u0431\u0441\u043e\u043b\u044e\u0442\u043d\u043e\u0435 \u0437\u043d\u0430\u0447\u0435\u043d\u0438\u0435.'
    },
    explain_resp: {
      title: '\u0427\u0430\u0441\u0442\u043e\u0442\u0430 \u0434\u044b\u0445\u0430\u043d\u0438\u044f',
      body: '\u041d\u043e\u0440\u043c\u0430 \u0434\u043b\u044f \u0432\u0437\u0440\u043e\u0441\u043b\u044b\u0445 \u0432 \u043f\u043e\u043a\u043e\u0435 \u2014 12\u201320\u202f\u0434\u044b\u0445/\u043c\u0438\u043d. \u0421\u0442\u043e\u0439\u043a\u043e\u0435 \u043f\u043e\u0432\u044b\u0448\u0435\u043d\u0438\u0435 (>20) \u043c\u043e\u0436\u0435\u0442 \u0441\u0438\u0433\u043d\u0430\u043b\u0438\u0437\u0438\u0440\u043e\u0432\u0430\u0442\u044c \u0437\u0430\u0431\u043e\u043b\u0435\u0432\u0430\u043d\u0438\u0435 \u0434\u044b\u0445\u0430\u0442\u0435\u043b\u044c\u043d\u044b\u0445 \u043f\u0443\u0442\u0435\u0439, \u0430\u0432\u0442\u043e\u043d\u043e\u043c\u043d\u044b\u0439 \u0441\u0442\u0440\u0435\u0441\u0441 \u0438\u043b\u0438 \u043f\u0435\u0440\u0435\u0442\u0440\u0435\u043d\u0438\u0440\u043e\u0432\u0430\u043d\u043d\u043e\u0441\u0442\u044c. \u0427\u0414 \u0447\u0430\u0441\u0442\u043e \u0440\u0430\u0441\u0442\u0451\u0442 \u0437\u0430 1\u20132 \u0434\u043d\u044f \u0434\u043e \u043f\u043e\u044f\u0432\u043b\u0435\u043d\u0438\u044f \u0434\u0440\u0443\u0433\u0438\u0445 \u043c\u0430\u0440\u043a\u0435\u0440\u043e\u0432 \u0438\u043d\u0444\u0435\u043a\u0446\u0438\u0438.'
    }
  },
  sr: {
    explain_hrv: {
      title: 'Varijabilnost sr\u010danog ritma (HRV)',
      body: 'RMSSD \u2014 varijacija izme\u0111u otkucaja srca \u2014 primarni je biomarker oporavka. Vi\u0161i HRV signalizira dobro oporavljeni nervni sistem. Ocjena koristi dinami\u010dki prag \u00b11\u202fSD u odnosu na va\u0161 li\u010dni 30-dnevni prosjek, prilagodiv\u0161i se va\u0161oj prirodnoj svakodnevnoj varijabilnosti.'
    },
    explain_rhr: {
      title: 'Puls u miru',
      body: 'Puls u miru odra\u017eava kardiovaskularnu efikasnost. Ni\u017ei puls nego obi\u010dno ukazuje na solidan oporavak; povi\u0161eni puls \u010desto signalizira stres, bolest ili nepotpuni oporavak. Trendovi tokom 5+ dana su informativniji od pojedinih mjerenja.'
    },
    explain_readiness_score: {
      title: 'Ocjena spremnosti',
      body: 'Izra\u010dunava se kao HRV\u202f\u00d7\u202f40\u202f% + Puls u miru\u202f\u00d7\u202f30\u202f% + San\u202f\u00d7\u202f30\u202f%. Svaka komponenta poredi va\u0161 prosjek za posljednjih 5 dana s va\u0161im 30-dnevnim li\u010dnim prosjekom. Ocjena 70 = normalan dan. Iznad 80 je stvarno dobro. 100 je iznimno.'
    },
    explain_sleep_deep: {
      title: 'Duboki (sporotalasni) san',
      body: 'Najvi\u0161e fizi\u010dki restorativna faza \u2014 oslobadja se hormon rasta, oporavljaju se mi\u0161i\u0107i, ja\u010da imunitet. Ciljajte \u226515\u202f% ukupnog sna. Duboki san prevladava u prvoj polovini no\u0107i i smanjuje se s godinama, alkoholom i kasnim vje\u017ebanjem.'
    },
    explain_sleep_rem: {
      title: 'REM san',
      body: 'REM podr\u017eava konsolidaciju pam\u0107enja i emocionalnu regulaciju. Zdravi odrasli provode ~20\u201325\u202f% sna u REM fazi. Alkohol i nedostatak sna nesrazmjerno suzbijaju REM \u2014 \u010dak jedna kasna no\u0107 mo\u017ee smanjiti REM za 30\u201340\u202f%.'
    },
    explain_sleep_reg: {
      title: 'Regularnost sna',
      body: 'Dosljedno vrijeme odlaska na spavanje i bu\u0111enja predvi\u0111a zdravlje neovisno o trajanju. Meta-analiza iz 2024. (n\u202f\u2248\u202f60\u202f000) pokazala je da je regularnost sna ja\u010di prediktor smrtnosti od svih uzroka nego srednje trajanje. Vrijednost \u00b1Xh pokazuje standardno odstupanje trajanja va\u0161eg sna.'
    },
    explain_steps: {
      title: 'Cilj za korake',
      body: 'Va\u0161 cilj za korake je va\u0161 li\u010dni 30-dnevni prosjek \u2014 ne fiksnih 10\u202f000. Ovo se prilagodava va\u0161em na\u010dinu \u017eivota. Ostajanje unutar 10\u202f% va\u0161eg prosjeka ukazuje na dosljednu aktivnost. Nagli pad signalizira smanjenu aktivnost.'
    },
    explain_exercise: {
      title: 'Vrijeme vje\u017ebanja',
      body: 'SZO preporu\u010duje 150\u2013300\u202fmin/sedmicu umjerene aerobne aktivnosti (~30\u202fmin/dan). Apple Exercise prsten broji aktivne minute iznad intenziteta brzog hoda \u2014 ne ukupno vrijeme kretanja. Intenzivna aktivnost se broji dvostruko.'
    },
    explain_spo2: {
      title: 'Kiseonik u krvi (SpO\u2082)',
      body: 'Normalni SpO\u2082 u miru je 95\u2013100\u202f%. Ispod 92\u202f% ukazuje na smanjenu respiratornu rezervu (BTS smjernice). Potro\u0161a\u010dki ure\u0111aji mogu precijeniti SpO\u2082 za 2\u20133\u202f%, posebno kod osoba s tamnijim tonom ko\u017ee.'
    },
    explain_vo2: {
      title: 'VO\u2082 Maks',
      body: 'Zlatni standard kardiorespiratornog fitnesa. AHA ga preporu\u010duje kao klini\u010dki vitalni znak. Svako pobolj\u0161anje za 1 MET smanjuje rizik od smrtnosti za ~13\u202f%. Procjene nosivog ure\u0111aja imaju gre\u0161ku \u00b110\u201315\u202f% \u2014 pratite li\u010dni trend, ne apsolutni broj.'
    },
    explain_resp: {
      title: 'Respiratorni ritam',
      body: 'Normalan ritam odrasle osobe u miru je 12\u201320\u202fud/min. Stalni porast (>20) mo\u017ee signalizirati respiratorne bolesti, autonomni stres ili pretreniranost. RR \u010desto raste 1\u20132 dana prije ostalih markera tokom infekcije.'
    }
  }
};

function getExplain(key) {
  var d = SECTION_EXPLAIN_DATA[LANG] || SECTION_EXPLAIN_DATA['en'];
  return d[key] || SECTION_EXPLAIN_DATA['en'][key];
}

function showSection(key) {
  if (!briefingData) return;
  currentSection = key;
  $('dashboard-view').style.display = 'none';
  $('chart-view').style.display = 'none';
  $('section-view').style.display = 'block';
  window.scrollTo(0, 0);
  renderSectionView(key);
}

function hideSectionView() {
  currentSection = null;
  destroySectionCharts();
  $('section-view').style.display = 'none';
  $('dashboard-view').style.display = 'block';
  window.scrollTo(0, 0);
}

function destroySectionCharts() {
  sectionCharts.forEach(function(c) { try { c.destroy(); } catch(e) {} });
  sectionCharts = [];
}

function renderSectionView(key) {
  destroySectionCharts();
  var meta = SECTION_META[key];
  if (!meta) return;

  var secData = null;
  if (briefingData && briefingData.sections) {
    for (var i = 0; i < briefingData.sections.length; i++) {
      if (briefingData.sections[i].key === key) { secData = briefingData.sections[i]; break; }
    }
  }

  var content = $('section-content');
  content.innerHTML = '';

  // Header
  var iconKey = key === 'cardio' ? 'heart' : key === 'recovery' ? 'battery' : key;
  var statusHtml = secData ? '<span class="sec-status-badge status-' + secData.status + '">' + t('status_' + secData.status) + '</span>' : '';
  var hdr = document.createElement('div');
  hdr.className = 'sec-header';
  hdr.innerHTML =
    '<div class="sec-title-row">' +
      '<div class="sec-icon sec-icon-' + key + '">' + (ICON_MAP[iconKey] || '') + '</div>' +
      '<div class="sec-title">' + (secData ? secData.title : key) + '</div>' +
      statusHtml +
    '</div>' +
    (secData && secData.summary ? '<div class="sec-summary">' + secData.summary + '</div>' : '');
  content.appendChild(hdr);

  // Detail rows from briefing
  if (secData && secData.details && secData.details.length) {
    var detailBlock = document.createElement('div');
    detailBlock.className = 'sec-detail-block';
    secData.details.forEach(function(d) {
      var row = document.createElement('div');
      row.className = 'insight-detail';
      row.innerHTML =
        '<span class="detail-indicator ' + (d.trend || 'stable') + '"></span>' +
        '<span class="detail-label">' + d.label + '</span>' +
        '<span class="detail-value">' + d.value + '</span>' +
        '<span class="detail-note">' + (d.note || '') + '</span>';
      detailBlock.appendChild(row);
    });
    content.appendChild(detailBlock);
  }

  // Sleep analysis stats (sleep section only)
  if (key === 'sleep' && briefingData && briefingData.sleep) {
    var slp = briefingData.sleep;
    var sleepPanel = document.createElement('div');
    sleepPanel.className = 'sec-sleep-stats';
    [
      { label: t('deep_sleep'), val: formatHM(slp.deep_avg) },
      { label: t('rem_sleep'),  val: formatHM(slp.rem_avg) },
      { label: t('awake_time'), val: formatHM(slp.awake_avg) },
      { label: t('efficiency'), val: slp.efficiency.toFixed(0) + '%', accent: slp.efficiency >= 85 }
    ].forEach(function(item) {
      sleepPanel.innerHTML += '<div class="sleep-stat"><div class="sleep-stat-label">' + item.label + '</div>' +
        '<div class="sleep-stat-value' + (item.accent ? ' accent' : '') + '">' + item.val + '</div></div>';
    });
    content.appendChild(sleepPanel);
  }

  // Charts area
  var chartsArea = document.createElement('div');
  chartsArea.className = 'sec-charts-area';
  content.appendChild(chartsArea);

  var from30 = daysAgoStr(29), to30 = todayStr();
  meta.charts.forEach(function(cfg, idx) {
    var block = document.createElement('div');
    block.className = 'sec-chart-block';
    block.innerHTML =
      '<div class="sec-chart-title">' + (t(cfg.labelKey) || cfg.labelKey) +
      (cfg.unit ? ' <span class="sec-chart-unit">(' + cfg.unit + ')</span>' : '') + '</div>' +
      '<div class="sec-chart-wrap"><canvas id="' + cfg.id + '"></canvas></div>';
    chartsArea.appendChild(block);

    if (cfg.stacked) {
      loadSectionSleepChart(cfg.id, from30, to30, idx);
    } else if (cfg.virtual) {
      loadSectionReadinessChart(cfg.id, idx);
    } else {
      loadSectionMetricChart(cfg, from30, to30, idx);
    }
  });

  // Explanations ("How it works")
  if (meta.explainKeys && meta.explainKeys.length) {
    var explainArea = document.createElement('div');
    explainArea.className = 'sec-explain-area';
    var heading = document.createElement('div');
    heading.className = 'sec-explain-heading';
    heading.textContent = t('how_it_works');
    explainArea.appendChild(heading);
    var explainGrid = document.createElement('div');
    explainGrid.className = 'sec-explain-grid';
    meta.explainKeys.forEach(function(ek) {
      var ex = getExplain(ek);
      if (!ex) return;
      var card = document.createElement('div');
      card.className = 'sec-explain-card';
      card.innerHTML = '<div class="sec-explain-title">' + ex.title + '</div><div class="sec-explain-body">' + ex.body + '</div>';
      explainGrid.appendChild(card);
    });
    explainArea.appendChild(explainGrid);
    content.appendChild(explainArea);
  }
}

function loadSectionMetricChart(cfg, from, to, idx) {
  var url = '/api/metrics/data?metric=' + encodeURIComponent(cfg.metric) +
    '&from=' + from + '&to=' + to + '&bucket=day&agg=' + cfg.agg;
  fetch(url).then(function(r) { return r.json(); }).then(function(data) {
    var pts = (data.points || []).filter(function(p) { return p.qty > 0; });
    if (!pts.length) return;
    var canvas = $(cfg.id);
    if (!canvas) return;
    // Keep raw dates for lookup; store formatted labels separately for display
    var rawDates = pts.map(function(p) { return p.date; });
    var labels = rawDates.map(fmtAxisDate);
    var vals = pts.map(function(p) { return p.qty; });
    var isBar = cfg.type === 'bar';
    var datasets = [{
      label: t(cfg.labelKey) || cfg.labelKey,
      data: vals,
      borderColor: cfg.color,
      backgroundColor: isBar ? cfg.color + '88' : cfg.color + '18',
      fill: !isBar,
      borderWidth: isBar ? 0 : 2,
      borderRadius: isBar ? 3 : 0,
      pointRadius: 0,
      tension: 0.35
    }];
    // Reference range shading (SpO2 normal zone, RR normal zone)
    if (cfg.refRange) {
      datasets.push({ label: '_refmax', data: labels.map(function() { return cfg.refRange.max; }), borderColor: '#05966640', borderDash: [4, 3], borderWidth: 1.5, pointRadius: 0, fill: false, tension: 0 });
      datasets.push({ label: '_refmin', data: labels.map(function() { return cfg.refRange.min; }), borderColor: '#05966640', borderDash: [4, 3], borderWidth: 1.5, pointRadius: 0, fill: '-1', backgroundColor: '#05966610', tension: 0 });
    }
    var c = new Chart(canvas, {
      type: cfg.type,
      data: { labels: labels, datasets: datasets },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            backgroundColor: '#fff', borderColor: '#e7e5e4', borderWidth: 1,
            titleColor: '#78716c', bodyColor: '#1c1917', padding: 8,
            filter: function(item) { return item.dataset.label && !item.dataset.label.startsWith('_'); },
            callbacks: {
              // label is already formatted by fmtAxisDate — return as-is
              title: function(items) { return items[0].label; },
              label: function(ctx) { return ' ' + fmt2(ctx.parsed.y) + (cfg.unit ? ' ' + cfg.unit : ''); }
            }
          }
        },
        scales: {
          x: { ticks: { color: '#78716c', maxTicksLimit: 10, font: { size: 11 } }, grid: { color: '#f0efed' } },
          y: { beginAtZero: isBar, ticks: { color: '#78716c', font: { size: 11 } }, grid: { color: '#f0efed' } }
        }
      }
    });
    sectionCharts[idx] = c;
  }).catch(function() {});
}

function loadSectionReadinessChart(canvasId, idx) {
  fetch('/api/readiness-history?days=30').then(function(r) { return r.json(); }).then(function(d) {
    var pts = d.points || [];
    if (!pts.length) return;
    var canvas = $(canvasId);
    if (!canvas) return;
    var labels = pts.map(function(p) { return fmtAxisDate(p.date); });
    var vals = pts.map(function(p) { return p.score; });
    var c = new Chart(canvas, {
      type: 'line',
      data: {
        labels: labels,
        datasets: [{ label: t('trend_readiness'), data: vals, borderColor: '#0ea5e9', backgroundColor: '#0ea5e918', fill: true, borderWidth: 2, pointRadius: 0, tension: 0.35 }]
      },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            backgroundColor: '#fff', borderColor: '#e7e5e4', borderWidth: 1,
            titleColor: '#78716c', bodyColor: '#1c1917', padding: 8,
            callbacks: {
              title: function(items) { return items[0].label; },
              label: function(ctx) { return ' ' + t('trend_readiness') + ': ' + Math.round(ctx.parsed.y) + '%'; }
            }
          }
        },
        scales: {
          x: { ticks: { color: '#78716c', maxTicksLimit: 10, font: { size: 11 } }, grid: { color: '#f0efed' } },
          y: { min: 0, max: 100, ticks: { color: '#78716c', font: { size: 11 }, callback: function(v) { return v + '%'; } }, grid: { color: '#f0efed' } }
        }
      }
    });
    sectionCharts[idx] = c;
  }).catch(function() {});
}

function loadSectionSleepChart(canvasId, from, to, idx) {
  Promise.all(SLEEP_PHASES.map(function(ph) {
    return fetch('/api/metrics/data?metric=' + ph.metric + '&from=' + from + '&to=' + to + '&bucket=day&agg=AVG').then(function(r) { return r.json(); });
  })).then(function(results) {
    var labelSet = new Set();
    results.forEach(function(r) { (r.points || []).forEach(function(p) { labelSet.add(p.date); }); });
    var rawDates = Array.from(labelSet).sort();
    if (!rawDates.length) return;
    var canvas = $(canvasId);
    if (!canvas) return;
    var ptMaps = results.map(function(r) {
      var m = {};
      (r.points || []).forEach(function(p) { m[p.date] = p.qty; });
      return m;
    });
    var datasets = SLEEP_PHASES.map(function(ph, i) {
      return {
        label: t(ph.labelKey),
        data: rawDates.map(function(l) { return ptMaps[i][l] || 0; }),
        backgroundColor: ph.color + 'cc', borderColor: ph.color, borderWidth: 1, stack: 'sleep', borderRadius: 3
      };
    });
    var displayLabels = rawDates.map(fmtAxisDate);
    var c = new Chart($(canvasId), {
      type: 'bar',
      data: { labels: displayLabels, datasets: datasets },
      options: {
        responsive: true, maintainAspectRatio: false,
        interaction: { mode: 'index', intersect: false },
        plugins: {
          legend: { display: true, labels: { color: '#78716c', boxWidth: 10, font: { size: 11 } } },
          tooltip: {
            backgroundColor: '#fff', borderColor: '#e7e5e4', borderWidth: 1,
            titleColor: '#78716c', bodyColor: '#1c1917',
            callbacks: {
              title: function(items) { return items[0].label; },
              label: function(ctx) { return ' ' + ctx.dataset.label + ': ' + fmt2(ctx.parsed.y) + 'h'; }
            }
          }
        },
        scales: {
          x: { stacked: true, ticks: { color: '#78716c', maxTicksLimit: 10, font: { size: 11 } }, grid: { color: '#f0efed' } },
          y: { stacked: true, ticks: { color: '#78716c', font: { size: 11 }, callback: function(v) { return v + 'h'; } }, grid: { color: '#f0efed' } }
        }
      }
    });
    sectionCharts[idx] = c;
  }).catch(function() {});
}
`
