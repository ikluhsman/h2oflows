import { ref, computed } from 'vue'
import { useDiurnalPattern, type DiurnalPattern } from './useDiurnalPattern'

interface CacheEntry {
  pattern: DiurnalPattern
  fetchedAt: number
}

const cache = new Map<string, CacheEntry>()
const STALE_MS = 15 * 60 * 1000 // 15 min

/**
 * Returns a reactive diurnal pattern for the given gauge, fetching 48h of
 * readings on first access (then cached for 15 min). Designed for dashboard
 * card use — lightweight, shared across components.
 */
export function useDiurnalCache(gaugeId: string) {
  const { apiBase } = useRuntimeConfig().public

  const pattern = ref<DiurnalPattern>({
    detected: false, phase: null, estimatedPeakHour: null,
    peakCfs: null, troughCfs: null, swingPct: null, forecast: null,
  })

  const cached = cache.get(gaugeId)
  if (cached && Date.now() - cached.fetchedAt < STALE_MS) {
    pattern.value = cached.pattern
  } else {
    fetchAndAnalyze()
  }

  async function fetchAndAnalyze() {
    try {
      const since = new Date(Date.now() - 48 * 3_600_000).toISOString()
      const res = await fetch(`${apiBase}/api/v1/gauges/${gaugeId}/readings?since=${since}&limit=500`)
      if (!res.ok) return
      const readings = ([...(await res.json())]).reverse()
      const result = useDiurnalPattern(readings)
      pattern.value = result
      cache.set(gaugeId, { pattern: result, fetchedAt: Date.now() })
    } catch { /* silent */ }
  }

  return { pattern: computed(() => pattern.value) }
}
