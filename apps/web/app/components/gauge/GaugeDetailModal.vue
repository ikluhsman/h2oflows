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
            {{ gauge.currentCfs != null ? gauge.currentCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-sm text-gray-500">cfs</span>
          <TrendArrow v-if="gauge.currentCfs != null" :gauge-id="gauge.id" class="text-lg" />
          <span
            v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
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

        <!-- 48-hour graph -->
        <GaugeGraph
          :gauge-id="gauge.id"
          :current-cfs="gauge.currentCfs"
          :reach-slug="gauge.contextReachSlug ?? gauge.reachSlug"
        />

        <!-- Last updated -->
        <p v-if="gauge.lastReadingAt" class="text-xs text-gray-500">
          Last reading {{ lastReadingRelative }}
        </p>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { useDiurnalCache } from '~/composables/useDiurnalCache'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ gauge: WatchedGauge }>()

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

const sourceUrl = computed(() => {
  switch (props.gauge.source) {
    case 'usgs': return `https://waterdata.usgs.gov/monitoring-location/${props.gauge.externalId}/`
    case 'dwr':  return `https://dwr.state.co.us/Tools/Stations/${props.gauge.externalId}`
    default:     return '#'
  }
})

const statusBadgeClass = computed(() => {
  const band = props.gauge.flowBandLabel
  if (band === 'below_recommended') return 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400'
  if (band === 'low_runnable')      return 'bg-lime-100 dark:bg-lime-950/50 text-lime-700 dark:text-lime-400'
  if (band === 'runnable')          return 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400'
  if (band === 'med_runnable')      return 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400'
  if (band === 'high_runnable')     return 'bg-green-100 dark:bg-green-950/50 text-green-700 dark:text-green-500'
  if (band === 'above_recommended') return 'bg-blue-100 dark:bg-blue-950/50 text-blue-700 dark:text-blue-400'
  switch (props.gauge.flowStatus) {
    case 'runnable': return 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400'
    case 'caution':  return 'bg-amber-100 dark:bg-amber-950/50 text-amber-700 dark:text-amber-400'
    case 'low':      return 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400'
    case 'flood':    return 'bg-blue-100 dark:bg-blue-950/50 text-blue-700 dark:text-blue-400'
    default:         return 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
  }
})

const statusLabel = computed(() => {
  if (props.gauge.flowBandLabel) {
    return props.gauge.flowBandLabel
      .split('_')
      .map(w => w.charAt(0).toUpperCase() + w.slice(1))
      .join(' ')
  }
  switch (props.gauge.flowStatus) {
    case 'runnable': return 'Runnable'
    case 'caution':  return 'Minimum'
    case 'low':      return 'Too Low'
    case 'flood':    return 'Flood Stage'
    default:         return 'Unknown'
  }
})

const cfsClass = computed(() => {
  const band = props.gauge.flowBandLabel
  if (band === 'low_runnable')  return 'text-lime-500'
  if (band === 'med_runnable')  return 'text-emerald-500'
  if (band === 'high_runnable') return 'text-green-600 dark:text-green-500'
  return {
    'text-emerald-400 dark:text-emerald-500': props.gauge.flowStatus === 'runnable',
    'text-amber-400':                         props.gauge.flowStatus === 'caution',
    'text-red-400':                           props.gauge.flowStatus === 'low',
    'text-blue-400 dark:text-blue-500':       props.gauge.flowStatus === 'flood',
    'text-gray-400':                          !props.gauge.flowStatus || props.gauge.flowStatus === 'unknown',
  }
})

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
