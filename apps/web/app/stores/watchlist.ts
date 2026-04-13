import { defineStore } from 'pinia'

// A gauge the user has saved to their dashboard.
// A watchlist item is identified by (id, contextReachSlug) — the same gauge
// can appear more than once when added in the context of different reaches
// (e.g. PLAGRACO for Bailey and PLAGRACO for Foxton are separate items).
export interface WatchedGauge {
  id: string
  externalId: string
  source: string
  name: string | null
  // Reach context — the specific reach this watchlist item is scoped to.
  // null = standalone gauge add with no reach context.
  contextReachSlug: string | null
  contextReachCommonName: string | null  // e.g. "Foxton"
  contextReachFullName: string | null   // e.g. "Buffalo Creek to South Platte"
  contextReachRiverName: string | null  // e.g. "South Platte River"
  // All reaches associated with this gauge (for informational display)
  reachId: string | null
  reachName: string | null          // combined display string e.g. "Bailey / Foxton"
  reachNames: string[]              // individual reach names, parallel to reachSlugs
  reachSlug: string | null
  reachSlugs: string[]              // all reaches that use this gauge as primary
  reachCommonNames: string[]        // common names parallel to reachSlugs
  reachRelationship: string | null  // primary | upstream_indicator | downstream_indicator | tributary
  watershedName: string | null
  basinName: string | null
  riverName: string | null
  stateAbbr: string | null
  // Gauge location — populated from the batch API on dashboard load
  lat: number | null
  lng: number | null
  // Latest reading — refreshed by the dashboard poller
  currentCfs: number | null
  // Flow status resolved against the context reach's flow ranges.
  // For standalone gauges (no contextReachSlug), uses the alphabetically-first reach.
  flowStatus: 'runnable' | 'caution' | 'low' | 'flood' | 'unknown'
  // Named flow band from flow_ranges (e.g. "optimal", "fun") — null if no ranges seeded
  flowBandLabel: string | null
  lastReadingAt: string | null
  // Watch state — kept for trip recorder removal in Phase 8
  watchState: 'saved' | 'active'
  activeSince: string | null
}

