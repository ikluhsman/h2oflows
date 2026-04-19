import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'

/**
 * useWatchlistSync — wraps watchlist add/remove with server persistence.
 *
 * Uses getToken() directly for auth checks — more reliable than isAuthenticated
 * which can be false immediately after page load while Supabase restores the
 * session asynchronously. If no token, falls back to localStorage-only.
 */
export function useWatchlistSync() {
  const store = useWatchlistStore()
  const { apiBase } = useRuntimeConfig().public
  const { getToken } = useAuth()

  async function addAndSync(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
    store.addGauge(gauge)
    const token = await getToken()
    if (token) {
      fetch(`${apiBase}/api/v1/watchlist`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          gauge_id: gauge.id,
          reach_slug: gauge.contextReachSlug ?? null,
        }),
      }).catch(() => {})
    }
  }

  async function removeAndSync(gaugeId: string, contextReachSlug?: string | null) {
    store.removeGauge(gaugeId, contextReachSlug)
    const token = await getToken()
    if (token) {
      const slug = contextReachSlug ?? null
      const qs = slug ? `?reach_slug=${encodeURIComponent(slug)}` : ''
      fetch(`${apiBase}/api/v1/watchlist/${gaugeId}${qs}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      }).catch(() => {})
    }
  }

  /**
   * Fetches the server watchlist and adds any (gauge, reach) pairs not already
   * in the local store. Calls the batch API with reach context to hydrate metadata.
   */
  async function loadFromServer() {
    const token = await getToken()
    if (!token) return

    const res = await fetch(`${apiBase}/api/v1/watchlist`, {
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => null)
    if (!res?.ok) return

    const data = await res.json()
    // New format: [{gauge_id, reach_slug}]. Fall back to old gauge_ids for compat.
    const serverItems: { gauge_id: string; reach_slug: string | null }[] =
      data.items ?? (data.gauge_ids ?? []).map((id: string) => ({ gauge_id: id, reach_slug: null }))

    // Find items not already in the local store (matched by gauge_id + reach_slug)
    const newItems = serverItems.filter(item => {
      const slug = item.reach_slug ?? null
      return !store.gauges.some(g => g.id === item.gauge_id && (g.contextReachSlug ?? null) === slug)
    })
    if (newItems.length === 0) return

    // Build batch ids with reach context: "uuid:reach-slug" or "uuid"
    const batchIds = newItems
      .map(item => item.reach_slug ? `${item.gauge_id}:${item.reach_slug}` : item.gauge_id)
      .join(',')

    const batchRes = await fetch(`${apiBase}/api/v1/gauges/batch?ids=${batchIds}`, {
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => null)
    if (!batchRes?.ok) return

    const batchData = await batchRes.json()
    for (const f of batchData.features ?? []) {
      const p = f.properties
      const coords = f.geometry?.coordinates as [number, number] | undefined
      store.addGauge(featureToWatchedGauge(p, coords))
    }
  }

  /**
   * Pushes all locally-stored (gauge, reach) pairs to the server.
   * Called after login to ensure gauges added while anonymous are persisted.
   */
  async function pushLocalToServer() {
    const token = await getToken()
    if (!token) return
    for (const gauge of store.gauges) {
      fetch(`${apiBase}/api/v1/watchlist`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          gauge_id: gauge.id,
          reach_slug: gauge.contextReachSlug ?? null,
        }),
      }).catch(() => {})
    }
  }

  return { addAndSync, removeAndSync, loadFromServer, pushLocalToServer }
}

// Shared helper: map a batch API feature properties object to a WatchedGauge shape.
export function featureToWatchedGauge(
  p: Record<string, any>,
  coords?: [number, number],
): Omit<WatchedGauge, 'watchState' | 'activeSince'> {
  return {
    id:                     p.id,
    externalId:             p.external_id,
    source:                 p.source,
    name:                   p.name ?? null,
    contextReachSlug:       p.context_reach_slug ?? null,
    contextReachCommonName: p.context_reach_common_name ?? null,
    contextReachFullName:   p.context_reach_full_name ?? null,
    contextReachRiverName:  p.context_reach_river_name ?? null,
    contextReachBasinGroup:     p.context_reach_basin_group ?? null,
    contextReachCenterLng:      p.context_reach_center_lng ?? null,
    contextReachPermitRequired: p.context_reach_permit_required ?? false,
    contextReachMultiDayDays:   p.context_reach_multi_day_days ?? 1,
    reachId:                p.reach_id ?? null,
    reachName:              p.reach_name ?? null,
    reachNames:             p.reach_names ?? [],
    reachSlug:              p.reach_slug ?? null,
    reachSlugs:             p.reach_slugs ?? [],
    reachCommonNames:       p.reach_common_names ?? [],
    reachRelationship:      p.reach_relationship ?? null,
    watershedName:          p.watershed_name ?? null,
    basinName:              p.basin_name ?? null,
    riverName:              p.river_name ?? null,
    stateAbbr:              p.state_abbr ?? null,
    lng:                    coords?.[0] ?? null,
    lat:                    coords?.[1] ?? null,
    currentCfs:             p.current_cfs ?? null,
    flowStatus:             p.flow_status ?? 'unknown',
    flowBandLabel:          p.flow_band_label ?? null,
    lastReadingAt:          p.last_reading_at ?? null,
  }
}
