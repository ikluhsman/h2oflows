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
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
          <!-- View mode toggle -->
          <div class="flex items-center gap-0.5 bg-gray-100 dark:bg-gray-800 rounded-lg p-1">
            <button
              v-for="m in VIEW_MODES" :key="m.key"
              class="p-1.5 rounded-md transition-colors"
              :class="viewMode === m.key
                ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
                : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'"
              :title="m.label"
              @click="setViewMode(m.key)"
            >
              <!-- List icon -->
              <svg v-if="m.key === 'list'" class="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
                <line x1="2" y1="4" x2="14" y2="4"/><line x1="2" y1="8" x2="14" y2="8"/><line x1="2" y1="12" x2="14" y2="12"/>
              </svg>
              <!-- Comfortable icon: 2x2 grid -->
              <svg v-else-if="m.key === 'comfortable'" class="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="1" y="1" width="6" height="6" rx="1"/><rect x="9" y="1" width="6" height="6" rx="1"/>
                <rect x="1" y="9" width="6" height="6" rx="1"/><rect x="9" y="9" width="6" height="6" rx="1"/>
              </svg>
              <!-- Full icon -->
              <svg v-else class="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="2" y="2" width="12" height="4" rx="1"/><rect x="2" y="7" width="12" height="3" rx="1"/><rect x="2" y="11" width="12" height="3" rx="1"/>
              </svg>
            </button>
          </div>
          <!-- Group by gauge toggle — only shown when shared gauges exist -->
          <button
            v-if="hasSharedGauges"
            class="flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium transition-colors"
            :class="groupByGauge
              ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300 shadow-sm'
              : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200'"
            title="Group by gauge"
            @click="groupByGauge = !groupByGauge"
          >
            <svg v-if="groupByGauge" class="w-4 h-4 shrink-0" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <rect x="1" y="5" width="5" height="6" rx="1"/>
              <rect x="10" y="5" width="5" height="6" rx="1"/>
              <line x1="6" y1="8" x2="10" y2="8"/>
            </svg>
            <svg v-else class="w-4 h-4 shrink-0" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <rect x="1" y="5" width="5" height="6" rx="1"/>
              <rect x="10" y="5" width="5" height="6" rx="1"/>
              <line x1="6.5" y1="6.5" x2="9.5" y2="9.5"/>
            </svg>
            <span class="hidden sm:inline">Group gauge</span>
          </button>
          <!-- Expand / Collapse all -->
          <button
            class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 font-medium transition-colors whitespace-nowrap"
            @click="toggleAllSections"
          >{{ allExpanded ? 'Collapse all' : 'Expand all' }}</button>
          </div>
          <UButton size="xs" color="neutral" variant="outline" icon="i-heroicons-plus" @click="searchOpen = true">
            Add gauge
          </UButton>
        </div>

        <section v-for="stateGroup in byStateTree" :key="stateGroup.name" class="mb-4">
          <!-- State header: large, collapsible, h1+hr style -->
          <button class="flex items-center gap-3 w-full text-left mb-3" @click="toggleState(stateGroup.name)">
            <svg
              class="w-3.5 h-3.5 text-gray-400 dark:text-gray-500 shrink-0 transition-transform duration-200"
              :class="{ 'rotate-90': !collapsedStates.has(stateGroup.name) }"
              viewBox="0 0 20 20" fill="currentColor"
            >
              <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
            </svg>
            <h1 class="text-2xl font-bold text-gray-900 dark:text-white shrink-0">{{ stateGroup.name === '—' ? 'No State' : stateGroup.name }}</h1>
            <div class="flex-1 h-px bg-gray-300 dark:bg-gray-700" />
            <span class="text-sm text-gray-400 shrink-0">({{ stateGroup.reachCount }})</span>
          </button>

          <template v-if="!collapsedStates.has(stateGroup.name)">
            <div v-for="basin in stateGroup.basins" :key="basin.name" class="mb-4 pl-2">
              <!-- Basin header: collapsible -->
              <button class="flex items-center gap-2 w-full text-left mb-3" @click="toggleBasin(stateGroup.name, basin.name)">
                <svg
                  class="w-3 h-3 text-gray-400 dark:text-gray-500 shrink-0 transition-transform duration-150"
                  :class="{ 'rotate-90': !collapsedBasins.has(basinKey(stateGroup.name, basin.name)) }"
                  viewBox="0 0 20 20" fill="currentColor"
                >
                  <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
                </svg>
                <h2 class="text-sm font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">{{ basin.name }} Basin</h2>
                <span class="text-xs text-gray-400">({{ basin.reachCount }})</span>
                <div class="flex-1 h-px bg-gray-200 dark:bg-gray-700/60" />
              </button>

              <template v-if="!collapsedBasins.has(basinKey(stateGroup.name, basin.name))">
              <!-- Reaches grouped by river -->
              <div class="mb-2">
                <template v-for="river in basin.rivers" :key="river.name">
                  <!-- River section divider -->
                  <div class="flex items-center gap-2 mt-4 first:mt-0 mb-2">
                    <svg class="w-4 h-4 text-blue-500/70 dark:text-blue-400/70 shrink-0" viewBox="0 0 32 32" fill="none" aria-hidden="true">
                      <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
                      <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" opacity="0.6"/>
                    </svg>
                    <span class="text-base font-semibold text-blue-600 dark:text-blue-400 shrink-0">{{ river.name }}</span>
                    <div class="flex-1 h-px bg-gray-200 dark:bg-gray-700" />
                  </div>
                  <!-- Cards wrapper -->
                  <template v-if="groupByGauge">
                    <template v-for="split in [splitReachGroups(river.reaches)]" :key="'split'">
                      <div v-if="split.gaugeGroups.length > 0" :class="viewMode === 'list' ? 'space-y-1.5' : 'grid sm:grid-cols-2 gap-2'">
                        <GaugeReachGroup
                          v-for="group in split.gaugeGroups"
                          :key="group.lead.id"
                          :lead-gauge="group.lead"
                          :reach-items="group.all"
                          :density="viewMode"
                          :hide-river-name="true"
                          @open="(g, mode) => openGauge(g, mode)"
                          @remove-group="group.all.forEach(g => removeAndSync(g.id, g.contextReachSlug))"
                        />
                      </div>
                      <DashboardReachGroup
                        v-if="split.ungrouped.length > 0"
                        :reaches="split.ungrouped"
                        :density="viewMode"
                        :class="split.gaugeGroups.length > 0 ? 'mt-3' : ''"
                        @open="(g, mode) => openGauge(g, mode)"
                        @remove="(g) => removeAndSync(g.id, g.contextReachSlug)"
                      />
                    </template>
                  </template>
                  <template v-else>
                    <DashboardReachGroup
                      :reaches="river.reaches"
                      :density="viewMode"
                      @open="(g, mode) => openGauge(g, mode)"
                      @remove="(g) => removeAndSync(g.id, g.contextReachSlug)"
                    />
                  </template>
                </template><!-- end v-for river -->
              </div>

              <!-- Standalone gauges (no reach context) -->
              <div v-if="basin.standaloneGauges.length > 0" class="mb-2 mt-1">
                <div class="flex items-center gap-2 py-1">
                  <svg class="w-3.5 h-3.5 text-gray-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
                    <path d="M12 12 16 8"/>
                    <path d="M3 12a9 9 0 0 1 18 0"/>
                  </svg>
                  <span class="text-xs font-semibold text-gray-500 dark:text-gray-400">Standalone Gauges</span>
                </div>
                <div :class="reachContainerClass">
                  <div
                    v-for="g in basin.standaloneGauges"
                    :key="g.id"
                    class="rounded-2xl border border-gray-200 dark:border-gray-700/60 bg-white dark:bg-gray-900 shadow-sm cursor-pointer active:opacity-80 transition-opacity"
                    :class="viewMode === 'list' ? 'px-3 py-2.5' : 'px-4 py-3'"
                    @click="openGauge(g, 'gauge')"
                  >
                    <div class="flex items-center gap-3">
                      <svg class="w-4 h-4 text-gray-400 dark:text-gray-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/>
                        <path d="M12 12 16 8"/>
                        <path d="M3 12a9 9 0 0 1 18 0"/>
                      </svg>
                      <span class="flex-1 min-w-0 text-sm font-semibold text-gray-800 dark:text-gray-200 truncate">
                        {{ g.name ?? `${g.source.toUpperCase()} ${g.externalId}` }}
                      </span>
                      <div class="w-24 shrink-0 hidden sm:block h-5 opacity-50 pointer-events-none">
                        <GaugeSparkline :gauge-id="g.id" flow-status="unknown" color="#3b82f6" compact />
                      </div>
                      <span :class="viewMode === 'list' ? 'text-sm font-bold tabular-nums text-gray-900 dark:text-white' : 'text-[22px] font-bold tabular-nums text-gray-900 dark:text-white leading-none'">
                        {{ g.currentCfs != null ? g.currentCfs.toLocaleString() : '—' }}
                      </span>
                      <span class="text-xs text-gray-400">cfs</span>
                    </div>
                  </div>
                </div>
              </div>
              </template><!-- end basin v-if -->
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

