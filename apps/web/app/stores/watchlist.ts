import { defineStore } from 'pinia'

// A gauge the user has saved to their dashboard.
export interface WatchedGauge {
  id: string
  externalId: string
  source: string
  name: string | null
  reachId: string | null
  reachName: string | null          // combined display string e.g. "Bailey / Foxton"
  reachNames: string[]              // individual reach names, parallel to reachSlugs
  reachSlug: string | null
  reachSlugs: string[]              // all reaches that use this gauge as primary
  reachRelationship: string | null  // primary | upstream_indicator | downstream_indicator | tributary
  featured: boolean
  pollTier: 'trusted' | 'demand' | 'cold'
  watershedName: string | null
  basinName: string | null
  riverName: string | null
  stateAbbr: string | null
  // Gauge location — populated from the batch API on dashboard load
  lat: number | null
  lng: number | null
  // Latest reading — refreshed by the dashboard poller
  currentCfs: number | null
  flowStatus: 'runnable' | 'caution' | 'low' | 'flood' | 'unknown'
  // Named flow band from flow_ranges (e.g. "optimal", "fun") — null if no ranges seeded
  flowBandLabel: string | null
  lastReadingAt: string | null
  // Watch state
  watchState: 'saved' | 'active'
  // When the active trip was started (ISO string)
  activeSince: string | null
}

export const useWatchlistStore = defineStore('watchlist', {
  state: () => ({
    gauges: [] as WatchedGauge[],
    // Active trip — set when any gauge transitions to 'active' watch state
    activeTrip: null as {
      gaugeId: string
      reachSlug: string | null
      reachName: string | null
      startedAt: string
      startCfs: number | null
    } | null,
  }),

  getters: {
    // Gauges grouped by reach name for the main dashboard layout.
    // Gauges without a reach fall into the null bucket, rendered last as "Other Gauges".
    byReach(state): { reach: string | null; gauges: WatchedGauge[] }[] {
      const map = new Map<string | null, WatchedGauge[]>()
      for (const g of state.gauges) {
        const key = g.reachName ?? null
        if (!map.has(key)) map.set(key, [])
        map.get(key)!.push(g)
      }
      // Sort: named reaches first (insertion order), null last
      const result: { reach: string | null; gauges: WatchedGauge[] }[] = []
      for (const [reach, gauges] of map) {
        if (reach !== null) result.push({ reach, gauges })
      }
      if (map.has(null)) result.push({ reach: null, gauges: map.get(null)! })
      return result
    },

    // Gauges grouped by river/watershed — used by the dashboard.
    // Named rivers sorted alphabetically; ungrouped gauges fall into "Other" at the end.
    byRiver(state): { river: string; gauges: WatchedGauge[] }[] {
      const map = new Map<string, WatchedGauge[]>()
      for (const g of state.gauges) {
        const key = g.watershedName ?? 'Other'
        if (!map.has(key)) map.set(key, [])
        map.get(key)!.push(g)
      }
      const named = [...map.entries()]
        .filter(([k]) => k !== 'Other')
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([river, gauges]) => ({ river, gauges }))
      const other = map.get('Other')
      return other ? [...named, { river: 'Other', gauges: other }] : named
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
    addGauge(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
      if (this.gauges.find(g => g.id === gauge.id)) return
      this.gauges.push({ ...gauge, watchState: 'saved', activeSince: null })
    },

    removeGauge(gaugeId: string) {
      // If this gauge had an active trip, end it first
      if (this.activeGauge?.id === gaugeId) {
        this.endTrip()
      }
      this.gauges = this.gauges.filter(g => g.id !== gaugeId)
    },

    // Transition a gauge from 'saved' → 'active' and start trip recording.
    // Only one gauge can be active at a time.
    startTrip(gaugeId: string) {
      const gauge = this.gauges.find(g => g.id === gaugeId)
      if (!gauge) return

      // End any existing active trip first
      if (this.activeGauge && this.activeGauge.id !== gaugeId) {
        this.endTrip()
      }

      gauge.watchState = 'active'
      gauge.activeSince = new Date().toISOString()

      this.activeTrip = {
        gaugeId,
        reachSlug: gauge.reachSlug,
        reachName: gauge.reachName,
        startedAt: gauge.activeSince,
        startCfs: gauge.currentCfs,
      }
    },

    // Transition back to 'saved' and close the trip record.
    // The caller is responsible for handing off the trip data to the upload queue.
    endTrip() {
      const active = this.activeGauge
      if (!active) return
      active.watchState = 'saved'
      active.activeSince = null
      this.activeTrip = null
    },

    // Update the latest reading for a gauge (called by the dashboard refresh).
    updateReading(gaugeId: string, cfs: number, flowStatus: WatchedGauge['flowStatus'], readingAt: string) {
      const gauge = this.gauges.find(g => g.id === gaugeId)
      if (!gauge) return
      gauge.currentCfs = cfs
      gauge.flowStatus = flowStatus
      gauge.lastReadingAt = readingAt
    },

    // Refresh gauge metadata from the API (watershed, reach name, current cfs, etc.)
    // Called on dashboard mount to sync persisted watchlist with fresh server data.
    refreshFromApi(fresh: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
      const gauge = this.gauges.find(g => g.id === fresh.id)
      if (!gauge) return
      gauge.name          = fresh.name
      gauge.featured      = fresh.featured
      gauge.reachId           = fresh.reachId
      gauge.reachName         = fresh.reachName
      gauge.reachNames        = fresh.reachNames
      gauge.reachSlug         = fresh.reachSlug
      gauge.reachSlugs        = fresh.reachSlugs
      gauge.reachRelationship = fresh.reachRelationship
      gauge.pollTier      = fresh.pollTier
      gauge.watershedName = fresh.watershedName
      gauge.basinName     = fresh.basinName
      gauge.riverName     = fresh.riverName
      gauge.stateAbbr     = fresh.stateAbbr
      gauge.lat           = fresh.lat
      gauge.lng           = fresh.lng
      gauge.currentCfs    = fresh.currentCfs
      gauge.flowStatus    = fresh.flowStatus
      gauge.flowBandLabel = fresh.flowBandLabel
      gauge.lastReadingAt = fresh.lastReadingAt
    },
  },

  // Persist to localStorage so the watchlist survives page reloads.
  // On mobile this will eventually move to IndexedDB via a Capacitor plugin.
  persist: true,
})
