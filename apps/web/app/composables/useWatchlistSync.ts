import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'

/**
 * useWatchlistSync — wraps watchlist add/remove with server persistence.
 *
 * - When authenticated, changes are mirrored to /api/v1/watchlist immediately.
 * - When anonymous, only localStorage is updated (existing behaviour).
 * - loadFromServer() merges the server watchlist into the local store.
 * - pushLocalToServer() syncs any locally-saved gauges up to the server
 *   (called once after login to handle gauges added while anonymous).
 */
export function useWatchlistSync() {
  const store = useWatchlistStore()
  const { apiBase } = useRuntimeConfig().public
  const { isAuthenticated, getToken } = useAuth()

  async function authedFetch(method: string, path: string, body?: unknown) {
    const token = await getToken()
    if (!token) return null
    return fetch(`${apiBase}${path}`, {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  }

  async function addAndSync(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
    store.addGauge(gauge)
    if (isAuthenticated.value) {
      await authedFetch('POST', '/api/v1/watchlist', { gauge_id: gauge.id }).catch(() => {})
    }
  }

  async function removeAndSync(gaugeId: string) {
    store.removeGauge(gaugeId)
    if (isAuthenticated.value) {
      await authedFetch('DELETE', `/api/v1/watchlist/${gaugeId}`).catch(() => {})
    }
  }

  /**
   * Fetches the server watchlist and adds any gauges not already in the local
   * store. Calls the batch API to hydrate metadata for new entries.
   */
  async function loadFromServer() {
    const res = await authedFetch('GET', '/api/v1/watchlist')
    if (!res?.ok) return
    const data = await res.json()
    const serverIds: string[] = data.gauge_ids ?? []

    const localIds = new Set(store.gauges.map(g => g.id))
    const newIds = serverIds.filter(id => !localIds.has(id))
    if (newIds.length === 0) return

    const token = await getToken()
    const batchRes = await fetch(`${apiBase}/api/v1/gauges/batch?ids=${newIds.join(',')}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!batchRes.ok) return
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
   * The server's ON CONFLICT DO NOTHING makes this safe to call repeatedly.
   */
  async function pushLocalToServer() {
    for (const gauge of store.gauges) {
      await authedFetch('POST', '/api/v1/watchlist', { gauge_id: gauge.id }).catch(() => {})
    }
  }

  return { addAndSync, removeAndSync, loadFromServer, pushLocalToServer }
}
