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
        <!-- Density toggle with current-view label -->
        <div class="hidden sm:flex items-center gap-1.5 shrink-0">
          <span class="text-xs text-gray-400 dark:text-gray-500">{{ currentDensityLabel }}</span>
          <div class="flex items-center rounded-md border border-gray-200 dark:border-gray-700 overflow-hidden">
            <UTooltip v-for="d in DENSITIES" :key="d.value" :text="d.label">
              <button
                class="px-2 py-1.5 transition-colors"
                :class="density === d.value
                  ? 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200'
                  : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'"
                @click="density = d.value"
              >
                <component :is="d.icon" class="w-3.5 h-3.5" />
              </button>
            </UTooltip>
          </div>
        </div>
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
        <div class="flex items-center justify-between -mb-2">
          <span />
          <UButton size="xs" color="neutral" variant="outline" icon="i-heroicons-plus" @click="searchOpen = true">
            Add gauge
          </UButton>
        </div>
        <section v-for="group in store.byRiver" :key="group.river" class="mb-6">
          <div class="flex items-center gap-2 mb-3">
            <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide">{{ group.river }}</h2>
            <div class="flex-1 h-px bg-gray-200 dark:bg-gray-800" />
          </div>

          <!-- List density: single column rows -->
          <div v-if="density === 'list'" class="flex flex-col gap-1.5">
            <GaugeCard
              v-for="gauge in group.gauges"
              :key="gauge.id"
              :gauge="gauge"
              density="list"
              @open="openGauge(gauge)"
            />
          </div>

          <!-- Card densities: grid layout -->
          <div v-else :class="gridClass">
            <GaugeCard
              v-for="gauge in group.gauges"
              :key="gauge.id"
              :gauge="gauge"
              :density="density"
              @open="openGauge(gauge)"
            />
          </div>
        </section>

        <section v-if="aggregateGauges.length >= 2" class="border border-gray-300 dark:border-gray-700 rounded-xl p-4">
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
              @remove-gauge="removeAndSync($event)"
              @open-gauge="(id) => { const g = store.gauges.find(x => x.id === id); if (g) openGauge(g) }"
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
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'

definePageMeta({ ssr: false })

const store = useWatchlistStore()
const { refresh } = useWatchlistRefresh()
const { isAuthenticated } = useAuth()
const { addAndSync, removeAndSync, loadFromServer, pushLocalToServer } = useWatchlistSync()

// ── Density toggle ────────────────────────────────────────────────────────────
type Density = 'compact' | 'comfortable' | 'full' | 'list'
const DENSITY_KEY = 'h2oflow_dashboard_density'

// Inline SVG icon components for the toggle buttons
const IconGrid4 = { template: `<svg viewBox="0 0 16 16" fill="currentColor"><rect x="1" y="1" width="6" height="6" rx="1"/><rect x="9" y="1" width="6" height="6" rx="1"/><rect x="1" y="9" width="6" height="6" rx="1"/><rect x="9" y="9" width="6" height="6" rx="1"/></svg>` }
const IconGrid3 = { template: `<svg viewBox="0 0 16 16" fill="currentColor"><rect x="1" y="1" width="4" height="14" rx="1"/><rect x="6" y="1" width="4" height="14" rx="1"/><rect x="11" y="1" width="4" height="14" rx="1"/></svg>` }
const IconGrid2 = { template: `<svg viewBox="0 0 16 16" fill="currentColor"><rect x="1" y="1" width="6" height="14" rx="1"/><rect x="9" y="1" width="6" height="14" rx="1"/></svg>` }
const IconList  = { template: `<svg viewBox="0 0 16 16" fill="currentColor"><rect x="1" y="2" width="14" height="2.5" rx="1"/><rect x="1" y="6.75" width="14" height="2.5" rx="1"/><rect x="1" y="11.5" width="14" height="2.5" rx="1"/></svg>` }

const DENSITIES = [
  { value: 'compact'     as Density, label: 'Compact',     icon: IconGrid4 },
  { value: 'comfortable' as Density, label: 'Comfortable', icon: IconGrid3 },
  { value: 'full'        as Density, label: 'Full',        icon: IconGrid2 },
  { value: 'list'        as Density, label: 'List',        icon: IconList  },
]

const density = ref<Density>('comfortable')
onMounted(() => {
  const saved = localStorage.getItem(DENSITY_KEY) as Density | null
  if (saved && DENSITIES.some(d => d.value === saved)) density.value = saved
})
watch(density, val => localStorage.setItem(DENSITY_KEY, val))

const currentDensityLabel = computed(() =>
  DENSITIES.find(d => d.value === density.value)?.label ?? ''
)

const gridClass = computed(() => ({
  'compact':     'grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2',
  'comfortable': 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3',
  'full':        'grid grid-cols-1 sm:grid-cols-2 gap-4',
}[density.value] ?? 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3'))

// ── Server sync ───────────────────────────────────────────────────────────────
let serverSynced = false
async function syncWithServer() {
  if (serverSynced) return
  serverSynced = true
  await loadFromServer()
  await pushLocalToServer()
}

watch(isAuthenticated, (val) => { if (val) syncWithServer() })

let refreshTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  if (isAuthenticated.value) syncWithServer()
  refresh()
  refreshTimer = setInterval(refresh, 60_000)
})
onUnmounted(() => { if (refreshTimer) clearInterval(refreshTimer) })

// ── UI state ─────────────────────────────────────────────────────────────────
const searchOpen = ref(false)

function handleAdd(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
  addAndSync(gauge)
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
