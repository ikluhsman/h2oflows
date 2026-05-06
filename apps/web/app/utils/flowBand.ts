// Shared flow-band display helpers. DB stores 3 bands: low / running / high.

export type FlowBand = 'low' | 'running' | 'high'

export type FlowStatus = 'runnable' | 'caution' | 'flood' | 'unknown' | string

// ── Display labels ──────────────────────────────────────────────────────────

const LABEL: Record<string, string> = {
  low:     'Too Low',
  running: 'Running',
  high:    'High',
}

export function flowBandLabel(band?: string | null, status?: string | null): string {
  if (band && LABEL[band]) return LABEL[band]
  switch (status) {
    case 'runnable': return 'Running'
    case 'caution':  return 'Too Low'
    case 'flood':    return 'High'
    default:         return 'Unknown'
  }
}

// ── Badge pill classes (bg + text) ─────────────────────────────────────────

const BADGE: Record<string, string> = {
  low:     'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400',
  running: 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400',
  high:    'bg-sky-100 dark:bg-sky-950/50 text-sky-700 dark:text-sky-400',
}

export function flowBandBadgeClass(band?: string | null, status?: string | null): string {
  if (band && BADGE[band]) return BADGE[band]
  switch (status) {
    case 'runnable': return BADGE.running
    case 'caution':  return BADGE.low
    case 'flood':    return BADGE.high
    default:         return 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
  }
}

// ── CFS number text color ──────────────────────────────────────────────────

const CFS_TEXT: Record<string, string> = {
  low:     'text-red-500',
  running: 'text-emerald-500',
  high:    'text-sky-500',
}

export function flowBandCfsClass(band?: string | null, status?: string | null): string {
  if (band && CFS_TEXT[band]) return CFS_TEXT[band]
  switch (status) {
    case 'runnable': return 'text-emerald-500'
    case 'caution':  return 'text-red-500'
    case 'flood':    return 'text-sky-500'
    default:         return 'text-gray-400'
  }
}

// ── Solid hex colors (for SVG strokes / legend swatches) ───────────────────

const SOLID: Record<string, string> = {
  low:     '#ef4444', // red-500
  running: '#34d399', // emerald-400
  high:    '#38bdf8', // sky-400
}

export function flowBandSolidColor(band?: string | null, status?: string | null): string {
  if (band && SOLID[band]) return SOLID[band]
  switch (status) {
    case 'runnable': return SOLID.running
    case 'caution':  return SOLID.low
    case 'flood':    return SOLID.high
    default:         return '#9ca3af' // gray-400
  }
}

// ── Translucent fills (chart bands) ────────────────────────────────────────

export const FLOW_BAND_FILL: Record<string, string> = {
  low:     'rgba(239,68,68,0.22)',   // red
  running: 'rgba(52,211,153,0.28)',  // emerald-400
  high:    'rgba(56,189,248,0.25)',  // sky-400
}

// Map the live band label to a coarse status bucket.
export function flowStatusForBand(band?: string | null): FlowStatus {
  if (!band) return 'unknown'
  if (band === 'low')     return 'caution'
  if (band === 'high')    return 'flood'
  if (band === 'running') return 'runnable'
  return 'unknown'
}
