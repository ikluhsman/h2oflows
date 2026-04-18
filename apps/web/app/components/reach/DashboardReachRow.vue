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
        <span class="flex-1 min-w-0 text-sm font-medium text-gray-800 dark:text-gray-200 truncate">{{ reachName }}</span>
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
        <button
          class="p-1 rounded-lg text-gray-300 dark:text-gray-600 hover:text-red-400 transition-colors shrink-0"
          aria-label="Remove"
          @click.stop="removeAndSync(gauge.id, gauge.contextReachSlug)"
        >
          <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
            <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/>
          </svg>
        </button>
      </div>
    </template>

    <!-- ── COMPACT mode: two-row card, sparkline right-side ───────────────── -->
    <template v-else-if="view === 'compact'">
      <div class="flex items-center gap-3 px-4 pt-3.5 pb-1.5">
        <svg class="w-4 h-4 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-hidden="true">
          <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
          <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
        </svg>
        <span class="flex-1 min-w-0 text-sm font-semibold text-gray-900 dark:text-white truncate">{{ reachName }}</span>
        <div class="shrink-0 text-right leading-none">
          <span class="text-[22px] font-bold tabular-nums" :class="cfsColorClass">
            {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-400 ml-0.5">cfs</span>
        </div>
      </div>
      <div class="flex items-center gap-2 px-4 pb-3">
        <span class="text-[10px] font-medium text-gray-400 dark:text-gray-500 uppercase tracking-wider shrink-0 max-w-24 truncate">{{ gaugeName }}</span>
        <div class="flex-1 min-w-0" />
        <!-- Sparkline: fixed width right side, pointer-events-none -->
        <div class="w-24 shrink-0 h-5 opacity-50 pointer-events-none">
          <GaugeSparkline :gauge-id="gauge.id" flow-status="unknown" :color="sparklineColor" compact @latest-cfs="liveCfs = $event" />
        </div>
        <span
          v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
          :class="['shrink-0 inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
        >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
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
          class="p-1 rounded-lg text-gray-300 dark:text-gray-600 hover:text-red-400 transition-colors shrink-0"
          aria-label="Remove"
          @click.stop="removeAndSync(gauge.id, gauge.contextReachSlug)"
        >
          <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
            <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/>
          </svg>
        </button>
      </div>
    </template>

    <!-- ── FULL mode: two-row card + taller sparkline ─────────────────────── -->
    <template v-else>
      <div class="flex items-center gap-3 px-4 pt-3.5 pb-1.5">
        <svg class="w-4 h-4 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-hidden="true">
          <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
          <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
        </svg>
        <div class="flex-1 min-w-0">
          <div class="text-sm font-semibold text-gray-900 dark:text-white truncate">{{ reachName }}</div>
          <div v-if="gauge.contextReachRiverName" class="text-[10px] text-blue-500/80 dark:text-blue-400/70 uppercase tracking-wide font-medium truncate mt-0.5">
            {{ gauge.contextReachRiverName }}
          </div>
        </div>
        <div class="shrink-0 text-right leading-none">
          <span class="text-[22px] font-bold tabular-nums" :class="cfsColorClass">
            {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-400 ml-0.5">cfs</span>
        </div>
      </div>
      <!-- Taller sparkline row, pointer-events-none -->
      <div class="px-4 pb-1.5 pointer-events-none">
        <div class="h-10 opacity-50">
          <GaugeSparkline :gauge-id="gauge.id" flow-status="unknown" :color="sparklineColor" @latest-cfs="liveCfs = $event" />
        </div>
      </div>
      <div class="flex items-center gap-2 px-4 pb-3">
        <span class="text-[10px] font-medium text-gray-400 dark:text-gray-500 uppercase tracking-wider shrink-0 max-w-24 truncate">{{ gaugeName }}</span>
        <span
          v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
          :class="['shrink-0 inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
        >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
        <span v-if="lastReadingRelative" class="flex-1 text-[10px] text-gray-400 truncate text-right">{{ lastReadingRelative }}</span>
        <div v-else class="flex-1" />
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
          class="p-1 rounded-lg text-gray-300 dark:text-gray-600 hover:text-red-400 transition-colors shrink-0"
          aria-label="Remove"
          @click.stop="removeAndSync(gauge.id, gauge.contextReachSlug)"
        >
          <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
            <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/>
          </svg>
        </button>
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

defineEmits<{ (e: 'openGauge', gauge: WatchedGauge): void }>()

const { removeAndSync } = useWatchlistSync()

const liveCfs = ref<number | null>(null)
const displayCfs = computed(() => liveCfs.value ?? props.gauge.currentCfs)

const reachName = computed(() =>
  props.gauge.contextReachCommonName
    ?? props.gauge.contextReachFullName
    ?? props.gauge.reachName
    ?? props.gauge.name
    ?? props.gauge.externalId
)

const gaugeName = computed(() =>
  props.gauge.name ?? `${props.gauge.source.toUpperCase()} ${props.gauge.externalId}`
)

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
