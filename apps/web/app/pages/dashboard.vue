<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-5xl mx-auto px-4 py-6 pb-20 sm:pb-6 space-y-8">

      <!-- Empty state -->
      <div v-if="store.gauges.length === 0" class="mt-20 flex flex-col items-center gap-4 text-center">
        <div class="text-5xl">🌊</div>
        <h2 class="text-xl font-semibold">No reaches yet</h2>
        <p class="text-gray-500 max-w-sm text-sm">
          Search for a reach or gauge and add it to your dashboard.
        </p>
        <UButton color="primary" @click="searchOpen = true">Find a gauge</UButton>
      </div>

      <!-- Reaches grouped by basin → river -->
      <template v-else>
        <div class="flex items-center justify-end mb-4">
          <UButton size="xs" color="neutral" variant="outline" icon="i-heroicons-plus" @click="searchOpen = true">
            Add gauge
          </UButton>
        </div>

        <section v-for="basin in byBasinTree" :key="basin.name" class="mb-2">
          <!-- Basin header -->
          <button class="flex items-center gap-2 mb-2 w-full text-left" @click="toggleBasin(basin.name)">
            <svg
              class="w-3 h-3 text-gray-400 transition-transform duration-200"
              :class="{ 'rotate-90': !collapsedBasins.has(basin.name) }"
              viewBox="0 0 20 20" fill="currentColor"
            >
              <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
            </svg>
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white uppercase tracking-wide">{{ basin.name }}</h2>
            <span class="text-xs text-gray-400">({{ basin.reachCount }})</span>
            <div class="flex-1 h-px bg-gray-200 dark:bg-gray-800" />
          </button>

          <template v-if="!collapsedBasins.has(basin.name)">
            <!-- River groups within basin -->
            <div v-for="river in basin.rivers" :key="river.name" class="ml-2 mb-1">
              <!-- River header -->
              <button
                class="flex items-center gap-2 py-1 w-full text-left"
                @click="toggleRiver(basin.name, river.name)"
              >
                <svg
                  class="w-2.5 h-2.5 text-gray-400 transition-transform duration-200"
                  :class="{ 'rotate-90': !collapsedRivers.has(`${basin.name}::${river.name}`) }"
                  viewBox="0 0 20 20" fill="currentColor"
                >
                  <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
                </svg>
                <span class="text-xs font-semibold text-gray-700 dark:text-gray-300">{{ river.name }}</span>
                <span class="text-xs text-gray-400">({{ river.reaches.length }})</span>
              </button>

              <!-- Reach rows -->
              <div
                v-if="!collapsedRivers.has(`${basin.name}::${river.name}`)"
                class="ml-2 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden divide-y divide-gray-100 dark:divide-gray-800"
              >
                <DashboardReachRow
                  v-for="reach in river.reaches"
                  :key="`${reach.id}::${reach.contextReachSlug}`"
                  :gauge="reach"
                  @open-gauge="openGauge($event, 'gauge')"
                />
              </div>
            </div>

            <!-- Standalone gauges (no reach context) -->
            <div v-if="basin.standaloneGauges.length > 0" class="ml-2 mb-1">
              <div class="flex items-center gap-2 py-1">
                <svg class="w-3.5 h-3.5 text-gray-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
                  <path d="M12 12 16 8"/>
                  <path d="M3 12a9 9 0 0 1 18 0"/>
                </svg>
                <span class="text-xs font-semibold text-gray-500 dark:text-gray-400">Standalone Gauges</span>
              </div>
              <div class="ml-2 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden divide-y divide-gray-100 dark:divide-gray-800">
                <div
                  v-for="g in basin.standaloneGauges"
                  :key="g.id"
                  class="flex items-center gap-2 sm:gap-3 px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors group cursor-pointer"
                  @click="openGauge(g, 'gauge')"
                >
                  <svg class="w-4 h-4 text-gray-400 dark:text-gray-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
                    <path d="M12 12 16 8"/>
                    <path d="M3 12a9 9 0 0 1 18 0"/>
                  </svg>
                  <span class="flex-1 min-w-0 text-sm font-medium text-gray-600 dark:text-gray-400 truncate">
                    {{ g.name ?? `${g.source.toUpperCase()} ${g.externalId}` }}
                  </span>
                  <div class="w-20 shrink-0 hidden sm:block opacity-60">
                    <GaugeSparkline :gauge-id="g.id" flow-status="unknown" color="#3b82f6" compact />
                  </div>
                  <span class="text-sm font-bold tabular-nums text-gray-900 dark:text-white">
                    {{ g.currentCfs != null ? g.currentCfs.toLocaleString() : '—' }}
                    <span class="text-xs font-normal text-gray-400">cfs</span>
                  </span>
                  <button
                    class="rounded p-1 text-gray-300 dark:text-gray-600 hover:text-red-400 transition-colors shrink-0 opacity-0 group-hover:opacity-100"
                    aria-label="Remove"
                    @click.stop="removeAndSync(g.id, g.contextReachSlug)"
                  >
                    <svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
                      <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/>
                    </svg>
                  </button>
                </div>
              </div>
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
              @open-gauge="(id) => { const g = store.gauges.find(x => x.id === id); if (g) openGauge(g, 'gauge') }"
            />
          </ClientOnly>
        </section>
      </template>
    </main>

    <GaugeSearchModal v-model:open="searchOpen" @add="handleAdd" />
    <GaugeDetailModal v-if="detailGauge" v-model:open="detailOpen" :gauge="detailGauge" :mode="detailMode" />
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

