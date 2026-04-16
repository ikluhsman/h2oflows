<template>
  <!-- ─── LIST density ────────────────────────────────────────────────────── -->
  <div
    v-if="density === 'list'"
    class="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden"
  >
    <!-- Gauge station header row -->
    <div
      class="flex items-center gap-2 sm:gap-3 px-3 py-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
      @click="$emit('open', leadGauge)"
    >
      <div class="min-w-0 flex-1 flex items-center gap-1">
        <svg class="w-3 h-3 text-gray-300 dark:text-gray-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-label="Gauge">
          <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
          <path d="M12 12 16 8"/>
          <path d="M3 12a9 9 0 0 1 18 0"/>
        </svg>
        <span class="text-[10px] uppercase tracking-wider font-medium text-gray-400 dark:text-gray-500 truncate block">{{ gaugeName }}</span>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <div class="w-28 shrink-0 hidden sm:block opacity-50">
          <GaugeSparkline
            :gauge-id="leadGauge.id"
            :flow-status="displayFlowStatus"
            :flow-band-label="displayFlowBand"
            :reach-slug="leadGauge.contextReachSlug ?? leadGauge.reachSlug"
            compact
            @latest-cfs="liveCfs = $event"
            @live-flow-band="liveFlowBand = $event"
          />
        </div>
        <span class="text-base font-bold tabular-nums min-w-[3.5rem] text-right text-gray-900 dark:text-white">
          {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
          <span class="text-xs font-normal text-gray-400 dark:text-gray-500">cfs</span>
        </span>
      </div>
    </div>

    <!-- Reach sub-rows -->
    <div v-if="reachItems.length > 0" class="border-t border-gray-100 dark:border-gray-800">
      <div
        v-for="item in reachItems"
        :key="item.contextReachSlug!"
        class="flex items-center gap-2 pl-5 pr-3 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800/30 transition-colors group border-b border-gray-100/50 dark:border-gray-800/50 last:border-b-0"
      >
        <svg class="w-3.5 h-3.5 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-label="Reach">
          <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
          <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
        </svg>
        <NuxtLink
          :to="`/reaches/${item.contextReachSlug}`"
          class="flex-1 min-w-0 text-sm text-blue-600 dark:text-blue-400 truncate hover:underline"
          @click.stop
        >{{ item.contextReachCommonName ?? item.contextReachFullName ?? item.name }}</NuxtLink>
        <span
          v-if="item.flowStatus !== 'unknown' || item.flowBandLabel"
          :class="['inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium shrink-0', flowBandBadgeClass(item.flowBandLabel, item.flowStatus)]"
        >{{ flowBandLabel(item.flowBandLabel, item.flowStatus) }}</span>
        <button
          class="rounded p-1 text-gray-300 dark:text-gray-600 hover:text-red-400 transition-colors shrink-0"
          aria-label="Remove"
          @click.stop="removeAndSync(item.id, item.contextReachSlug)"
        >
          <svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
            <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/>
          </svg>
        </button>
      </div>
    </div>
    <div v-else class="border-t border-gray-100 dark:border-gray-800 pl-8 pr-3 py-1.5 text-xs text-gray-400 italic">
      No related reaches
    </div>
  </div>

  <!-- ─── CARD densities (compact / comfortable / full) ──────────────────── -->
  <div
    v-else
    class="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden cursor-pointer transition-all duration-200 hover:border-gray-300 dark:hover:border-gray-600"
    @click="$emit('open', leadGauge)"
  >
    <!-- Gauge header section -->
    <div :class="density === 'compact' ? 'p-2.5' : density === 'comfortable' ? 'p-3' : 'p-4'">
      <!-- Gauge name -->
      <div class="flex items-center gap-1" :class="density === 'compact' ? 'mb-0.5' : 'mb-1'">
        <svg class="text-gray-300 dark:text-gray-600 shrink-0" :class="density === 'compact' ? 'w-2.5 h-2.5' : 'w-3 h-3'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-label="Gauge">
          <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
          <path d="M12 12 16 8"/>
          <path d="M3 12a9 9 0 0 1 18 0"/>
        </svg>
        <span
          class="uppercase tracking-wider font-medium text-gray-400 dark:text-gray-500 truncate block leading-tight"
          :class="density === 'compact' ? 'text-[9px]' : 'text-[10px]'"
        >{{ gaugeName }}</span>
      </div>

      <!-- CFS + trend -->
      <div
        class="flex items-baseline gap-2 flex-wrap"
        :class="density === 'compact' ? 'mb-2' : 'mb-1.5'"
      >
        <span
          class="font-bold tabular-nums leading-none text-gray-900 dark:text-white"
          :class="density === 'compact' ? 'text-xl' : density === 'comfortable' ? 'text-2xl' : 'text-3xl'"
        >
          {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
        </span>
        <span class="text-xs text-gray-500">cfs</span>
        <TrendArrow v-if="currentCfs != null && density !== 'compact'" :gauge-id="leadGauge.id" class="text-lg" />
      </div>

      <!-- Sparkline — comfortable: compact; full: full (muted) -->
      <div v-if="density === 'comfortable'" class="opacity-50 mb-1">
        <GaugeSparkline
          :gauge-id="leadGauge.id"
          :flow-status="displayFlowStatus"
          :flow-band-label="displayFlowBand"
          :reach-slug="leadGauge.contextReachSlug ?? leadGauge.reachSlug"
          compact
          @latest-cfs="liveCfs = $event"
          @live-flow-band="liveFlowBand = $event"
        />
      </div>
      <div v-else-if="density === 'full'" class="opacity-50 mb-1.5">
        <GaugeSparkline
          :gauge-id="leadGauge.id"
          :flow-status="displayFlowStatus"
          :flow-band-label="displayFlowBand"
          :reach-slug="leadGauge.contextReachSlug ?? leadGauge.reachSlug"
          @latest-cfs="liveCfs = $event"
          @live-flow-band="liveFlowBand = $event"
        />
      </div>
    </div>

    <!-- Reach sub-list -->
    <div class="border-t border-gray-100 dark:border-gray-800">
      <div
        v-for="item in reachItems"
        :key="item.contextReachSlug!"
        class="flex items-center gap-1.5 px-3 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800/40 transition-colors group border-b border-gray-100/50 dark:border-gray-800/50 last:border-b-0"
        @click.stop
      >
        <svg class="w-3.5 h-3.5 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-label="Reach">
          <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
          <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
        </svg>
        <NuxtLink
          :to="`/reaches/${item.contextReachSlug}`"
          class="flex-1 min-w-0 text-sm text-gray-700 dark:text-gray-300 truncate hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
        >{{ item.contextReachCommonName ?? item.contextReachFullName ?? item.name }}</NuxtLink>
        <span
          v-if="item.flowStatus !== 'unknown' || item.flowBandLabel"
          :class="['inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium shrink-0', flowBandBadgeClass(item.flowBandLabel, item.flowStatus)]"
        >{{ flowBandLabel(item.flowBandLabel, item.flowStatus) }}</span>
        <button
          class="opacity-0 group-hover:opacity-100 rounded p-0.5 text-gray-400 hover:text-red-400 transition-all shrink-0"
          aria-label="Remove"
          @click.stop="removeAndSync(item.id, item.contextReachSlug)"
        >
          <svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
            <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/>
          </svg>
        </button>
      </div>
      <div v-if="reachItems.length === 0" class="px-3 py-1.5 text-xs text-gray-400 italic">
        No related reaches
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'

const props = defineProps<{
  leadGauge: WatchedGauge
  reachItems: WatchedGauge[]
  density?: 'compact' | 'comfortable' | 'full' | 'list'
}>()

const emit = defineEmits<{ (e: 'open', gauge: WatchedGauge): void }>()

const { removeAndSync } = useWatchlistSync()

const liveCfs      = ref<number | null>(null)
const liveFlowBand = ref<{ flowBandLabel: string | null; flowStatus: string } | null>(null)

const currentCfs        = computed(() => liveCfs.value ?? props.leadGauge.currentCfs)
const displayFlowBand   = computed(() => liveFlowBand.value?.flowBandLabel ?? props.leadGauge.flowBandLabel)
const displayFlowStatus = computed(() => liveFlowBand.value?.flowStatus    ?? props.leadGauge.flowStatus)

const gaugeName = computed(() =>
  props.leadGauge.name ?? `${props.leadGauge.source.toUpperCase()} ${props.leadGauge.externalId}`
)
</script>
