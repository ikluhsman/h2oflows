<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-5xl mx-auto px-4 py-6 pb-20 sm:pb-6 space-y-8">

      <!-- Empty state -->
      <div v-if="store.gauges.length === 0" class="mt-20 flex flex-col items-center gap-4 text-center">
        <div class="text-5xl">🌊</div>
        <h2 class="text-xl font-semibold">No gauges yet</h2>
        <p class="text-gray-500 max-w-sm text-sm">
          Search for a reach or gauge and add it to your dashboard.
        </p>
        <UButton color="primary" @click="searchOpen = true">Find a gauge</UButton>
      </div>

      <!-- Gauges grouped by basin -->
      <template v-else>
        <div class="flex items-center justify-between mb-4">
          <!-- Density segmented control -->
          <div class="inline-flex items-center rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-100 dark:bg-gray-900 p-0.5">
            <button
              v-for="d in DENSITIES" :key="d.value"
              class="px-2.5 py-1 rounded-md text-xs font-medium transition-all duration-150"
              :class="density === d.value
                ? 'bg-white dark:bg-gray-800 text-gray-900 dark:text-white shadow-sm'
                : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200'"
              @click="density = d.value"
            >{{ d.label }}</button>
          </div>

          <UButton size="xs" color="neutral" variant="outline" icon="i-heroicons-plus" @click="searchOpen = true">
            Add gauge
          </UButton>
        </div>
        <section v-for="group in store.byBasin" :key="group.basin" class="mb-6">
          <button class="flex items-center gap-2 mb-3 w-full text-left" @click="toggleBasin(group.basin)">
            <svg
              class="w-3 h-3 text-gray-400 transition-transform duration-200"
              :class="{ 'rotate-90': !collapsedBasins.has(group.basin) }"
              viewBox="0 0 20 20" fill="currentColor"
            >
              <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
            </svg>
            <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide">{{ group.basin }}</h2>
            <span class="text-xs text-gray-400">({{ group.gauges.length }})</span>
            <div class="flex-1 h-px bg-gray-200 dark:bg-gray-800" />
          </button>

          <template v-if="!collapsedBasins.has(group.basin)">
            <!-- List density: single column rows -->
            <div v-if="density === 'list'" class="flex flex-col gap-1.5">
              <GaugeCard
                v-for="gauge in group.gauges"
                :key="gauge.id"
                :gauge="gauge"
                density="list"
                :shared-with="sharedGaugeMap.get(`${gauge.id}:${gauge.contextReachSlug ?? ''}`)"
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
                :shared-with="sharedGaugeMap.get(`${gauge.id}:${gauge.contextReachSlug ?? ''}`)"
                @open="openGauge(gauge)"
              />
            </div>
          </template>
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
          <div class="flex items-center gap-2 mb-3">
            <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide">Gauge Map</h2>
            <div class="flex-1 h-px bg-gray-200 dark:bg-gray-800" />
            <button
              class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
              @click="mapVisible = !mapVisible"
            >{{ mapVisible ? 'Hide' : 'Show' }}</button>
          </div>
          <ClientOnly v-if="mapVisible">
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
store.deduplicate()
const { refresh } = useWatchlistRefresh()
const { isAuthenticated } = useAuth()
const { addAndSync, removeAndSync, loadFromServer, pushLocalToServer } = useWatchlistSync()

// ── Density toggle ────────────────────────────────────────────────────────────
type Density = 'compact' | 'comfortable' | 'full' | 'list'
const DENSITY_KEY = 'h2oflow_dashboard_density'

const DENSITIES = [
  { value: 'compact'     as Density, label: 'Compact'     },
  { value: 'comfortable' as Density, label: 'Comfortable' },
  { value: 'full'        as Density, label: 'Full'        },
  { value: 'list'        as Density, label: 'List'        },
]

const density = ref<Density>('comfortable')
onMounted(() => {
  const saved = localStorage.getItem(DENSITY_KEY) as Density | null
  if (saved && DENSITIES.some(d => d.value === saved)) density.value = saved
})
watch(density, val => localStorage.setItem(DENSITY_KEY, val))

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

// ── Shared gauge detection ────────────────────────────────────────────────────
// Maps each (gaugeId:contextReachSlug) key → array of other reach names on the
// dashboard that use the same physical gauge station.
const sharedGaugeMap = computed(() => {
  const idCount = new Map<string, number>()
  for (const g of store.gauges) idCount.set(g.id, (idCount.get(g.id) ?? 0) + 1)

  const result = new Map<string, string[]>()
  for (const g of store.gauges) {
    if ((idCount.get(g.id) ?? 0) <= 1) continue
    const key = `${g.id}:${g.contextReachSlug ?? ''}`
    const others = store.gauges
      .filter(x => x.id === g.id && (x.contextReachSlug ?? '') !== (g.contextReachSlug ?? ''))
      .map(x => x.contextReachCommonName ?? x.name ?? x.externalId ?? '')
      .filter(Boolean)
    result.set(key, others)
  }
  return result
})

// ── Collapsible basin sections ────────────────────────────────────────────────
const COLLAPSED_KEY = 'h2oflow_dashboard_collapsed'
const collapsedBasins = ref<Set<string>>(new Set())
onMounted(() => {
  try {
    const saved = localStorage.getItem(COLLAPSED_KEY)
    if (saved) collapsedBasins.value = new Set(JSON.parse(saved))
  } catch {}
})
function toggleBasin(basin: string) {
  const s = new Set(collapsedBasins.value)
  if (s.has(basin)) s.delete(basin); else s.add(basin)
  collapsedBasins.value = s
  localStorage.setItem(COLLAPSED_KEY, JSON.stringify([...s]))
}

// ── UI state ─────────────────────────────────────────────────────────────────
const searchOpen  = ref(false)
const MAP_VIS_KEY = 'h2oflow_dashboard_map_visible'
const mapVisible  = ref(true)
onMounted(() => {
  const saved = localStorage.getItem(MAP_VIS_KEY)
  if (saved !== null) mapVisible.value = saved !== 'false'
})
watch(mapVisible, val => localStorage.setItem(MAP_VIS_KEY, String(val)))

function handleAdd(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
  addAndSync(gauge)
}

const detailOpen  = ref(false)
const detailGauge = ref<WatchedGauge | null>(null)
function openGauge(gauge: WatchedGauge) { detailGauge.value = gauge; detailOpen.value = true }

const aggregateLabel  = ref('')
const aggregateGauges = ref<WatchedGauge[]>([])
function openAggregate(label: string, gauges: WatchedGauge[]) { aggregateLabel.value = label; aggregateGauges.value = gauges }
function closeAggregate() { aggregateGauges.value = [] }
</script>
