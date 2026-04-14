// Shared flow-band display helpers. DB stores internal keys
// (below_recommended, low_runnable, runnable, med_runnable, high_runnable,
// above_recommended); the UI translates them to friendlier labels + colors.

export type FlowBand =
  | 'below_recommended'
  | 'low_runnable'
  | 'runnable'
  | 'med_runnable'
  | 'high_runnable'
  | 'above_recommended'

export type FlowStatus = 'runnable' | 'caution' | 'low' | 'flood' | 'unknown' | string

// ── Display labels ──────────────────────────────────────────────────────────

const LABEL: Record<string, string> = {
  below_recommended: 'Too Low',
  low_runnable:      'Running',
  runnable:          'Fun',
  med_runnable:      'Fun',
  high_runnable:     'High',
  above_recommended: 'Very High',
}

export function flowBandLabel(band?: string | null, status?: string | null): string {
  if (band && LABEL[band]) return LABEL[band]
  switch (status) {
    case 'runnable': return 'Fun'
    case 'caution':  return 'Too Low'
    case 'low':      return 'Too Low'
    case 'flood':    return 'Very High'
    default:         return 'Unknown'
  }
}

// ── Badge pill classes (bg + text) ─────────────────────────────────────────

const BADGE: Record<string, string> = {
  below_recommended: 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400',
  low_runnable:      'bg-green-100 dark:bg-green-950/50 text-green-700 dark:text-green-400',
  runnable:          'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400',
  med_runnable:      'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400',
  high_runnable:     'bg-blue-100 dark:bg-blue-950/50 text-blue-700 dark:text-blue-400',
  above_recommended: 'bg-amber-100 dark:bg-amber-950/50 text-amber-700 dark:text-amber-400',
}

export function flowBandBadgeClass(band?: string | null, status?: string | null): string {
  if (band && BADGE[band]) return BADGE[band]
  switch (status) {
    case 'runnable': return BADGE.runnable
    case 'caution':  return BADGE.below_recommended
    case 'low':      return BADGE.below_recommended
    case 'flood':    return BADGE.above_recommended
    default:         return 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
  }
}

// ── CFS number text color ──────────────────────────────────────────────────

const CFS_TEXT: Record<string, string> = {
  below_recommended: 'text-red-500',
  low_runnable:      'text-green-500',
  runnable:          'text-emerald-500',
  med_runnable:      'text-emerald-500',
  high_runnable:     'text-blue-500',
  above_recommended: 'text-amber-500',
}

export function flowBandCfsClass(band?: string | null, status?: string | null): string {
  if (band && CFS_TEXT[band]) return CFS_TEXT[band]
  switch (status) {
    case 'runnable': return 'text-emerald-500'
    case 'caution':  return 'text-red-500'
    case 'low':      return 'text-red-500'
    case 'flood':    return 'text-amber-500'
    default:         return 'text-gray-400'
  }
}

// ── Solid hex colors (for SVG strokes / legend swatches) ───────────────────

const SOLID: Record<string, string> = {
  below_recommended: '#ef4444', // red-500
  low_runnable:      '#22c55e', // green-500
  runnable:          '#10b981', // emerald-500
  med_runnable:      '#10b981', // emerald-500
  high_runnable:     '#3b82f6', // blue-500
  above_recommended: '#f59e0b', // amber-500
}

export function flowBandSolidColor(band?: string | null, status?: string | null): string {
  if (band && SOLID[band]) return SOLID[band]
  switch (status) {
    case 'runnable': return '#10b981'
    case 'caution':  return '#ef4444'
    case 'low':      return '#ef4444'
    case 'flood':    return '#f59e0b'
    default:         return '#9ca3af' // gray-400
  }
}

// ── Translucent fills (chart bands) ────────────────────────────────────────

export const FLOW_BAND_FILL: Record<string, string> = {
  below_recommended: 'rgba(239,68,68,0.22)',  // red
  low_runnable:      'rgba(34,197,94,0.25)',  // green
  runnable:          'rgba(16,185,129,0.30)', // emerald
  med_runnable:      'rgba(16,185,129,0.28)', // emerald
  high_runnable:     'rgba(59,130,246,0.25)', // blue
  above_recommended: 'rgba(245,158,11,0.25)', // amber
}

// Map the live band label to a coarse status bucket (used when the modal
// needs to talk about flowStatus even though it's driven by a band label).
export function flowStatusForBand(band?: string | null): FlowStatus {
  if (!band) return 'unknown'
  if (band === 'below_recommended') return 'caution'
  if (band === 'above_recommended') return 'flood'
  if (band === 'runnable' || band === 'low_runnable' ||
      band === 'med_runnable' || band === 'high_runnable') return 'runnable'
  return 'unknown'
}
