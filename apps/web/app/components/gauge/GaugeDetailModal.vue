<template>
  <UModal v-model:open="open" :ui="{ width: 'max-w-xl' }">
    <template #header>
      <div class="flex items-start justify-between gap-3 w-full">
        <div class="min-w-0 flex-1">
          <!-- Common name — large, clickable to reach page -->
          <NuxtLink
            v-if="primaryReachSlug"
            :to="`/reaches/${primaryReachSlug}`"
            class="flex items-center gap-1.5 group"
            @click="open = false"
          >
            <span class="text-lg font-bold text-gray-900 dark:text-white truncate group-hover:text-blue-500 transition-colors leading-tight">{{ displayName }}</span>
            <svg class="w-3.5 h-3.5 shrink-0 text-gray-400 group-hover:text-blue-500 transition-colors mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/>
            </svg>
          </NuxtLink>
          <h2 v-else class="text-lg font-bold text-gray-900 dark:text-white truncate leading-tight">{{ displayName }}</h2>
          <!-- Gauge source/id as muted subtext -->
          <p class="text-xs text-gray-400 truncate mt-0.5">
            <a :href="sourceUrl" target="_blank" rel="noopener" class="hover:text-blue-400 underline underline-offset-2">
              {{ gauge.source.toUpperCase() }} · {{ gauge.externalId }}
            </a>
            <span v-if="gauge.watershedName"> · {{ gauge.watershedName }}</span>
          </p>
        </div>
        <!-- Close button only in header -->
        <button
          class="shrink-0 p-1 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          aria-label="Close"
          @click="open = false"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6 6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>
    </template>

    <template #body>
      <div class="space-y-3">
        <!-- CFS + badge + trend arrow — left-aligned, prominent -->
        <div class="flex items-baseline gap-2 flex-wrap">
          <span class="text-3xl font-bold tabular-nums leading-none" :class="cfsClass">
            {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-sm text-gray-500">cfs</span>
          <TrendArrow v-if="displayCfs != null" :gauge-id="gauge.id" class="text-lg" />
          <span
            v-if="displayFlowStatus !== 'unknown' || displayFlowBand"
            :class="['inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', statusBadgeClass]"
          >{{ statusLabel }}</span>
        </div>

        <!-- Diurnal context -->
        <div v-if="diurnal.detected" class="flex items-center gap-1.5 text-xs text-indigo-500 dark:text-indigo-400">
          <svg class="w-3 h-3 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
          <span class="truncate">
            {{ diurnalPhaseLabel }}
            <template v-if="diurnal.forecast"> · {{ diurnal.forecast.label }}</template>
            <template v-if="diurnal.swingPct != null"> · {{ diurnal.swingPct }}% swing</template>
          </span>
        </div>

        <!-- Add / remove from dashboard -->
        <div class="flex items-center gap-2">
          <button
            v-if="isOnDashboard"
            class="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-400 hover:bg-red-50 dark:hover:bg-red-950/40 hover:text-red-600 dark:hover:text-red-400 transition-colors group"
            @click="toggleDashboard"
          >
            <svg class="w-4 h-4 group-hover:hidden" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd"/>
            </svg>
            <svg class="w-4 h-4 hidden group-hover:block" viewBox="0 0 20 20" fill="currentColor">
              <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/>
            </svg>
            <span class="group-hover:hidden">On Dashboard</span>
            <span class="hidden group-hover:inline">Remove</span>
          </button>
          <button
            v-else
            class="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium bg-blue-600 hover:bg-blue-700 text-white transition-colors"
            @click="toggleDashboard"
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z"/>
            </svg>
            Add to Dashboard
          </button>
        </div>

        <!-- 48-hour graph — emits live CFS + band so we can sync display -->
        <GaugeGraph
          :gauge-id="gauge.id"
          :current-cfs="displayCfs"
          :reach-slug="gauge.contextReachSlug ?? gauge.reachSlug"
          @latest-cfs="liveCfs = $event"
          @live-flow-band="liveFlowBand = $event"
        />

        <!-- Last updated -->
        <p v-if="gauge.lastReadingAt" class="text-xs text-gray-500">
          Last reading {{ lastReadingRelative }}
        </p>

        <!-- View reach link -->
        <NuxtLink
          v-if="primaryReachSlug"
          :to="`/reaches/${primaryReachSlug}`"
          class="inline-flex items-center gap-1.5 text-sm font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors"
          @click="open = false"
        >
          View this Reach
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M5 12h14M12 5l7 7-7 7"/>
          </svg>
        </NuxtLink>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { useWatchlistStore } from '~/stores/watchlist'
import { useDiurnalCache } from '~/composables/useDiurnalCache'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ gauge: WatchedGauge }>()

