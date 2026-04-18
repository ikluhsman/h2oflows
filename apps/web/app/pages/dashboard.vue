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
              :class="[
                viewMode === m.key
                  ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
                  : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300',
                m.key === 'comfortable' ? 'hidden sm:block' : '',
              ]"
              :title="m.label"
              @click="setViewMode(m.key)"
            >
              <!-- List icon -->
              <svg v-if="m.key === 'list'" class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
                <line x1="2" y1="4" x2="14" y2="4"/><line x1="2" y1="8" x2="14" y2="8"/><line x1="2" y1="12" x2="14" y2="12"/>
              </svg>
              <!-- Compact icon: 2 full-width stacked cards -->
              <svg v-else-if="m.key === 'compact'" class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="2" y="2" width="12" height="5" rx="1"/><rect x="2" y="9" width="12" height="5" rx="1"/>
              </svg>
              <!-- Comfortable icon: 2x2 grid -->
              <svg v-else-if="m.key === 'comfortable'" class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="1" y="1" width="6" height="6" rx="1"/><rect x="9" y="1" width="6" height="6" rx="1"/>
                <rect x="1" y="9" width="6" height="6" rx="1"/><rect x="9" y="9" width="6" height="6" rx="1"/>
              </svg>
              <!-- Full icon -->
              <svg v-else class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="2" y="2" width="12" height="4" rx="1"/><rect x="2" y="7" width="12" height="3" rx="1"/><rect x="2" y="11" width="12" height="3" rx="1"/>
              </svg>
            </button>
          </div>
          <!-- Group by gauge toggle — only shown when shared gauges exist -->
          <button
            v-if="hasSharedGauges"
            class="p-1.5 rounded-md transition-colors"
            :class="groupByGauge
              ? 'bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm'
              : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'"
            title="Group by gauge"
            @click="groupByGauge = !groupByGauge"
          >
            <!-- Link/merge icon: two cards linked into one -->
            <svg class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <rect x="1" y="5" width="5" height="6" rx="1"/>
              <rect x="10" y="5" width="5" height="6" rx="1"/>
              <line x1="6" y1="8" x2="10" y2="8"/>
            </svg>
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
            <!-- River groups within basin.
                 Only show river sub-header when there are multiple rivers (or standalones). -->
            <div v-for="river in basin.rivers" :key="river.name" class="mb-2">
              <!-- River header — hidden when this basin has only one river and no standalones -->
              <button
                v-if="basin.rivers.length > 1 || basin.standaloneGauges.length > 0"
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

              <!-- Reach cards — always visible when only one river (no header to collapse) -->
              <div
                v-if="basin.rivers.length === 1 && basin.standaloneGauges.length === 0 || !collapsedRivers.has(`${basin.name}::${river.name}`)"
                :class="reachContainerClass"
              >
                <!-- Grouped mode: merge reaches sharing the same gauge into one card -->
                <template v-if="groupByGauge">
                  <template v-for="group in groupReaches(river.reaches)" :key="group.lead.id">
                    <GaugeReachGroup
                      v-if="group.all.length > 1"
                      :lead-gauge="group.lead"
                      :reach-items="group.all"
                      :density="viewMode"
                      @open="(g, mode) => openGauge(g, mode)"
                    />
                    <DashboardReachRow
                      v-else
                      :gauge="group.lead"
                      :view="rowView"
                      @open-gauge="openGauge($event, 'reach')"
                      @remove-gauge="removeAndSync($event.id, $event.contextReachSlug)"
                    />
                  </template>
                </template>
                <!-- Normal mode: one card per reach -->
                <template v-else>
                  <DashboardReachRow
                    v-for="reach in river.reaches"
                    :key="`${reach.id}::${reach.contextReachSlug}`"
                    :gauge="reach"
                    :view="rowView"
                    @open-gauge="openGauge($event, 'reach')"
                    @remove-gauge="removeAndSync($event.id, $event.contextReachSlug)"
                  />
                </template>
              </div>
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

    const basin = cleanBasinName(g.contextReachBasinGroup)
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
        .map(([rName, reaches]) => ({
          name: rName,
          // Sort upstream→downstream: gauge lng ascending (west = upstream for CO rivers).
          // Nulls go last.
          reaches: [...reaches].sort((a, b) => {
            const al = a.lng, bl = b.lng
            if (al == null && bl == null) return 0
            if (al == null) return 1
            if (bl == null) return -1
            return al - bl
          }),
        }))
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

// ── Expand / collapse all ────────────────────────────────────────────────────
const allExpanded = computed(() => collapsedBasins.value.size === 0 && collapsedRivers.value.size === 0)

function toggleAllSections() {
  if (allExpanded.value) {
    // Collapse everything
    const basins = new Set(byBasinTree.value.map(b => b.name))
    const rivers = new Set(
      byBasinTree.value.flatMap(b => b.rivers.map(r => `${b.name}::${r.name}`))
    )
    collapsedBasins.value = basins
    collapsedRivers.value = rivers
    localStorage.setItem(COLLAPSED_KEY, JSON.stringify([...basins]))
    localStorage.setItem(COLLAPSED_RIVERS_KEY, JSON.stringify([...rivers]))
  } else {
    // Expand everything
    collapsedBasins.value = new Set()
    collapsedRivers.value = new Set()
    localStorage.setItem(COLLAPSED_KEY, '[]')
    localStorage.setItem(COLLAPSED_RIVERS_KEY, '[]')
  }
}

// ── View mode ────────────────────────────────────────────────────────────────
const VIEW_MODE_KEY = 'h2oflow_dashboard_view_mode'
const VIEW_MODES = [
  { key: 'list',        label: 'List'        },
  { key: 'compact',     label: 'Compact'     },
  { key: 'comfortable', label: 'Comfortable' },
  { key: 'full',        label: 'Full'        },
] as const
type ViewMode = 'list' | 'compact' | 'comfortable' | 'full'
const viewMode = ref<ViewMode>('compact')
onMounted(() => {
  const saved = localStorage.getItem(VIEW_MODE_KEY)
  if (saved === 'list' || saved === 'compact' || saved === 'comfortable' || saved === 'full') {
    viewMode.value = saved
  }
})
function setViewMode(m: ViewMode) {
  viewMode.value = m
  localStorage.setItem(VIEW_MODE_KEY, m)
}

// Maps viewMode → DashboardReachRow 'view' prop
const rowView = computed<'list' | 'compact' | 'full'>(() => {
  if (viewMode.value === 'full') return 'full'
  if (viewMode.value === 'list') return 'list'
  return 'compact' // 'compact' and 'comfortable' both use compact cards
})

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
function groupReaches(reaches: WatchedGauge[]): GaugeGroup[] {
  const map = new Map<string, WatchedGauge[]>()
  for (const r of reaches) {
    if (!map.has(r.id)) map.set(r.id, [])
    map.get(r.id)!.push(r)
  }
  return [...map.values()].map(all => ({ lead: all[0]!, all }))
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
