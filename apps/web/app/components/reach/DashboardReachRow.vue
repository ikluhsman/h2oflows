<template>
  <div
    class="rounded-2xl border border-gray-200 dark:border-gray-700/60 bg-white dark:bg-gray-900 shadow-sm overflow-hidden cursor-pointer active:opacity-80 transition-opacity"
    @click="$emit('openGauge', gauge)"
  >
    <!-- ── LIST mode: single compact row ──────────────────────────────────── -->
    <template v-if="view === 'list'">
      <div class="flex items-center gap-3 px-3 py-2.5">
        <svg class="w-3.5 h-3.5 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-hidden="true">
          <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
          <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
        </svg>
        <div class="flex-1 min-w-0">
          <span class="text-sm font-medium text-gray-800 dark:text-gray-200 truncate block">{{ reachName }}</span>
          <span v-if="riverDisplayName" class="text-xs text-gray-400 dark:text-gray-500 truncate block leading-tight">{{ riverDisplayName }}</span>
        </div>
        <!-- Sparkline: fixed width, desktop only, pointer-events-none so card click passes through -->
        <div class="w-24 shrink-0 hidden sm:block h-5 opacity-50 pointer-events-none">
          <GaugeSparkline :gauge-id="gauge.id" flow-status="unknown" :color="sparklineColor" compact @latest-cfs="liveCfs = $event" />
        </div>
        <span
          v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
          :class="['shrink-0 inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
        >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
        <span class="text-sm font-bold tabular-nums shrink-0" :class="cfsColorClass">
          {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          <span class="text-xs font-normal text-gray-400">cfs</span>
        </span>
        <NuxtLink
          :to="`/reaches/${gauge.contextReachSlug}`"
          class="p-1 rounded-lg text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors shrink-0"
          aria-label="View reach detail"
          @click.stop
        >
          <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11"/>
          </svg>
        </NuxtLink>
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
          <div class="flex-1 min-w-0">
            <span class="text-sm font-semibold text-gray-900 dark:text-white truncate block leading-tight">{{ reachName }}</span>
            <span v-if="riverDisplayName" class="text-xs text-gray-400 dark:text-gray-500 truncate block leading-tight">{{ riverDisplayName }}</span>
          </div>
          <span
            v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
            :class="['shrink-0 inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
          >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
          <span class="text-lg font-bold tabular-nums shrink-0" :class="cfsColorClass">
            {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-400 shrink-0">cfs</span>
          <NuxtLink
            :to="`/reaches/${gauge.contextReachSlug}`"
            class="p-1 rounded-lg text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors shrink-0"
            aria-label="View reach detail"
            @click.stop
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11"/>
            </svg>
          </NuxtLink>
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
          <div class="flex-1 min-w-0">
            <span class="text-sm font-semibold text-gray-900 dark:text-white truncate block leading-tight">{{ reachName }}</span>
            <span v-if="riverDisplayName" class="text-xs text-gray-400 dark:text-gray-500 truncate block leading-tight">{{ riverDisplayName }}</span>
          </div>
          <span
            v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
            :class="['shrink-0 inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
          >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
          <span class="text-[22px] font-bold tabular-nums shrink-0 leading-none" :class="cfsColorClass">
            {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-400 shrink-0">cfs</span>
          <NuxtLink
            :to="`/reaches/${gauge.contextReachSlug}`"
            class="p-1 rounded-lg text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors shrink-0"
            aria-label="View reach detail"
            @click.stop
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11"/>
            </svg>
          </NuxtLink>
          <button
            class="p-1 rounded-lg text-gray-300 dark:text-gray-600 hover:text-red-500 dark:hover:text-red-400 transition-colors shrink-0"
            aria-label="Remove from dashboard"
            @click.stop="$emit('removeGauge', gauge)"
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="5" y1="5" x2="15" y2="15"/><line x1="15" y1="5" x2="5" y2="15"/>
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
