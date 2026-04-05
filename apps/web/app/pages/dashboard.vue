<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader>
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
      <template #actions>
        <span class="text-xs text-gray-400 dark:text-gray-500 hidden sm:inline">
          <NuxtLink to="/" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">Sign in to save</NuxtLink>
        </span>
        <UButton size="sm" color="primary" variant="soft" icon="i-heroicons-plus" @click="searchOpen = true">
          Add gauge
        </UButton>
      </template>
    </AppHeader>

    <main class="max-w-5xl mx-auto px-4 py-6 space-y-8">

      <!-- Empty state -->
      <div v-if="store.gauges.length === 0" class="mt-20 flex flex-col items-center gap-4 text-center">
        <div class="text-5xl">🌊</div>
        <h2 class="text-xl font-semibold">No gauges yet</h2>
        <p class="text-gray-500 max-w-sm text-sm">
          Search for a river gauge, section, or put-in and add it to your dashboard.
        </p>
        <UButton color="primary" @click="searchOpen = true">Find a gauge</UButton>
      </div>

      <!-- Gauges grouped by river -->
      <template v-else>
        <section v-for="group in store.byRiver" :key="group.river" class="mb-6">
          <div class="flex items-center gap-2 mb-3">
            <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide">{{ group.river }}</h2>
            <div class="flex-1 h-px bg-gray-200 dark:bg-gray-800" />
            <UTooltip :text="group.gauges.length < 2 ? 'Add another gauge from this river to compare' : 'Compare gauges'">
              <UButton
                size="xs" color="neutral" variant="ghost" icon="i-heroicons-chart-bar"
                :disabled="group.gauges.length < 2"
                @click="openAggregate(group.river, group.gauges)"
              >Compare</UButton>
            </UTooltip>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            <GaugeCard
              v-for="gauge in group.gauges"
              :key="gauge.id"
              :gauge="gauge"
              @open="openGauge(gauge)"
            />
          </div>
        </section>

        <section v-if="aggregateGauges.length >= 2" class="border border-gray-200 dark:border-gray-700 rounded-xl p-4">
          <div class="flex items-center justify-between mb-4">
            <h3 class="font-semibold text-sm">{{ aggregateLabel }} · Flow Comparison</h3>
            <UButton size="xs" color="neutral" variant="ghost" icon="i-heroicons-x-mark" @click="closeAggregate" />
          </div>
          <AggregateGraph :gauges="aggregateGauges" />
        </section>

        <!-- Dashboard map -->
        <section>
          <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Gauge Map</h2>
          <ClientOnly>
            <DashboardMap
              :gauges="store.gauges"
              @remove-gauge="store.removeGauge($event)"
            />
          </ClientOnly>
        </section>
      </template>
    </main>

    <GaugeSearchModal v-model:open="searchOpen" @add="handleAdd" />
    <GaugeDetailModal v-if="detailGauge" v-model:open="detailOpen" :gauge="detailGauge" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'

definePageMeta({ ssr: false })

const store = useWatchlistStore()
const { refresh } = useWatchlistRefresh()

let refreshTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  refresh()
  refreshTimer = setInterval(refresh, 60_000)
})
onUnmounted(() => { if (refreshTimer) clearInterval(refreshTimer) })

const searchOpen = ref(false)

function handleAdd(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
  store.addGauge(gauge)
}

const activeTripLabel = computed(() => {
  const trip = store.activeTrip
  if (!trip) return ''
  const ms = Date.now() - new Date(trip.startedAt).getTime()
  const h = Math.floor(ms / 3_600_000)
  const m = Math.floor((ms % 3_600_000) / 60_000)
  return trip.reachName ? `${trip.reachName} · ${h > 0 ? `${h}h ` : ''}${m}m` : `${h > 0 ? `${h}h ` : ''}${m}m`
})

const detailOpen  = ref(false)
const detailGauge = ref<WatchedGauge | null>(null)
function openGauge(gauge: WatchedGauge) { detailGauge.value = gauge; detailOpen.value = true }

const aggregateLabel  = ref('')
const aggregateGauges = ref<WatchedGauge[]>([])
function openAggregate(label: string, gauges: WatchedGauge[]) { aggregateLabel.value = label; aggregateGauges.value = gauges }
function closeAggregate() { aggregateGauges.value = [] }
</script>