export const useWatchlistStore = defineStore('watchlist', {
  state: () => ({
    gauges: [] as WatchedGauge[],
    // Active trip — kept for trip recorder removal in Phase 8
    activeTrip: null as {
      gaugeId: string
      reachSlug: string | null
      reachName: string | null
      startedAt: string
      startCfs: number | null
    } | null,
  }),

  getters: {
    // Gauges grouped by context reach for the main dashboard layout.
    // Groups by contextReachSlug; standalone gauges fall into the null bucket last.
    // Migrated gauges without contextReachSlug are treated as standalone (null).
    byReach(state): { reach: string | null; gauges: WatchedGauge[] }[] {
      const map = new Map<string | null, WatchedGauge[]>()
      for (const g of state.gauges) {
        const key = g.contextReachSlug ?? null
        if (!map.has(key)) map.set(key, [])
        map.get(key)!.push(g)
      }
      // Named reaches first (insertion order), null/standalone last
      const result: { reach: string | null; gauges: WatchedGauge[] }[] = []
      for (const [reach, gauges] of map) {
        if (reach !== null) result.push({ reach, gauges })
      }
      if (map.has(null)) result.push({ reach: null, gauges: map.get(null)! })
      return result
    },

    // Gauges grouped by river name for the main dashboard layout.
    byRiver(state): { river: string | null; gauges: WatchedGauge[] }[] {
      const map = new Map<string | null, WatchedGauge[]>()
      for (const g of state.gauges) {
        const key = g.contextReachRiverName ?? g.riverName ?? g.watershedName ?? null
        if (!map.has(key)) map.set(key, [])
        map.get(key)!.push(g)
      }
      const named: { river: string; gauges: WatchedGauge[] }[] = []
      for (const [river, gauges] of map) {
        if (river !== null) named.push({ river, gauges })
      }
      named.sort((a, b) => a.river.localeCompare(b.river))
      const result: { river: string | null; gauges: WatchedGauge[] }[] = [...named]
      if (map.has(null)) result.push({ river: null, gauges: map.get(null)! })
      return result
    },

    // Gauges grouped by watershed — used by the aggregate graph picker
    byWatershed(state): Record<string, WatchedGauge[]> {
      return state.gauges.reduce((acc, g) => {
        const key = g.watershedName ?? 'Other'
        if (!acc[key]) acc[key] = []
        acc[key].push(g)
        return acc
      }, {} as Record<string, WatchedGauge[]>)
    },

    activeGauge(state): WatchedGauge | undefined {
      return state.gauges.find(g => g.watchState === 'active')
    },

    hasActiveTrip(state): boolean {
      return state.activeTrip !== null
    },
  },

  actions: {
    // Identity: (id, contextReachSlug). The same gauge UUID can exist multiple
    // times with different reach contexts. null and undefined both mean standalone.
    addGauge(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
      const slug = gauge.contextReachSlug ?? null
      if (this.gauges.find(g => g.id === gauge.id && (g.contextReachSlug ?? null) === slug)) return
      // If adding with a reach context, remove any standalone version of the same gauge
      if (slug) this.gauges = this.gauges.filter(g => !(g.id === gauge.id && !g.contextReachSlug))
      this.gauges.push({ ...gauge, contextReachSlug: slug, watchState: 'saved', activeSince: null })
    },

    // Remove exact duplicates that may have leaked through localStorage migration or race conditions
    deduplicate() {
      const seen = new Set<string>()
      this.gauges = this.gauges.filter(g => {
        const key = `${g.id}::${g.contextReachSlug ?? ''}`
        if (seen.has(key)) return false
        seen.add(key)
        return true
      })
    },

    removeGauge(gaugeId: string, contextReachSlug?: string | null) {
      // If removing a specific (gauge, reach) pair
      const slug = contextReachSlug !== undefined ? (contextReachSlug ?? null) : null
      if (contextReachSlug !== undefined) {
        this.gauges = this.gauges.filter(g => !(g.id === gaugeId && (g.contextReachSlug ?? null) === slug))
      } else {
        // Legacy: remove all entries for this gauge ID
        this.gauges = this.gauges.filter(g => g.id !== gaugeId)
      }
    },

    // Transition a gauge from 'saved' → 'active' and start trip recording.
    startTrip(gaugeId: string) {
      const gauge = this.gauges.find(g => g.id === gaugeId)
      if (!gauge) return
      if (this.activeGauge && this.activeGauge.id !== gaugeId) this.endTrip()
      gauge.watchState = 'active'
      gauge.activeSince = new Date().toISOString()
      this.activeTrip = {
        gaugeId,
        reachSlug: gauge.contextReachSlug,
        reachName: gauge.contextReachCommonName ?? gauge.reachName,
        startedAt: gauge.activeSince,
        startCfs: gauge.currentCfs,
      }
    },

    endTrip() {
      const active = this.activeGauge
      if (!active) return
      active.watchState = 'saved'
      active.activeSince = null
      this.activeTrip = null
    },

    updateReading(gaugeId: string, cfs: number, flowStatus: WatchedGauge['flowStatus'], readingAt: string) {
      const gauge = this.gauges.find(g => g.id === gaugeId)
      if (!gauge) return
      gauge.currentCfs = cfs
      gauge.flowStatus = flowStatus
      gauge.lastReadingAt = readingAt
    },

    // Refresh gauge metadata from the API. Matched by (id, contextReachSlug).
    // The API echoes back context_reach_slug so we can find the right watchlist item.
    refreshFromApi(fresh: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
      const slug = fresh.contextReachSlug ?? null
      const gauge = this.gauges.find(g => g.id === fresh.id && (g.contextReachSlug ?? null) === slug)
      if (!gauge) return
      gauge.name                  = fresh.name
      gauge.contextReachSlug      = slug
      gauge.contextReachCommonName = fresh.contextReachCommonName ?? null
      gauge.contextReachFullName  = fresh.contextReachFullName ?? null
      gauge.contextReachRiverName = fresh.contextReachRiverName ?? null
      gauge.reachId               = fresh.reachId
      gauge.reachName             = fresh.reachName
      gauge.reachNames            = fresh.reachNames
      gauge.reachSlug             = fresh.reachSlug
      gauge.reachSlugs            = fresh.reachSlugs
      gauge.reachCommonNames      = fresh.reachCommonNames ?? []
      gauge.reachRelationship     = fresh.reachRelationship
      gauge.watershedName         = fresh.watershedName
      gauge.basinName             = fresh.basinName
      gauge.riverName             = fresh.riverName
      gauge.stateAbbr             = fresh.stateAbbr
      gauge.lat                   = fresh.lat
      gauge.lng                   = fresh.lng
      gauge.currentCfs            = fresh.currentCfs
      gauge.flowStatus            = fresh.flowStatus
      gauge.flowBandLabel         = fresh.flowBandLabel
      gauge.lastReadingAt         = fresh.lastReadingAt
    },
  },

  persist: true,
})
