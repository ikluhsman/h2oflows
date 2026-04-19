<template>
  <div
    class="rounded-2xl border border-gray-200 dark:border-gray-700/60 bg-white dark:bg-gray-900 shadow-sm overflow-hidden cursor-pointer active:opacity-80 transition-opacity"
    @click="$emit('openGauge', gauge)"
  >
    <!-- ── LIST mode: single compact row ──────────────────────────────────── -->
    <template v-if="view === 'list'">
      <div class="flex items-center gap-2 px-3 py-2.5">
        <!-- Reach name + river name + link button (link lives next to name) -->
        <div class="flex items-center gap-1 min-w-0 flex-1">
          <div class="min-w-0">
            <span class="text-sm font-medium text-gray-800 dark:text-gray-200 truncate block">{{ reachName }}</span>
            <span v-if="riverDisplayName && !hideRiverName" class="text-xs text-gray-400 dark:text-gray-500 truncate block leading-tight">{{ riverDisplayName }}</span>
          </div>
          <NuxtLink
            :to="`/reaches/${gauge.contextReachSlug}`"
            class="shrink-0 p-1 rounded text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors"
            aria-label="View reach detail"
            @click.stop
          >
            <svg class="w-3 h-3" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11"/>
            </svg>
          </NuxtLink>
        </div>
        <!-- Sparkline: fixed width, desktop only, pointer-events-none so card click passes through -->
        <div class="w-24 shrink-0 hidden sm:block h-5 opacity-50 pointer-events-none">
          <GaugeSparkline :gauge-id="gauge.id" flow-status="unknown" :color="sparklineColor" compact @latest-cfs="liveCfs = $event" />
        </div>
        <!-- Fixed-width badge slot keeps CFS column aligned across cards -->
        <div class="shrink-0 w-20 flex justify-end">
          <span
            v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
            :class="['inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
          >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
        </div>
        <span class="text-sm font-bold tabular-nums shrink-0 w-16 text-right" :class="cfsColorClass">
          {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          <span class="text-xs font-normal text-gray-400">cfs</span>
        </span>
        <button
          class="shrink-0 p-1.5 rounded-lg text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
          aria-label="Remove from dashboard"
          @click.stop="$emit('removeGauge', gauge)"
        >
          <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C9.327 4.025 10 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd"/>
          </svg>
        </button>
      </div>
    </template>

    <!-- ── COMPACT mode: details top, sparkline below ─────────────────────── -->
    <template v-else-if="view === 'compact'">
      <div class="px-4 pt-3 pb-1">
        <div class="flex items-center gap-2">
          <svg class="w-3.5 h-3.5 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-hidden="true">
            <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
            <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
          </svg>
          <!-- Reach name + river name + link (link lives next to name) -->
          <div class="flex items-center gap-1 min-w-0 flex-1">
            <div class="min-w-0">
              <span class="text-sm font-semibold text-gray-900 dark:text-white truncate block leading-tight">{{ reachName }}</span>
              <span v-if="riverDisplayName && !hideRiverName" class="text-xs text-gray-400 dark:text-gray-500 truncate block leading-tight">{{ riverDisplayName }}</span>
            </div>
            <NuxtLink
              :to="`/reaches/${gauge.contextReachSlug}`"
              class="shrink-0 p-1 rounded text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors"
              aria-label="View reach detail"
              @click.stop
            >
              <svg class="w-3 h-3" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11"/>
              </svg>
            </NuxtLink>
          </div>
          <span
            v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
            :class="['shrink-0 inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
          >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
          <span class="text-lg font-bold tabular-nums shrink-0" :class="cfsColorClass">
            {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-400 shrink-0">cfs</span>
          <button
            class="shrink-0 p-1.5 rounded-lg text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
            aria-label="Remove from dashboard"
            @click.stop="$emit('removeGauge', gauge)"
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C9.327 4.025 10 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd"/>
            </svg>
          </button>
        </div>
        <a
          :href="gaugeSourceUrl"
          target="_blank"
          rel="noopener"
          class="text-[11px] font-medium text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300 transition-colors"
          @click.stop
        >{{ gaugeSourceLabel }}</a>
      </div>
      <div class="px-4 pb-2 pointer-events-none opacity-50 h-8">
        <GaugeSparkline :gauge-id="gauge.id" flow-status="unknown" :color="sparklineColor" compact @latest-cfs="liveCfs = $event" />
      </div>
    </template>

    <!-- ── FULL mode: details top, sparkline below ───────────────────────── -->
    <template v-else>
      <div class="px-4 pt-3 pb-1">
        <div class="flex items-center gap-2">
          <svg class="w-3.5 h-3.5 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-hidden="true">
            <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
            <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
          </svg>
          <div class="flex items-center gap-1 min-w-0 flex-1">
            <div class="min-w-0">
              <span class="text-sm font-semibold text-gray-900 dark:text-white truncate block leading-tight">{{ reachName }}</span>
              <span v-if="riverDisplayName && !hideRiverName" class="text-xs text-gray-400 dark:text-gray-500 truncate block leading-tight">{{ riverDisplayName }}</span>
            </div>
            <NuxtLink
              :to="`/reaches/${gauge.contextReachSlug}`"
              class="shrink-0 p-1 rounded text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors"
              aria-label="View reach detail"
              @click.stop
            >
              <svg class="w-3 h-3" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11"/>
              </svg>
            </NuxtLink>
          </div>
          <span
            v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
            :class="['shrink-0 inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
          >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
          <span class="text-[22px] font-bold tabular-nums shrink-0 leading-none" :class="cfsColorClass">
            {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-400 shrink-0">cfs</span>
          <button
            class="shrink-0 p-1.5 rounded-lg text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
            aria-label="Remove from dashboard"
            @click.stop="$emit('removeGauge', gauge)"
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C9.327 4.025 10 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd"/>
            </svg>
          </button>
        </div>
        <div class="flex items-center gap-3 mt-1">
          <a
            :href="gaugeSourceUrl"
            target="_blank"
            rel="noopener"
            class="text-[11px] font-medium text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300 transition-colors truncate"
            @click.stop
          >{{ gaugeSourceLabel }}</a>
          <span v-if="lastReadingRelative" class="text-[10px] text-gray-400 truncate">{{ lastReadingRelative }}</span>
        </div>
      </div>
      <div class="px-4 pb-4 pointer-events-none opacity-50 h-20">
        <GaugeSparkline :gauge-id="gauge.id" flow-status="unknown" :color="sparklineColor" @latest-cfs="liveCfs = $event" />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { flowBandBadgeClass, flowBandLabel } from '~/utils/flowBand'

const props = defineProps<{
  gauge: WatchedGauge
  view?: 'list' | 'compact' | 'full'
  hideRiverName?: boolean
}>()

defineEmits<{
  (e: 'openGauge', gauge: WatchedGauge): void
  (e: 'removeGauge', gauge: WatchedGauge): void
}>()

const liveCfs = ref<number | null>(null)
const displayCfs = computed(() => liveCfs.value ?? props.gauge.currentCfs)

const reachName = computed(() =>
  props.gauge.contextReachCommonName
    ?? props.gauge.contextReachFullName
    ?? props.gauge.reachName
    ?? props.gauge.name
    ?? props.gauge.externalId
)

const riverDisplayName = computed(() =>
  props.gauge.contextReachRiverName ?? props.gauge.riverName ?? null
)

const gaugeSourceLabel = computed(() => {
  const src = props.gauge.source?.toUpperCase() ?? 'USGS'
  return `${src}-${props.gauge.externalId}`
})

const gaugeSourceUrl = computed(() => {
  const id = props.gauge.externalId
  if (props.gauge.source === 'dwr') {
    return `https://dwr.state.co.us/Tools/Stations/${id}`
  }
  return `https://waterdata.usgs.gov/monitoring-location/${id}/`
})

const cfsColorClass = computed(() => {
  const s = props.gauge.flowStatus
  if (s === 'running')                    return 'text-green-600 dark:text-green-400'
  if (s === 'high')                       return 'text-orange-500 dark:text-orange-400'
  if (s === 'very_high' || s === 'flood') return 'text-red-600 dark:text-red-400'
  if (s === 'low')                        return 'text-amber-500 dark:text-amber-400'
  return 'text-gray-900 dark:text-white'
})

const sparklineColor = computed(() => {
  const s = props.gauge.flowStatus
  if (s === 'running')                    return '#22c55e'
  if (s === 'high')                       return '#f97316'
  if (s === 'very_high' || s === 'flood') return '#ef4444'
  if (s === 'low')                        return '#f59e0b'
  return '#3b82f6'
})

const lastReadingRelative = computed(() => {
  const t = props.gauge.lastReadingAt
  if (!t) return ''
  const ms = Date.now() - new Date(t).getTime()
  const m = Math.floor(ms / 60_000)
  if (m < 1)  return 'just now'
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ${m % 60}m ago`
})
</script>
