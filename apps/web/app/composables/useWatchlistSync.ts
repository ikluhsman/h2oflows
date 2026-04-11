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
        body: JSON.stringify({ gauge_id: gauge.id }),
      }).catch(() => {})
    }
  }

  async function removeAndSync(gaugeId: string) {
    store.removeGauge(gaugeId)
    const token = await getToken()
    if (token) {
      fetch(`${apiBase}/api/v1/watchlist/${gaugeId}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      }).catch(() => {})
    }
  }

  /**
   * Fetches the server watchlist and adds any gauges not already in the local
   * store. Calls the batch API to hydrate metadata for new entries.
   */
  async function loadFromServer() {
    const token = await getToken()
    if (!token) return

    const res = await fetch(`${apiBase}/api/v1/watchlist`, {
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => null)
    if (!res?.ok) return

    const data = await res.json()
    const serverIds: string[] = data.gauge_ids ?? []

    const localIds = new Set(store.gauges.map(g => g.id))
    const newIds = serverIds.filter(id => !localIds.has(id))
    if (newIds.length === 0) return

    const batchRes = await fetch(`${apiBase}/api/v1/gauges/batch?ids=${newIds.join(',')}`, {
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => null)
    if (!batchRes?.ok) return

    const batchData = await batchRes.json()
    for (const f of batchData.features ?? []) {
      const p = f.properties
      const coords = f.geometry?.coordinates as [number, number] | undefined
      store.addGauge({
        id:                p.id,
        externalId:        p.external_id,
        source:            p.source,
        name:              p.name ?? null,
        featured:          p.featured ?? false,
        reachId:           p.reach_id ?? null,
        reachName:         p.reach_name ?? null,
        reachNames:        p.reach_names ?? [],
        reachSlug:         p.reach_slug ?? null,
        reachSlugs:        p.reach_slugs ?? [],
        reachRelationship: p.reach_relationship ?? null,
        pollTier:          p.poll_tier ?? 'cold',
        watershedName:     p.watershed_name ?? null,
        basinName:         p.basin_name ?? null,
        riverName:         p.river_name ?? null,
        stateAbbr:         p.state_abbr ?? null,
        lng:               coords?.[0] ?? null,
        lat:               coords?.[1] ?? null,
        currentCfs:        p.current_cfs ?? null,
        flowStatus:        p.flow_status ?? 'unknown',
        flowBandLabel:     p.flow_band_label ?? null,
        lastReadingAt:     p.last_reading_at ?? null,
      } satisfies Omit<WatchedGauge, 'watchState' | 'activeSince'>)
    }
  }

  /**
   * Pushes all locally-stored gauge IDs to the server. Called after login to
   * ensure gauges added while anonymous are persisted to the user's account.
   */
  async function pushLocalToServer() {
    const token = await getToken()
    if (!token) return
    for (const gauge of store.gauges) {
      fetch(`${apiBase}/api/v1/watchlist`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ gauge_id: gauge.id }),
      }).catch(() => {})
    }
  }

  return { addAndSync, removeAndSync, loadFromServer, pushLocalToServer }
}
