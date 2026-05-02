<template>
  <!-- ─── LIST density ────────────────────────────────────────────────────── -->
  <div
    v-if="density === 'list'"
    class="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden"
  >
    <div
      v-for="reach in reaches"
      :key="`${reach.id}::${reach.contextReachSlug}`"
      class="flex items-center gap-2 px-3 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800/30 transition-colors border-b border-gray-100/50 dark:border-gray-800/50 last:border-b-0 cursor-pointer"
      @click="$emit('open', reach, 'reach')"
    >
      <div class="flex items-center gap-1 min-w-0 flex-1">
        <span class="min-w-0 text-sm text-gray-700 dark:text-gray-300 truncate">
          {{ reachLabel(reach) }}
        </span>
        <NuxtLink
          :to="`/reaches/${reach.contextReachSlug}`"
          class="shrink-0 p-0.5 rounded text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors"
          aria-label="View reach page"
          @click.stop
        >
          <svg class="w-3 h-3" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11"/>
          </svg>
        </NuxtLink>
      </div>
      <div class="w-36 shrink-0 hidden sm:block h-6 opacity-60 pointer-events-none">
        <GaugeSparkline :gauge-id="reach.id" flow-status="unknown" :color="sparklineColor(reach)" compact @latest-cfs="(v) => setLiveCfs(reach, v)" />
      </div>
      <span
        v-if="displayFlowStatus(reach) !== 'unknown' || displayFlowBandLabel(reach)"
        :class="['inline-flex items-center rounded-full px-2 py-0.5 text-xs font-bold shrink-0', flowBandBadgeClass(displayFlowBandLabel(reach), displayFlowStatus(reach))]"
      >{{ flowBandLabel(displayFlowBandLabel(reach), displayFlowStatus(reach)) }}</span>
      <span class="w-16 shrink-0 text-right text-sm font-bold tabular-nums" :class="cfsColorClass(reach)">
        {{ displayCfs(reach) != null ? displayCfs(reach)!.toLocaleString() : '—' }}
        <span class="text-xs font-normal text-gray-400">cfs</span>
      </span>
      <button
        class="shrink-0 p-1.5 rounded-lg text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
        aria-label="Remove"
        @click.stop="$emit('remove', reach)"
      >
        <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C9.327 4.025 10 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd"/>
        </svg>
      </button>
    </div>
  </div>

  <!-- ─── CARD densities (compact / comfortable / full) ──────────────────── -->
  <div
    v-else
    class="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden"
  >
    <div
      v-for="reach in reaches"
      :key="`${reach.id}::${reach.contextReachSlug}`"
      class="border-b border-gray-100/50 dark:border-gray-800/50 last:border-b-0 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/30 transition-colors"
      :class="density === 'compact' ? 'px-2.5 py-2' : density === 'comfortable' ? 'px-3 py-2.5' : 'px-4 py-3'"
      @click="$emit('open', reach, 'reach')"
    >
      <!-- Compact: single horizontal row (unchanged) -->
      <template v-if="density === 'compact'">
        <div class="flex items-center gap-2">
          <div class="flex items-center gap-1 min-w-0 flex-1">
            <span class="min-w-0 text-sm font-semibold text-gray-800 dark:text-gray-100 truncate">{{ reachLabel(reach) }}</span>
            <NuxtLink
              :to="`/reaches/${reach.contextReachSlug}`"
              class="shrink-0 p-0.5 rounded text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors"
              aria-label="View reach page"
              @click.stop
            >
              <svg class="w-3 h-3" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11"/>
              </svg>
            </NuxtLink>
          </div>
          <span
            v-if="displayFlowStatus(reach) !== 'unknown' || displayFlowBandLabel(reach)"
            :class="['inline-flex items-center rounded-full px-2 py-0.5 text-xs font-bold shrink-0', flowBandBadgeClass(displayFlowBandLabel(reach), displayFlowStatus(reach))]"
          >{{ flowBandLabel(displayFlowBandLabel(reach), displayFlowStatus(reach)) }}</span>
          <span class="text-lg font-bold tabular-nums shrink-0 leading-none" :class="cfsColorClass(reach)">
            {{ displayCfs(reach) != null ? displayCfs(reach)!.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-400 shrink-0">cfs</span>
          <button
            class="shrink-0 p-1.5 rounded-lg text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
            aria-label="Remove"
            @click.stop="$emit('remove', reach)"
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C9.327 4.025 10 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd"/>
            </svg>
          </button>
        </div>
      </template>

      <!-- Comfortable / Full: left column (name + badge + sparkline) + right CFS -->
      <template v-else>
        <div class="flex items-start gap-3">
          <!-- Left -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-1">
              <span class="min-w-0 font-semibold text-gray-800 dark:text-gray-100 truncate" :class="density === 'comfortable' ? 'text-base' : 'text-base'">
                {{ reachLabel(reach) }}
              </span>
              <NuxtLink
                :to="`/reaches/${reach.contextReachSlug}`"
                class="shrink-0 p-0.5 rounded text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-colors"
                aria-label="View reach page"
                @click.stop
              >
                <svg class="w-3 h-3" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11"/>
                </svg>
              </NuxtLink>
              <span
                v-if="displayFlowStatus(reach) !== 'unknown' || displayFlowBandLabel(reach)"
                :class="['inline-flex items-center rounded-full px-2 py-0.5 text-xs font-bold shrink-0', flowBandBadgeClass(displayFlowBandLabel(reach), displayFlowStatus(reach))]"
              >{{ flowBandLabel(displayFlowBandLabel(reach), displayFlowStatus(reach)) }}</span>
            </div>
            <div class="mt-1.5 opacity-60 pointer-events-none" :class="density === 'full' ? 'h-14' : 'h-6'">
              <GaugeSparkline :gauge-id="reach.id" flow-status="unknown" :color="sparklineColor(reach)" :compact="density !== 'full'" @latest-cfs="(v) => setLiveCfs(reach, v)" />
            </div>
          </div>
          <!-- Right: CFS + remove -->
          <div class="flex items-start gap-1 shrink-0">
            <div class="text-right">
              <div class="font-bold tabular-nums leading-none" :class="[density === 'comfortable' ? 'text-2xl' : 'text-3xl', cfsColorClass(reach)]">
                {{ displayCfs(reach) != null ? displayCfs(reach)!.toLocaleString() : '—' }}
              </div>
              <div class="text-xs text-gray-400 mt-0.5">cfs</div>
            </div>
            <button
              class="mt-0.5 p-1.5 rounded-lg text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
              aria-label="Remove"
              @click.stop="$emit('remove', reach)"
            >
              <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C9.327 4.025 10 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd"/>
              </svg>
            </button>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, onMounted, watch } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { flowBandBadgeClass, flowBandLabel } from '~/utils/flowBand'

const props = defineProps<{
  reaches: WatchedGauge[]
  density?: 'compact' | 'comfortable' | 'full' | 'list'
}>()

const emit = defineEmits<{
  (e: 'open', gauge: WatchedGauge, mode: 'gauge' | 'reach'): void
  (e: 'remove', gauge: WatchedGauge): void
}>()

const liveCfsMap = reactive<Record<string, number>>({})
const { prefetch, bandForCfs, statusForBand } = useReachFlowBand()

function reachKey(reach: WatchedGauge): string {
  return `${reach.id}::${reach.contextReachSlug}`
}

function setLiveCfs(reach: WatchedGauge, v: number) {
  liveCfsMap[reachKey(reach)] = v
}

function displayCfs(reach: WatchedGauge): number | null {
  return liveCfsMap[reachKey(reach)] ?? reach.currentCfs
}

function displayFlowBandLabel(reach: WatchedGauge): string | null {
  return bandForCfs(reach.contextReachSlug, displayCfs(reach)) ?? reach.flowBandLabel ?? null
}

function displayFlowStatus(reach: WatchedGauge): string {
  return statusForBand(displayFlowBandLabel(reach)) ?? reach.flowStatus ?? 'unknown'
}

function prefetchAll() {
  for (const r of props.reaches) {
    if (r.contextReachSlug) prefetch(r.contextReachSlug)
  }
}

onMounted(prefetchAll)
watch(() => props.reaches, prefetchAll)

function reachLabel(reach: WatchedGauge): string {
  return reach.contextReachCommonName ?? reach.contextReachFullName ?? reach.reachName ?? reach.name ?? reach.externalId
}

function sparklineColor(reach: WatchedGauge): string {
  const band = displayFlowBandLabel(reach)
  if (band === 'running')   return '#22c55e'
  if (band === 'high')      return '#16a34a'
  if (band === 'very_high') return '#38bdf8'
  if (band === 'too_low')   return '#ef4444'
  return '#3b82f6'
}

function cfsColorClass(reach: WatchedGauge): string {
  const band = displayFlowBandLabel(reach)
  if (band === 'running')   return 'text-emerald-500 dark:text-emerald-400'
  if (band === 'high')      return 'text-green-600 dark:text-green-500'
  if (band === 'very_high') return 'text-sky-500 dark:text-sky-400'
  if (band === 'too_low')   return 'text-red-500 dark:text-red-400'
  return 'text-gray-900 dark:text-white'
}
</script>
