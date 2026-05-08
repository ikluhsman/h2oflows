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

// ── Palette-aware solid hex colors (keyed by Tailwind primary name) ─────────
// running = primary brand color; low = warm danger; high = cool flood

export const PALETTE_FLOW_SOLID: Record<string, { low: string; running: string; high: string }> = {
  blue:    { low: '#ef4444', running: '#34d399', high: '#38bdf8' },
  sky:     { low: '#f472b6', running: '#0ea5e9', high: '#818cf8' },
  teal:    { low: '#ef4444', running: '#2dd4bf', high: '#8b5cf6' },
  emerald: { low: '#fbbf24', running: '#10b981', high: '#0ea5e9' },
  indigo:  { low: '#f472b6', running: '#818cf8', high: '#2dd4bf' },
  orange:  { low: '#ef4444', running: '#f97316', high: '#60a5fa' },
  fuchsia: { low: '#fbbf24', running: '#d946ef', high: '#38bdf8' },
  rose:    { low: '#f97316', running: '#f43f5e', high: '#818cf8' },
  lime:    { low: '#ef4444', running: '#84cc16', high: '#22d3ee' },
}

// ── Badge pill classes per palette (pre-declared so Tailwind includes them) ─

export const PALETTE_BADGE_CLASS: Record<string, { low: string; running: string; high: string }> = {
  blue:    { low: 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400',         running: 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400', high: 'bg-sky-100 dark:bg-sky-950/50 text-sky-700 dark:text-sky-400' },
  sky:     { low: 'bg-pink-100 dark:bg-pink-950/50 text-pink-600 dark:text-pink-400',     running: 'bg-sky-100 dark:bg-sky-950/50 text-sky-700 dark:text-sky-400',                 high: 'bg-indigo-100 dark:bg-indigo-950/50 text-indigo-700 dark:text-indigo-400' },
  teal:    { low: 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400',         running: 'bg-teal-100 dark:bg-teal-950/50 text-teal-700 dark:text-teal-400',             high: 'bg-violet-100 dark:bg-violet-950/50 text-violet-700 dark:text-violet-400' },
  emerald: { low: 'bg-amber-100 dark:bg-amber-950/50 text-amber-700 dark:text-amber-400', running: 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400', high: 'bg-sky-100 dark:bg-sky-950/50 text-sky-700 dark:text-sky-400' },
  indigo:  { low: 'bg-pink-100 dark:bg-pink-950/50 text-pink-600 dark:text-pink-400',     running: 'bg-indigo-100 dark:bg-indigo-950/50 text-indigo-700 dark:text-indigo-400',     high: 'bg-teal-100 dark:bg-teal-950/50 text-teal-700 dark:text-teal-400' },
  orange:  { low: 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400',         running: 'bg-orange-100 dark:bg-orange-950/50 text-orange-700 dark:text-orange-400',     high: 'bg-blue-100 dark:bg-blue-950/50 text-blue-700 dark:text-blue-400' },
  fuchsia: { low: 'bg-amber-100 dark:bg-amber-950/50 text-amber-700 dark:text-amber-400', running: 'bg-fuchsia-100 dark:bg-fuchsia-950/50 text-fuchsia-700 dark:text-fuchsia-400', high: 'bg-sky-100 dark:bg-sky-950/50 text-sky-700 dark:text-sky-400' },
  rose:    { low: 'bg-orange-100 dark:bg-orange-950/50 text-orange-700 dark:text-orange-400', running: 'bg-rose-100 dark:bg-rose-950/50 text-rose-700 dark:text-rose-400',         high: 'bg-indigo-100 dark:bg-indigo-950/50 text-indigo-700 dark:text-indigo-400' },
  lime:    { low: 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400',         running: 'bg-lime-100 dark:bg-lime-950/50 text-lime-700 dark:text-lime-400',             high: 'bg-cyan-100 dark:bg-cyan-950/50 text-cyan-700 dark:text-cyan-400' },
}

export function flowBandPaletteBadgeClass(band?: string | null, status?: string | null, primary?: string | null): string {
  const table = PALETTE_BADGE_CLASS[primary ?? 'blue'] ?? PALETTE_BADGE_CLASS.blue
  const b = band ?? (status === 'caution' ? 'low' : status === 'runnable' ? 'running' : status === 'flood' ? 'high' : null)
  return b ? (table[b as keyof typeof table] ?? '') : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
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
