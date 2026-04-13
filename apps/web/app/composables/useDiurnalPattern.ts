/**
 * useDiurnalPattern — detects snowmelt-driven diurnal flow cycles.
 *
 * Colorado rivers (and most snowmelt-fed western rivers) follow a predictable
 * daily rhythm from ~April through July: flows rise through the morning as
 * temperatures warm and snowmelt accelerates, peak in the early-to-mid afternoon,
 * then fall through the evening as temperatures drop.
 *
 * A paddler checking at 8am needs to know "this will peak at ~1,400 around 3pm"
 * not just "it's 1,040 right now." This composable derives that context from the
 * 48h rolling cache — no extra API calls required.
 */

export interface DiurnalForecast {
  cfs:  number
  hour: number   // 0–23 local time
  label: string  // e.g. "~216 cfs by 3pm"
}

export interface DiurnalPattern {
  detected:          boolean
  phase:             'rising' | 'falling' | 'near_peak' | 'near_trough' | 'stable' | null
  estimatedPeakHour: number | null   // 0–23 local time
  peakCfs:           number | null
  troughCfs:         number | null
  swingPct:          number | null   // (peak − trough) / trough × 100
  forecast:          DiurnalForecast | null  // 4h look-ahead
}

interface Reading {
  cfs:       number
  timestamp: string
}

const NULL_PATTERN: DiurnalPattern = {
  detected:          false,
  phase:             null,
  estimatedPeakHour: null,
  peakCfs:           null,
  troughCfs:         null,
  swingPct:          null,
  forecast:          null,
}

/**
 * Analyzes an array of recent readings (any order) and returns a DiurnalPattern.
 * Readings should cover at least 24h for reliable detection; 48h is ideal.
 */
export function useDiurnalPattern(readings: Reading[]): DiurnalPattern {
  if (readings.length < 12) return NULL_PATTERN

  const now = Date.now()
  const h24 = 24 * 60 * 60 * 1000
  const h48 = 48 * 60 * 60 * 1000

  // Split into two 24h windows.
  const yesterday = readings.filter(r => {
    const age = now - new Date(r.timestamp).getTime()
    return age >= h24 && age < h48
  })
  const today = readings.filter(r => {
    const age = now - new Date(r.timestamp).getTime()
    return age < h24
  })

  // Need reasonable coverage in both windows to detect a pattern.
  if (yesterday.length < 8 || today.length < 4) return NULL_PATTERN

  // Find yesterday's peak and trough.
  const yPeak  = maxBy(yesterday, r => r.cfs)
  const yTrough = minBy(yesterday, r => r.cfs)
  if (!yPeak || !yTrough) return NULL_PATTERN

  const swing = ((yPeak.cfs - yTrough.cfs) / yTrough.cfs) * 100

  // A meaningful diurnal swing is at least 20%.
  // Regulated rivers or rain-driven floods won't pass this test.
  if (swing < 20) return NULL_PATTERN

  // Peak should fall in the 10:00–20:00 window (10am–8pm local).
  // If a river peaks at 2am it's rain, not snowmelt.
  const peakHour = new Date(yPeak.timestamp).getHours()
  if (peakHour < 10 || peakHour > 20) return NULL_PATTERN

  // Pattern confirmed. Now determine today's phase.
  const phase = detectPhase(today, yPeak.cfs, yTrough.cfs)

  // 4-hour look-ahead: estimate CFS by interpolating yesterday's hourly curve.
  const forecast = computeForecast(yesterday, 4)

  return {
    detected:          true,
    phase,
    estimatedPeakHour: peakHour,
    peakCfs:           yPeak.cfs,
    troughCfs:         yTrough.cfs,
    swingPct:          Math.round(swing),
    forecast,
  }
}

/**
 * Determine the current phase of today's diurnal cycle.
 * Uses the last 2 hours of readings for trend direction and compares
 * the current value to yesterday's peak/trough for proximity.
 */
function detectPhase(
  today: Reading[],
  yPeakCfs: number,
  yTroughCfs: number,
): DiurnalPattern['phase'] {
  if (today.length < 2) return 'stable'

  const sorted = [...today].sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
  )
  const current = sorted[sorted.length - 1].cfs

  // Near peak: within 10% of yesterday's peak.
  if (Math.abs(current - yPeakCfs) / yPeakCfs < 0.10) return 'near_peak'

  // Near trough: within 10% of yesterday's trough.
  if (Math.abs(current - yTroughCfs) / yTroughCfs < 0.10) return 'near_trough'

  // Trend: compare last reading to the average of readings 1–2h ago.
  const twoHoursAgo = Date.now() - 2 * 60 * 60 * 1000
  const recent = sorted.filter(r => new Date(r.timestamp).getTime() >= twoHoursAgo)
  if (recent.length < 2) return 'stable'

  const oldest = recent[0].cfs
  const delta  = current - oldest

  // Require at least a 2% change to call it rising/falling.
  if (delta / oldest >  0.02) return 'rising'
  if (delta / oldest < -0.02) return 'falling'
  return 'stable'
}

/**
 * Estimate what CFS will be `hoursAhead` hours from now, using yesterday's
 * hourly profile as the template. Finds the reading from yesterday at the
 * same hour-of-day as the target time.
 */
function computeForecast(yesterday: Reading[], hoursAhead: number): DiurnalForecast | null {
  if (yesterday.length < 8) return null

  const targetTime = new Date(Date.now() + hoursAhead * 3_600_000)
  const targetHour = targetTime.getHours()

  // Bucket yesterday's readings by hour, take the average per bucket.
  const buckets = new Map<number, number[]>()
  for (const r of yesterday) {
    const h = new Date(r.timestamp).getHours()
    if (!buckets.has(h)) buckets.set(h, [])
    buckets.get(h)!.push(r.cfs)
  }

  const hourAvg = buckets.get(targetHour)
  if (!hourAvg || hourAvg.length === 0) return null

  const cfs = Math.round(hourAvg.reduce((a, b) => a + b, 0) / hourAvg.length)
  const ampm = targetHour >= 12 ? 'pm' : 'am'
  const display = targetHour % 12 === 0 ? 12 : targetHour % 12

  return {
    cfs,
    hour: targetHour,
    label: `~${cfs.toLocaleString()} cfs by ${display}${ampm}`,
  }
}

// ---- Tiny helpers -----------------------------------------------------------

function maxBy<T>(arr: T[], fn: (v: T) => number): T | null {
  if (!arr.length) return null
  return arr.reduce((best, v) => fn(v) > fn(best) ? v : best)
}

function minBy<T>(arr: T[], fn: (v: T) => number): T | null {
  if (!arr.length) return null
  return arr.reduce((best, v) => fn(v) < fn(best) ? v : best)
}