// ── Reach-primary grouping: state → basin → river → reaches ─────────────────

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
interface StateGroup { name: string; reachCount: number; basins: BasinGroup[] }

const byStateTree = computed<StateGroup[]>(() => {
  // state → basin → river → reaches
  type BasinEntry = { rivers: Map<string, WatchedGauge[]>; standalone: WatchedGauge[] }
  const stateMap = new Map<string, Map<string, BasinEntry>>()

  // De-duplicate: same gauge+reach should only appear once
  const seen = new Set<string>()
  for (const g of store.gauges) {
    const dedupeKey = `${g.id}::${g.contextReachSlug ?? ''}`
    if (seen.has(dedupeKey)) continue
    seen.add(dedupeKey)

    const state = g.stateAbbr ?? '—'
    const basin = cleanBasinName(g.contextReachBasinGroup)
      ?? cleanBasinName(g.watershedName)
      ?? cleanBasinName(g.basinName)
      ?? cleanBasinName(g.contextReachRiverName)
      ?? cleanBasinName(g.riverName)
      ?? 'Other'

    if (!stateMap.has(state)) stateMap.set(state, new Map())
    const basinMap = stateMap.get(state)!
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

  return [...stateMap.entries()]
    .sort(([a], [b]) => a === '—' ? 1 : b === '—' ? -1 : a.localeCompare(b))
    .map(([state, basinMap]) => {
      const basins = [...basinMap.entries()]
        .sort(([a], [b]) => a === 'Other' ? 1 : b === 'Other' ? -1 : a.localeCompare(b))
        .map(([bName, { rivers, standalone }]) => {
          const riverGroups = [...rivers.entries()]
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([rName, reaches]) => ({
              name: rName,
              reaches: [...reaches].sort((a, b) => {
                // river_order (stored, admin-set) preferred; fall back to center longitude
                const ao = a.contextReachRiverOrder
                const bo = b.contextReachRiverOrder
                if (ao != null && bo != null) return ao - bo
                if (ao != null) return -1
                if (bo != null) return 1
                const al = a.contextReachCenterLng ?? a.lng
                const bl = b.contextReachCenterLng ?? b.lng
                if (al == null && bl == null) return 0
                if (al == null) return 1
                if (bl == null) return -1
                return al - bl
              }),
            }))
          const reachCount = riverGroups.reduce((s, r) => s + r.reaches.length, 0) + standalone.length
          return { name: bName, reachCount, rivers: riverGroups, standaloneGauges: standalone }
        })
      const reachCount = basins.reduce((s, b) => s + b.reachCount, 0)
      return { name: state, reachCount, basins }
    })
})

