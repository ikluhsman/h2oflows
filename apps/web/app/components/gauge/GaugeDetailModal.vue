<template>
  <UModal v-model:open="open" :ui="{ width: 'max-w-xl' }">
    <template #header>
      <div class="flex items-start justify-between gap-3 w-full">
        <div class="min-w-0 flex-1">
          <!-- Reach mode: reach name links to reach page; gauge info as subtitle -->
          <template v-if="mode === 'reach' && reachTitle">
            <NuxtLink
              v-if="reachSlugForLink"
              :to="`/reaches/${reachSlugForLink}`"
              class="text-lg font-bold text-gray-900 dark:text-white truncate leading-tight hover:text-blue-600 dark:hover:text-blue-400 transition-colors block"
              @click="open = false"
            >{{ reachTitle }}</NuxtLink>
            <h2 v-else class="text-lg font-bold text-gray-900 dark:text-white truncate leading-tight">{{ reachTitle }}</h2>
            <p class="text-xs text-gray-400 truncate mt-0.5">
              {{ gaugeName }} ·
              <a :href="sourceUrl" target="_blank" rel="noopener" class="hover:text-blue-400 underline underline-offset-2">
                {{ gauge.source.toUpperCase() }} {{ gauge.externalId }}
              </a>
            </p>
          </template>
          <!-- Gauge mode (default): gauge name as title, no reach context -->
          <template v-else>
            <h2 class="text-lg font-bold text-gray-900 dark:text-white truncate leading-tight">{{ gaugeName }}</h2>
            <p class="text-xs text-gray-400 truncate mt-0.5">
              <a :href="sourceUrl" target="_blank" rel="noopener" class="hover:text-blue-400 underline underline-offset-2">
                {{ gauge.source.toUpperCase() }} · {{ gauge.externalId }}
              </a>
              <span v-if="gauge.watershedName"> · {{ gauge.watershedName }}</span>
            </p>
          </template>
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
        <!-- CFS + trend arrow — left-aligned, prominent -->
        <div class="flex items-baseline gap-2 flex-wrap">
          <span class="text-3xl font-bold tabular-nums leading-none text-gray-900 dark:text-white">
            {{ displayCfs != null ? displayCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-sm text-gray-500">cfs</span>
          <TrendArrow v-if="displayCfs != null" :gauge-id="gauge.id" class="text-lg" />
          <!-- Flow band badge — reach mode only -->
          <span
            v-if="mode === 'reach' && (gauge.flowStatus !== 'unknown' || gauge.flowBandLabel)"
            :class="['inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', flowBandBadgeClass(gauge.flowBandLabel, gauge.flowStatus)]"
          >{{ flowBandLabel(gauge.flowBandLabel, gauge.flowStatus) }}</span>
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

        <!-- Graph:
             reach mode — colored line + flow band fills + legend (below chart)
             gauge mode — neutral blue, no ranges, no legend -->
        <GaugeGraph
          :gauge-id="gauge.id"
          :current-cfs="displayCfs"
          :reach-slug="graphReachSlug"
          :no-ranges="mode !== 'reach'"
          :color="mode !== 'reach' ? '#3b82f6' : undefined"
          @latest-cfs="liveCfs = $event"
        />

        <!-- Last updated -->
        <p v-if="gauge.lastReadingAt" class="text-xs text-gray-500">
          Last reading {{ lastReadingRelative }}
        </p>

        <!-- View this reach — reach mode only -->
        <div v-if="mode === 'reach' && reachSlugForLink" class="pt-1 border-t border-gray-100 dark:border-gray-800">
          <NuxtLink
            :to="`/reaches/${reachSlugForLink}`"
            class="inline-flex items-center gap-1 text-sm text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300 font-medium transition-colors"
            @click="open = false"
          >
            View {{ reachTitle ?? 'reach' }} details
            <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M5 10h10M11 6l4 4-4 4"/>
            </svg>
          </NuxtLink>
        </div>

      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { useWatchlistStore } from '~/stores/watchlist'
import { flowBandBadgeClass, flowBandLabel } from '~/utils/flowBand'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{
  gauge: WatchedGauge
  mode?: 'gauge' | 'reach'  // 'gauge' = neutral blue, no bands; 'reach' = colored + reach name + bands
}>()

const liveCfs = ref<number | null>(null)
watch(open, (v) => { if (!v) liveCfs.value = null })

const displayCfs = computed(() => liveCfs.value ?? props.gauge.currentCfs)

const gaugeName = computed(() =>
  props.gauge.name ?? `${props.gauge.source.toUpperCase()} ${props.gauge.externalId}`
)

// Reach title (reach mode header) — prefer common name, fall back to put-in→take-out
const reachTitle = computed(() =>
  props.gauge.contextReachCommonName
    ?? props.gauge.contextReachFullName
    ?? null
)

// Slug used for "View this reach" link and the header NuxtLink
const reachSlugForLink = computed(() =>
  props.gauge.contextReachSlug
    ?? props.gauge.reachSlug
    ?? props.gauge.reachSlugs?.[0]
    ?? null
)

// Reach slug passed to GaugeGraph only in reach mode — drives flow band coloring + legend.
// null in gauge mode forces neutral blue graph with no bands.
const graphReachSlug = computed(() =>
  props.mode === 'reach'
    ? (props.gauge.contextReachSlug ?? props.gauge.reachSlug ?? null)
    : null
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
