<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <!-- Top bar -->
    <header class="sticky top-0 z-20 border-b border-gray-200 dark:border-gray-800 bg-white/90 dark:bg-gray-950/90 backdrop-blur-sm">
      <div class="max-w-5xl mx-auto px-4 py-3 flex items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <span class="text-xl font-bold tracking-tight">H2OFlow</span>
          <span class="text-xs text-gray-400 hidden sm:block">streamflow dashboard</span>
        </div>

        <!-- Active trip banner -->
        <div
          v-if="store.hasActiveTrip"
          class="flex items-center gap-1.5 text-xs font-medium text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950 rounded-full px-3 py-1"
        >
          <span class="relative flex h-2 w-2">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
            <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
          </span>
          Trip recording · {{ activeTripLabel }}
        </div>

        <UColorModeButton size="sm" color="neutral" variant="ghost" />

        <UButton
          size="sm"
          color="primary"
          variant="soft"
          icon="i-heroicons-plus"
          @click="searchOpen = true"
        >
          Add gauge
        </UButton>
      </div>
    </header>

    <main class="max-w-5xl mx-auto px-4 py-6 space-y-8">

      <!-- Empty state -->
      <div
        v-if="store.gauges.length === 0"
        class="mt-20 flex flex-col items-center gap-4 text-center"
      >
        <div class="text-5xl">🌊</div>
        <h2 class="text-xl font-semibold">No gauges yet</h2>
        <p class="text-gray-500 max-w-sm text-sm">
          Search for a river gauge, section, or put-in and add it to your dashboard.
        </p>
        <UButton color="primary" @click="searchOpen = true">
          Find a gauge
        </UButton>
      </div>

      <!-- Gauges grouped by reach -->
      <template v-else>
        <section
          v-for="group in store.byReach"
          :key="group.reach ?? '__ungrouped__'"
          class="mb-6"
        >
          <div class="flex items-center gap-2 mb-3">
            <NuxtLink
              v-if="group.gauges[0]?.reachSlug"
              :to="`/reaches/${group.gauges[0].reachSlug}`"
              class="text-sm font-semibold text-gray-500 uppercase tracking-wide hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
            >
              {{ group.reach }}
            </NuxtLink>
            <h2 v-else class="text-sm font-semibold text-gray-500 uppercase tracking-wide">
              {{ group.reach ?? 'Other Gauges' }}
            </h2>
            <div class="flex-1 h-px bg-gray-200 dark:bg-gray-800" />
            <!-- Aggregate graph button — enabled when ≥2 gauges in this section -->
            <UTooltip
              :text="group.gauges.length < 2 ? 'Add another gauge from this reach to compare' : 'Compare gauges'"
            >
              <UButton
                size="xs"
                color="neutral"
                variant="ghost"
                icon="i-heroicons-chart-bar"
                :disabled="group.gauges.length < 2"
                @click="openAggregate(group.reach ?? 'Other', group.gauges)"
              >
                Compare
              </UButton>
            </UTooltip>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            <GaugeCard
              v-for="gauge in group.gauges"
              :key="gauge.id"
              :gauge="gauge"
              hide-reach-subtitle
              @open="openGauge(gauge)"
            />
          </div>
        </section>

        <!-- Aggregate graph panel -->
        <section v-if="aggregateGauges.length >= 2" class="border border-gray-200 dark:border-gray-700 rounded-xl p-4">
          <div class="flex items-center justify-between mb-4">
            <h3 class="font-semibold text-sm">{{ aggregateLabel }} · Flow Comparison</h3>
            <UButton size="xs" color="neutral" variant="ghost" icon="i-heroicons-x-mark" @click="closeAggregate" />
          </div>
          <AggregateGraph :gauges="aggregateGauges" />
        </section>
      </template>
    </main>

    <!-- Gauge search modal -->
    <GaugeSearchModal v-model:open="searchOpen" @add="handleAdd" />

    <!-- Gauge detail modal -->
    <GaugeDetailModal
      v-if="detailGauge"
      v-model:open="detailOpen"
      :gauge="detailGauge"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'

definePageMeta({ ssr: false })

const store = useWatchlistStore()
const { refresh } = useWatchlistRefresh()

// Refresh metadata + current_cfs on mount, then every 60 seconds.
let refreshTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  refresh()
  refreshTimer = setInterval(refresh, 60_000)
})
onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

// Search modal
const searchOpen = ref(false)

function handleAdd(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
  store.addGauge(gauge)
}

// Active trip label (e.g. "Browns Canyon · 42m")
const activeTripLabel = computed(() => {
  const trip = store.activeTrip
  if (!trip) return ''
  const ms = Date.now() - new Date(trip.startedAt).getTime()
  const h = Math.floor(ms / 3_600_000)
  const m = Math.floor((ms % 3_600_000) / 60_000)
  const dur = h > 0 ? `${h}h ${m}m` : `${m}m`
  return trip.reachName ? `${trip.reachName} · ${dur}` : dur
})

// Gauge detail modal
const detailOpen  = ref(false)
const detailGauge = ref<WatchedGauge | null>(null)

function openGauge(gauge: WatchedGauge) {
  detailGauge.value = gauge
  detailOpen.value  = true
}

// Aggregate graph
const aggregateLabel = ref<string>('')
const aggregateGauges = ref<WatchedGauge[]>([])

function openAggregate(label: string, gauges: WatchedGauge[]) {
  aggregateLabel.value = label
  aggregateGauges.value = gauges
}
function closeAggregate() {
  aggregateGauges.value = []
}
</script>