// ── Collapsible sections ────────────────────────────────────────────────────
const COLLAPSED_STATES_KEY = 'h2oflow_dashboard_collapsed_states'
const COLLAPSED_BASINS_KEY = 'h2oflow_dashboard_collapsed_basins'
const collapsedStates = ref<Set<string>>(new Set())
const collapsedBasins = ref<Set<string>>(new Set())

onMounted(() => {
  try {
    const s = localStorage.getItem(COLLAPSED_STATES_KEY)
    if (s) collapsedStates.value = new Set(JSON.parse(s))
    const b = localStorage.getItem(COLLAPSED_BASINS_KEY)
    if (b) collapsedBasins.value = new Set(JSON.parse(b))
  } catch {}
})

function toggleState(state: string) {
  const s = new Set(collapsedStates.value)
  if (s.has(state)) s.delete(state); else s.add(state)
  collapsedStates.value = s
  localStorage.setItem(COLLAPSED_STATES_KEY, JSON.stringify([...s]))
}

function basinKey(stateName: string, basinName: string) { return `${stateName}::${basinName}` }

function toggleBasin(stateName: string, basinName: string) {
  const key = basinKey(stateName, basinName)
  const s = new Set(collapsedBasins.value)
  if (s.has(key)) s.delete(key); else s.add(key)
  collapsedBasins.value = s
  localStorage.setItem(COLLAPSED_BASINS_KEY, JSON.stringify([...s]))
}