// liveCfs / liveFlowBand are set by GaugeGraph once it loads fresh readings.
// These supersede the potentially-stale values from the watchlist store.
const liveCfs      = ref<number | null>(null)
const liveFlowBand = ref<{ flowBandLabel: string | null; flowStatus: string } | null>(null)
watch(open, (v) => { if (!v) { liveCfs.value = null; liveFlowBand.value = null } })

const displayCfs        = computed(() => liveCfs.value ?? props.gauge.currentCfs)
const displayFlowBand   = computed(() => liveFlowBand.value?.flowBandLabel ?? props.gauge.flowBandLabel)
const displayFlowStatus = computed(() => liveFlowBand.value?.flowStatus   ?? props.gauge.flowStatus)

const { pattern: diurnal } = useDiurnalCache(props.gauge.id)

const diurnalPhaseLabel = computed(() => {
  switch (diurnal.value.phase) {
    case 'rising':      return 'Rising'
    case 'falling':     return 'Falling'
    case 'near_peak':   return 'Near peak'
    case 'near_trough': return 'Near trough'
    case 'stable':      return 'Stable'
    default:            return ''
  }
})

const displayName = computed(() =>
  props.gauge.contextReachCommonName
  ?? props.gauge.reachName
  ?? props.gauge.name
  ?? props.gauge.externalId
)

const primaryReachSlug = computed(() =>
  props.gauge.contextReachSlug ?? props.gauge.reachSlug ?? props.gauge.reachSlugs?.[0] ?? null
)

// ── Dashboard add/remove ──────────────────────────────────────────────────
const watchlistStore = useWatchlistStore()
const { addAndSync, removeAndSync } = useWatchlistSync()

const isOnDashboard = computed(() =>
  watchlistStore.gauges.some(
    g => g.id === props.gauge.id &&
         (g.contextReachSlug ?? null) === (props.gauge.contextReachSlug ?? null)
  )
)

function toggleDashboard() {
  if (isOnDashboard.value) {
    removeAndSync(props.gauge.id, props.gauge.contextReachSlug)
  } else {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { watchState: _ws, activeSince: _as, ...gaugeData } = props.gauge
    addAndSync(gaugeData)
  }
}

const sourceUrl = computed(() => {
  switch (props.gauge.source) {
    case 'usgs': return `https://waterdata.usgs.gov/monitoring-location/${props.gauge.externalId}/`
    case 'dwr':  return `https://dwr.state.co.us/Tools/Stations/${props.gauge.externalId}`
    default:     return '#'
  }
})

const statusBadgeClass = computed(() => flowBandBadgeClass(displayFlowBand.value, displayFlowStatus.value))
const statusLabel      = computed(() => flowBandLabel(displayFlowBand.value, displayFlowStatus.value))
const cfsClass         = computed(() => flowBandCfsClass(displayFlowBand.value, displayFlowStatus.value))

const lastReadingRelative = computed(() => {
  if (!props.gauge.lastReadingAt) return ''
  const ms = Date.now() - new Date(props.gauge.lastReadingAt).getTime()
  const m = Math.floor(ms / 60_000)
  if (m < 1)  return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m ago`
})
</script>