// ── Reach-primary grouping: basin → river → reaches ─────────────────────────

function cleanBasinName(name: string | null): string | null {
  if (!name) return null
  const cleaned = name
    .replace(/^(Upper|Middle|Lower)\s+/i, '')
    .replace(/\s+(River|Rivers|Basin)s?$/i, '')
    .trim()
  return cleaned || null
}

interface RiverGroup { name: string; reaches: WatchedGauge[] }
interface BasinGroup { name: string; reachCount: number; rivers: RiverGroup[]; standaloneGauges: WatchedGauge[] }

const byBasinTree = computed<BasinGroup[]>(() => {
  const basinMap = new Map<string, { rivers: Map<string, WatchedGauge[]>; standalone: WatchedGauge[] }>()

  // De-duplicate: same gauge+reach should only appear once
  const seen = new Set<string>()
  for (const g of store.gauges) {
    const dedupeKey = `${g.id}::${g.contextReachSlug ?? ''}`
    if (seen.has(dedupeKey)) continue
    seen.add(dedupeKey)

    const basin = g.contextReachBasinGroup
      ?? cleanBasinName(g.watershedName)
      ?? cleanBasinName(g.basinName)
      ?? cleanBasinName(g.contextReachRiverName)
      ?? cleanBasinName(g.riverName)
      ?? 'Other'

    if (!basinMap.has(basin)) basinMap.set(basin, { rivers: new Map(), standalone: [] })
    const entry = basinMap.get(basin)!

    if (g.contextReachSlug) {
      const river = g.contextReachRiverName ?? g.riverName ?? 'Unknown River'
      if (!entry.rivers.has(river)) entry.rivers.set(river, [])
      entry.rivers.get(river)!.push(g)
    } else {
      entry.standalone.push(g)
    }
  }

  return [...basinMap.entries()]
    .sort(([a], [b]) => a === 'Other' ? 1 : b === 'Other' ? -1 : a.localeCompare(b))
    .map(([name, { rivers, standalone }]) => {
      const riverGroups = [...rivers.entries()]
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([rName, reaches]) => ({ name: rName, reaches }))
      const reachCount = riverGroups.reduce((s, r) => s + r.reaches.length, 0) + standalone.length
      return { name, reachCount, rivers: riverGroups, standaloneGauges: standalone }
    })
})

// ── Collapsible sections ────────────────────────────────────────────────────
const COLLAPSED_KEY = 'h2oflow_dashboard_collapsed'
const COLLAPSED_RIVERS_KEY = 'h2oflow_dashboard_collapsed_rivers'
const collapsedBasins = ref<Set<string>>(new Set())
const collapsedRivers = ref<Set<string>>(new Set())

onMounted(() => {
  try {
    const saved = localStorage.getItem(COLLAPSED_KEY)
    if (saved) collapsedBasins.value = new Set(JSON.parse(saved))
  } catch {}
  try {
    const saved = localStorage.getItem(COLLAPSED_RIVERS_KEY)
    if (saved) collapsedRivers.value = new Set(JSON.parse(saved))
  } catch {}
})

function toggleBasin(basin: string) {
  const s = new Set(collapsedBasins.value)
  if (s.has(basin)) s.delete(basin); else s.add(basin)
  collapsedBasins.value = s
  localStorage.setItem(COLLAPSED_KEY, JSON.stringify([...s]))
}

function toggleRiver(basin: string, river: string) {
  const key = `${basin}::${river}`
  const s = new Set(collapsedRivers.value)
  if (s.has(key)) s.delete(key); else s.add(key)
  collapsedRivers.value = s
  localStorage.setItem(COLLAPSED_RIVERS_KEY, JSON.stringify([...s]))
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
const detailMode  = ref<'gauge' | 'reach'>('gauge')
function openGauge(gauge: WatchedGauge, mode: 'gauge' | 'reach' = 'gauge') {
  detailGauge.value = gauge
  detailMode.value = mode
  detailOpen.value = true
}

const aggregateLabel  = ref('')
const aggregateGauges = ref<WatchedGauge[]>([])
function openAggregate(label: string, gauges: WatchedGauge[]) { aggregateLabel.value = label; aggregateGauges.value = gauges }
function closeAggregate() { aggregateGauges.value = [] }
</script>
