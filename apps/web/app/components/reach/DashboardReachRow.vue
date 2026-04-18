<template>
  <div
    class="flex items-center gap-2 sm:gap-3 px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors group"
  >
    <!-- River wave icon -->
    <svg class="w-3.5 h-3.5 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-label="Reach">
      <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
      <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
    </svg>

    <!-- Reach name — click navigates to detail page -->
    <NuxtLink
      :to="`/reaches/${gauge.contextReachSlug}`"
      class="flex-1 min-w-0 text-sm font-medium text-gray-800 dark:text-gray-200 truncate hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
    >{{ reachName }}</NuxtLink>

    <!-- Gauge name (subtle) -->
    <button
      class="hidden sm:block text-[10px] uppercase tracking-wider text-gray-400 dark:text-gray-500 shrink-0 truncate max-w-[8rem] hover:text-gray-600 dark:hover:text-gray-300 transition-colors cursor-pointer"
      :title="gaugeName"
      @click.stop="$emit('openGauge', gauge)"
    >{{ gaugeName }}</button>

    <!-- Sparkline -->
    <div class="w-20 shrink-0 hidden sm:block opacity-60">
      <GaugeSparkline
        :gauge-id="gauge.id"
        flow-status="unknown"
        color="#3b82f6"
        compact
        @latest-cfs="liveCfs = $event"
      />
    </div>

    <!-- Flow badge + CFS — click opens gauge modal -->
    <button
      class="flex items-center gap-1.5 shrink-0 rounded-md px-1.5 py-0.5 hover:bg-gray-100 dark:hover:bg-gray-700/50 transition-colors cursor-pointer"
      @click.stop="$emit('openGauge', gauge)"
    >
      <span
        v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel"
        :class="['inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
      >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
      <span class="text-sm font-bold tabular-nums text-gray-900 dark:text-white">
        {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
      </span>
      <span class="text-xs text-gray-400">cfs</span>
    </button>

    <!-- Remove button -->
    <button
      class="rounded p-1 text-gray-300 dark:text-gray-600 hover:text-red-400 transition-colors shrink-0 opacity-0 group-hover:opacity-100"
      aria-label="Remove"
      @click.stop="removeAndSync(gauge.id, gauge.contextReachSlug)"
    >
      <svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
        <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/>
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { flowBandBadgeClass, flowBandLabel } from '~/utils/flowBand'

const props = defineProps<{ gauge: WatchedGauge }>()

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
</script>
