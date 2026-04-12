import { useWatchlistStore } from '~/stores/watchlist'
import { featureToWatchedGauge } from '~/composables/useWatchlistSync'

// Fetches fresh metadata + current_cfs for all watched gauges from the API.
// Sends reach context so each (gauge, reach) pair gets the correct flow band coloring.
export function useWatchlistRefresh() {
  const store = useWatchlistStore()
  const { apiBase } = useRuntimeConfig().public

  async function refresh() {
    if (store.gauges.length === 0) return

    // Build "uuid:reach-slug" pairs for gauges with reach context; plain "uuid" for standalone.
    const ids = store.gauges
      .map(g => g.contextReachSlug ? `${g.id}:${g.contextReachSlug}` : g.id)
      .join(',')

    try {
      const res = await fetch(`${apiBase}/api/v1/gauges/batch?ids=${ids}`)
      if (!res.ok) return
      const data = await res.json()
      for (const f of data.features ?? []) {
        const p = f.properties
        const coords = f.geometry?.coordinates as [number, number] | undefined
        store.refreshFromApi(featureToWatchedGauge(p, coords))
      }
    } catch {
      // Non-fatal — stale data is better than crashing the dashboard
    }
  }

  return { refresh }
}