// ── Expand / collapse all ────────────────────────────────────────────────────
const allExpanded = computed(() => collapsedStates.value.size === 0 && collapsedBasins.value.size === 0)

function toggleAllSections() {
  if (allExpanded.value) {
    const states = new Set(byStateTree.value.map(s => s.name))
    collapsedStates.value = states
    localStorage.setItem(COLLAPSED_STATES_KEY, JSON.stringify([...states]))
  } else {
    collapsedStates.value = new Set()
    collapsedBasins.value = new Set()
    localStorage.setItem(COLLAPSED_STATES_KEY, '[]')
    localStorage.setItem(COLLAPSED_BASINS_KEY, '[]')
  }
}


// ── View mode ────────────────────────────────────────────────────────────────
const VIEW_MODE_KEY = 'h2oflow_dashboard_view_mode'
const VIEW_MODES = [
  { key: 'list',        label: 'List'        },
  { key: 'comfortable', label: 'Comfortable' },
  { key: 'full',        label: 'Full'        },
] as const
type ViewMode = 'list' | 'comfortable' | 'full'
const viewMode = ref<ViewMode>('comfortable')
onMounted(() => {
  const saved = localStorage.getItem(VIEW_MODE_KEY)
  if (saved === 'list' || saved === 'comfortable' || saved === 'full') {
    viewMode.value = saved
  } else if (saved === 'compact') {
    viewMode.value = 'comfortable' // migrate old compact saves
    localStorage.setItem(VIEW_MODE_KEY, 'comfortable')
  }
})
function setViewMode(m: ViewMode) {
  viewMode.value = m
  localStorage.setItem(VIEW_MODE_KEY, m)
}


// ── Group by gauge ────────────────────────────────────────────────────────────
const GROUP_KEY = 'h2oflow_dashboard_group_by_gauge'
const groupByGauge = ref(false)
onMounted(() => {
  const saved = localStorage.getItem(GROUP_KEY)
  if (saved !== null) groupByGauge.value = saved === 'true'
})
watch(groupByGauge, val => localStorage.setItem(GROUP_KEY, String(val)))

// True when at least one gauge ID appears on multiple reaches (toggle is meaningful).
const hasSharedGauges = computed(() => {
  const counts = new Map<string, number>()
  for (const g of store.gauges) counts.set(g.id, (counts.get(g.id) ?? 0) + 1)
  return [...counts.values()].some(c => c > 1)
})

interface GaugeGroup { lead: WatchedGauge; all: WatchedGauge[] }
interface SplitGroups { gaugeGroups: GaugeGroup[]; ungrouped: WatchedGauge[] }
function splitReachGroups(reaches: WatchedGauge[]): SplitGroups {
  const map = new Map<string, WatchedGauge[]>()
  for (const r of reaches) {
    if (!map.has(r.id)) map.set(r.id, [])
    map.get(r.id)!.push(r)
  }
  const gaugeGroups: GaugeGroup[] = []
  const ungrouped: WatchedGauge[] = []
  for (const all of map.values()) {
    if (all.length > 1) gaugeGroups.push({ lead: all[0]!, all })
    else ungrouped.push(all[0]!)
  }
  return { gaugeGroups, ungrouped }
}

// Container class: 2-col grid for comfortable + full
const reachContainerClass = computed(() =>
  viewMode.value === 'comfortable' || viewMode.value === 'full'
    ? 'grid sm:grid-cols-2 gap-2 mt-1'
    : 'space-y-2 mt-1'
)

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
