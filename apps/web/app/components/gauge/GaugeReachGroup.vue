<template>
  <!-- ─── LIST density ────────────────────────────────────────────────────── -->
  <div
    v-if="density === 'list'"
    class="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden"
  >
    <!-- Gauge station header row -->
    <div
      class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
      @click="$emit('open', leadGauge, 'gauge')"
    >
      <div class="min-w-0 flex-1 flex items-center gap-1.5">
        <svg class="w-4 h-4 text-gray-400 dark:text-gray-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-label="Gauge">
          <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
          <path d="M12 12 16 8"/>
          <path d="M3 12a9 9 0 0 1 18 0"/>
        </svg>
        <div class="min-w-0">
          <span class="text-sm font-medium text-gray-600 dark:text-gray-400 truncate block">{{ gaugeName }}</span>
          <span v-if="riverDisplayName && !hideRiverName" class="text-xs text-gray-400 dark:text-gray-500 truncate block leading-tight">{{ riverDisplayName }}</span>
        </div>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <div class="w-28 shrink-0 hidden sm:block opacity-60">
          <GaugeSparkline
            :gauge-id="leadGauge.id"
            flow-status="unknown"
            color="#3b82f6"
            compact
            @latest-cfs="liveCfs = $event"
          />
        </div>
        <span class="w-16 shrink-0 text-right text-base font-bold tabular-nums text-gray-900 dark:text-white">
          {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
        </span>
        <span class="text-xs font-normal text-gray-400 dark:text-gray-500 shrink-0">cfs</span>
        <button
          class="shrink-0 p-1.5 rounded-lg text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
          aria-label="Remove gauge group"
          @click.stop="$emit('remove-group')"
        >
          <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C9.327 4.025 10 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Reach sub-rows -->
    <div v-if="reachItems.length > 0" class="border-t border-gray-100 dark:border-gray-800">
      <div
        v-for="item in reachItems"
        :key="item.contextReachSlug!"
        class="flex items-center gap-2 pl-5 pr-3 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800/30 transition-colors border-b border-gray-100/50 dark:border-gray-800/50 last:border-b-0 cursor-pointer"
        @click.stop="$emit('open', item, 'reach')"
      >
        <!-- Name + link button side-by-side -->
        <div class="flex items-center gap-1 min-w-0 flex-1">
          <span class="min-w-0 text-sm text-gray-700 dark:text-gray-300 truncate">
            {{ item.contextReachCommonName ?? item.contextReachFullName ?? item.name }}
          </span>
          <NuxtLink
            :to="`/reaches/${item.contextReachSlug}`"
            class="shrink-0 p-0.5 rounded text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors"
            aria-label="View reach page"
            title="View reach page"
            @click.stop
          >
            <svg class="w-3 h-3" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </NuxtLink>
        </div>
        <span
          v-if="item.flowStatus !== 'unknown' || item.flowBandLabel"
          :class="['inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold shrink-0', flowBandBadgeClass(item.flowBandLabel, item.flowStatus)]"
        >{{ flowBandLabel(item.flowBandLabel, item.flowStatus) }}</span>
        <button
          class="shrink-0 p-1.5 rounded-lg text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
          aria-label="Remove"
          @click.stop="removeAndSync(item.id, item.contextReachSlug)"
        >
          <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C9.327 4.025 10 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd"/>
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
    @click="$emit('open', leadGauge, 'gauge')"
  >
    <!-- Gauge header section -->
    <div :class="density === 'compact' ? 'p-2.5' : density === 'comfortable' ? 'p-3' : 'p-4'">
      <!-- Gauge name -->
      <div class="flex items-center gap-1.5" :class="density === 'compact' ? 'mb-0.5' : 'mb-1'">
        <svg class="text-gray-400 dark:text-gray-500 shrink-0" :class="density === 'compact' ? 'w-3.5 h-3.5' : 'w-4 h-4'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-label="Gauge">
          <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
          <path d="M12 12 16 8"/>
          <path d="M3 12a9 9 0 0 1 18 0"/>
        </svg>
        <div class="min-w-0">
          <span
            class="font-medium text-gray-600 dark:text-gray-400 truncate block leading-tight"
            :class="density === 'compact' ? 'text-xs' : 'text-sm'"
          >{{ gaugeName }}</span>
          <span v-if="riverDisplayName && !hideRiverName" class="text-xs text-gray-400 dark:text-gray-500 truncate block leading-tight">{{ riverDisplayName }}</span>
        </div>
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

      <!-- Sparkline — comfortable: compact; full: full -->
      <div v-if="density === 'comfortable'" class="opacity-60 mb-1">
        <GaugeSparkline
          :gauge-id="leadGauge.id"
          flow-status="unknown"
          color="#3b82f6"
          compact
          @latest-cfs="liveCfs = $event"
        />
      </div>
      <div v-else-if="density === 'full'" class="opacity-60 mb-1.5">
        <GaugeSparkline
          :gauge-id="leadGauge.id"
          flow-status="unknown"
          color="#3b82f6"
          @latest-cfs="liveCfs = $event"
        />
      </div>
    </div>

    <!-- Reach sub-list -->
    <div class="border-t border-gray-100 dark:border-gray-800">
      <div
        v-for="item in reachItems"
        :key="item.contextReachSlug!"
        class="flex items-center gap-1.5 px-3 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800/40 transition-colors border-b border-gray-100/50 dark:border-gray-800/50 last:border-b-0 cursor-pointer"
        @click.stop="$emit('open', item, 'reach')"
      >
        <!-- Name + link button side-by-side -->
        <div class="flex items-center gap-1 min-w-0 flex-1">
          <span class="min-w-0 text-sm text-gray-700 dark:text-gray-300 truncate">
            {{ item.contextReachCommonName ?? item.contextReachFullName ?? item.name }}
          </span>
          <NuxtLink
            :to="`/reaches/${item.contextReachSlug}`"
            class="shrink-0 p-0.5 rounded text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors"
            aria-label="View reach page"
            title="View reach page"
            @click.stop
          >
            <svg class="w-3 h-3" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </NuxtLink>
        </div>
        <span
          v-if="item.flowStatus !== 'unknown' || item.flowBandLabel"
          :class="['inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold shrink-0', flowBandBadgeClass(item.flowBandLabel, item.flowStatus)]"
        >{{ flowBandLabel(item.flowBandLabel, item.flowStatus) }}</span>
        <button
          class="shrink-0 p-1.5 rounded-lg text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
          aria-label="Remove"
          @click.stop="removeAndSync(item.id, item.contextReachSlug)"
        >
          <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C9.327 4.025 10 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd"/>
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
  hideRiverName?: boolean
}>()

const emit = defineEmits<{
  (e: 'open', gauge: WatchedGauge, mode: 'gauge' | 'reach'): void
  (e: 'remove-group'): void
}>()

const { removeAndSync } = useWatchlistSync()

const liveCfs = ref<number | null>(null)

const currentCfs = computed(() => liveCfs.value ?? props.leadGauge.currentCfs)

const gaugeShortLabel = computed(() =>
  `${props.leadGauge.source.toUpperCase()}-${props.leadGauge.externalId}`
)

// Full card: show the human-readable gauge name. All other densities: show the short ID label.
const gaugeName = computed(() =>
  props.density === 'full'
    ? (props.leadGauge.name ?? gaugeShortLabel.value)
    : gaugeShortLabel.value
)

const riverDisplayName = computed(() =>
  props.leadGauge.contextReachRiverName ?? props.leadGauge.riverName ?? null
)
</script>
