import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'

// Fetches fresh metadata + current_cfs for all watched gauges from the API.
// Called on dashboard mount so persisted watchlist data stays in sync.
export function useWatchlistRefresh() {
  const store = useWatchlistStore()
  const { apiBase } = useRuntimeConfig().public

  async function refresh() {
    if (store.gauges.length === 0) return

    const ids = store.gauges.map(g => g.id).join(',')
    try {
      const res = await fetch(`${apiBase}/api/v1/gauges/batch?ids=${ids}`)
      if (!res.ok) return
      const data = await res.json()
      for (const f of data.features ?? []) {
        const p = f.properties
        const coords = f.geometry?.coordinates as [number, number] | undefined
        store.refreshFromApi({
          id:            p.id,
          externalId:    p.external_id,
          source:        p.source,
          name:          p.name ?? null,
          featured:      p.featured ?? false,
          reachId:           p.reach_id ?? null,
          reachName:         p.reach_name ?? null,
          reachNames:        p.reach_names ?? [],
          reachSlug:         p.reach_slug ?? null,
          reachSlugs:        p.reach_slugs ?? [],
          reachRelationship: p.reach_relationship ?? null,
          pollTier:      p.poll_tier,
          watershedName: p.watershed_name ?? null,
          basinName:     p.basin_name ?? null,
          riverName:     p.river_name ?? null,
          stateAbbr:     p.state_abbr ?? null,
          lng:           coords?.[0] ?? null,
          lat:           coords?.[1] ?? null,
          currentCfs:    p.current_cfs ?? null,
          flowStatus:    p.flow_status ?? 'unknown',
          flowBandLabel: p.flow_band_label ?? null,
          lastReadingAt: p.last_reading_at ?? null,
        } satisfies Omit<WatchedGauge, 'watchState' | 'activeSince'>)
      }
    } catch {
      // Non-fatal — stale data is better than crashing the dashboard
    }
  }

  return { refresh }
}
