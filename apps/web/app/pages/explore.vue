<template>
  <div class="h-screen flex flex-col overflow-hidden bg-white dark:bg-gray-950">

    <!-- Demo banner -->
    <div v-if="showDemoBanner" class="shrink-0 bg-amber-50 dark:bg-amber-950 border-b border-amber-200 dark:border-amber-800 px-4 py-2 flex items-center justify-between gap-4 text-sm">
      <p class="text-amber-800 dark:text-amber-200 text-center flex-1">
        <span class="font-semibold">Demo only.</span>
        River data is AI-seeded and unverified — do not use for trip planning or safety decisions.
      </p>
      <button class="shrink-0 text-amber-600 dark:text-amber-400 hover:text-amber-900 dark:hover:text-amber-100 font-medium transition-colors" @click="dismissBanner">Dismiss</button>
    </div>

    <AppHeader class="shrink-0" />

    <!-- Split-pane body -->
    <div class="flex-1 overflow-hidden flex relative">

      <!-- ── Left panel: basin → river → reach tree ────────────────────────── -->
      <aside
        class="shrink-0 border-r border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 flex flex-col overflow-hidden transition-all"
        :class="listVisible
          ? 'absolute sm:relative inset-0 sm:inset-auto z-30 sm:z-auto w-full sm:w-80'
          : 'hidden sm:flex sm:w-80'"
      >
        <!-- Search + mobile map toggle -->
        <div class="px-3 py-2.5 sm:border-b border-gray-100 dark:border-gray-800 shrink-0 flex items-center gap-2">
          <input
            v-model="query"
            type="search"
            placeholder="Search reaches, rivers, basins…"
            class="flex-1 text-sm bg-gray-100 dark:bg-gray-900 rounded-md px-3 py-1.5 text-gray-800 dark:text-gray-200 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
          <!-- New reach button (admin only) -->
          <button
            v-if="isDataAdmin"
            class="shrink-0 flex items-center gap-1 p-1.5 rounded-md text-gray-400 hover:text-blue-500 dark:hover:text-blue-400 hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors"
            title="New reach"
            @click="authorModalOpen = true"
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <circle cx="10" cy="10" r="8"/><line x1="10" y1="6" x2="10" y2="14"/><line x1="6" y1="10" x2="14" y2="10"/>
            </svg>
          </button>
          <!-- Back to map (mobile only, shown when list is visible) -->
          <button
            class="sm:hidden shrink-0 flex items-center gap-1 px-2 py-1.5 rounded-md text-xs font-medium text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-950/50 transition-colors"
            aria-label="Show map"
            @click="listVisible = false"
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 5l-5 5 5 5"/>
            </svg>
            Map
          </button>
        </div>

        <!-- Loading / error / empty states -->
        <div v-if="loading" class="flex-1 flex items-center justify-center text-sm text-gray-400">Loading…</div>
        <div v-else-if="loadError" class="flex-1 flex items-center justify-center text-sm text-red-400">{{ loadError }}</div>
        <div v-else-if="query.length >= 2 && filteredStates.length === 0" class="flex-1 flex items-center justify-center text-sm text-gray-400">
          No results for "{{ query }}"
        </div>

        <!-- Tree -->
        <div v-else class="flex-1 overflow-y-auto">
          <div v-for="stateGroup in filteredStates" :key="stateGroup.name">
            <!-- State header -->
            <button
              class="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-gray-100 dark:hover:bg-gray-800/60 transition-colors bg-gray-50 dark:bg-gray-900/50 border-b border-gray-200 dark:border-gray-700"
              @click="toggleState(stateGroup.name)"
            >
              <svg
                class="w-3 h-3 text-gray-500 shrink-0 transition-transform duration-150"
                :class="{ 'rotate-90': !collapsed.states.has(stateGroup.name) }"
                viewBox="0 0 20 20" fill="currentColor"
              >
                <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
              </svg>
              <span class="text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-widest flex-1 text-left">{{ stateGroup.name === '—' ? 'No State' : stateGroup.name }}</span>
              <span class="text-xs text-gray-400">{{ stateGroup.totalCount }}</span>
            </button>

            <template v-if="!collapsed.states.has(stateGroup.name)">
              <div v-for="basin in stateGroup.basins" :key="basin.name">
                <!-- Basin header -->
                <button
                  class="w-full flex items-center gap-2 px-3 py-1.5 text-left hover:bg-gray-50 dark:hover:bg-gray-900/50 transition-colors border-b border-gray-100 dark:border-gray-800/50"
                  @click="toggleBasin(`${stateGroup.name}::${basin.name}`)"
                >
                  <svg
                    class="w-3 h-3 text-gray-400 shrink-0 transition-transform duration-150"
                    :class="{ 'rotate-90': !collapsed.basins.has(`${stateGroup.name}::${basin.name}`) }"
                    viewBox="0 0 20 20" fill="currentColor"
                  >
                    <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
                  </svg>
                  <span class="text-xs font-semibold text-gray-600 dark:text-gray-300 flex-1 text-left">{{ basin.name }} Basin</span>
                  <span class="text-xs text-gray-400">{{ basin.reachCount }}</span>
                </button>

                <template v-if="!collapsed.basins.has(`${stateGroup.name}::${basin.name}`)">
                  <div v-for="river in basin.rivers" :key="river.name">
                    <!-- River header -->
                    <div class="flex items-center group/river">
                      <button
                        class="flex items-center gap-2 pl-6 pr-2 py-1.5 text-left hover:bg-gray-50 dark:hover:bg-gray-900/50 transition-colors flex-1 min-w-0"
                        @click="toggleRiver(stateGroup.name, basin.name, river.name)"
                      >
                        <svg
                          class="w-2.5 h-2.5 text-gray-400 shrink-0 transition-transform duration-150"
                          :class="{ 'rotate-90': !collapsed.rivers.has(`${stateGroup.name}::${basin.name}::${river.name}`) }"
                          viewBox="0 0 20 20" fill="currentColor"
                        >
                          <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
                        </svg>
                        <span class="text-xs font-semibold text-gray-600 dark:text-gray-400 flex-1 text-left truncate">{{ river.name }}</span>
                        <span class="text-xs text-gray-400 shrink-0">{{ river.reaches.length }}</span>
                      </button>
                      <!-- Bulk add all reaches in this river -->
                      <button
                        class="shrink-0 px-2 py-1.5 opacity-60 sm:opacity-0 sm:group-hover/river:opacity-100 hover:opacity-100 transition-opacity text-gray-400 hover:text-blue-500 dark:hover:text-blue-400"
                        :aria-label="`Add all ${river.name} reaches to dashboard`"
                        @click.stop="addAllRiverReaches(river.reaches)"
                      >
                        <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                          <circle cx="10" cy="10" r="8"/><line x1="10" y1="6" x2="10" y2="14"/><line x1="6" y1="10" x2="14" y2="10"/>
                        </svg>
                      </button>
                    </div>

                    <!-- Reach rows -->
                    <template v-if="!collapsed.rivers.has(`${stateGroup.name}::${basin.name}::${river.name}`)">
                      <div
                        v-for="reach in river.reaches"
                        :key="reach.slug"
                        :ref="(el) => setReachRef(reach.slug, el as HTMLElement | null)"
                        class="flex items-center gap-2 pl-10 pr-2 py-1.5 cursor-pointer transition-colors group"
                        :class="hoveredSlug === reach.slug
                          ? 'bg-blue-50 dark:bg-blue-950/40'
                          : 'hover:bg-gray-50 dark:hover:bg-gray-900/60'"
                        @mouseenter="hoveredSlug = reach.slug"
                        @mouseleave="hoveredSlug = null"
                        @click="onReachRowClick(reach)"
                      >
                        <!-- Flow dot -->
                        <span
                          class="w-2 h-2 rounded-full shrink-0"
                          :style="{ background: flowStatusColor(reach.flow_status) }"
                        />
                        <!-- Name -->
                        <span class="flex-1 min-w-0 text-sm text-gray-800 dark:text-gray-200 truncate">
                          {{ reach.common_name ?? reach.put_in_name ?? reach.slug }}
                        </span>
                        <!-- CFS -->
                        <span
                          v-if="reach.current_cfs != null"
                          class="text-xs font-medium tabular-nums shrink-0"
                          :style="{ color: flowStatusColor(reach.flow_status) }"
                        >{{ reach.current_cfs.toLocaleString() }}</span>
                        <span v-else class="text-xs text-gray-300 dark:text-gray-600 shrink-0">—</span>
                        <!-- Add to dashboard -->
                        <button
                          v-if="reach.gauge_id"
                          class="shrink-0 p-0.5 rounded transition-opacity"
                          :class="isOnDashboard(reach)
                            ? 'text-blue-500 dark:text-blue-400 opacity-100'
                            : 'text-gray-400 dark:text-gray-500 hover:text-blue-500 dark:hover:text-blue-400 opacity-100 sm:opacity-0 sm:group-hover:opacity-100'"
                          :aria-label="isOnDashboard(reach) ? 'On dashboard' : 'Add to dashboard'"
                          @click.stop="toggleDashboard(reach)"
                        >
                          <svg v-if="isOnDashboard(reach)" class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                          </svg>
                          <svg v-else class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                            <circle cx="10" cy="10" r="8"/><line x1="10" y1="6" x2="10" y2="14"/><line x1="6" y1="10" x2="14" y2="10"/>
                          </svg>
                        </button>
                        <!-- Reach page link -->
                        <NuxtLink
                          :to="`/reaches/${reach.slug}`"
                          class="shrink-0 p-0.5 rounded text-gray-300 dark:text-gray-600 hover:text-blue-500 dark:hover:text-blue-400 transition-opacity opacity-60 sm:opacity-0 sm:group-hover:opacity-100 hover:opacity-100"
                          aria-label="View reach"
                          @click.stop
                        >
                          <svg class="w-3 h-3" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M11 3H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-5M13 3h4m0 0v4m0-4L9 11" stroke-linecap="round" stroke-linejoin="round"/>
                          </svg>
                        </NuxtLink>
                      </div>
                    </template>
                  </div>
                </template>
              </div>
            </template>
          </div>
        </div>
      </aside>

      <!-- ── Right panel: map ──────────────────────────────────────────────── -->
      <div class="flex-1 min-w-0 relative">
        <ClientOnly>
          <ReachesMap
            ref="mapRef"
            :hovered-slug="hoveredSlug"
            @reaches-updated="onReachesUpdated"
            @zoom-updated="(z) => mapZoom = z"
            @hover-changed="onMapHover"
            @reach-click="onReachClick"
          />
        </ClientOnly>

        <!-- Mobile: open list (only shown when map is visible) -->
        <button
          v-if="!listVisible"
          class="sm:hidden absolute top-2 left-2 z-20 flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium shadow-md bg-white/95 dark:bg-gray-900/95 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-200"
          @click="listVisible = true"
        >
          <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clip-rule="evenodd"/>
          </svg>
          {{ mapReaches.length }} reaches
        </button>
      </div>
    </div>
  </div>

  <!-- Gauge detail modal -->
  <GaugeDetailModal
    v-if="detailGauge"
    v-model:open="detailOpen"
    :gauge="detailGauge"
    mode="reach"
  />

  <!-- New reach modal (admin only) -->
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-100"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="authorModalOpen"
        class="fixed inset-0 z-50 flex flex-col bg-gray-50 dark:bg-gray-950 overflow-y-auto"
      >
        <!-- Modal header -->
        <div class="sticky top-0 z-10 flex items-center justify-between px-4 py-3 bg-white/90 dark:bg-gray-950/90 backdrop-blur-sm border-b border-gray-200 dark:border-gray-800">
          <h2 class="text-sm font-semibold text-gray-800 dark:text-gray-100">New reach</h2>
          <button
            class="p-1.5 rounded-md text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            @click="authorModalOpen = false"
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M6 6l8 8M14 6l-8 8"/>
            </svg>
          </button>
        </div>
        <!-- Author component -->
        <div class="flex-1 max-w-5xl w-full mx-auto px-4 py-6">
          <ReachAuthor @created="onAuthorCreated" @cancel="authorModalOpen = false" />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from 'vue'
import type { ReachListItem as MapReachItem } from '~/components/map/ReachesMap.vue'
import type { ReachListItem } from '~/components/reach/ReachBrowseRow.vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { useWatchlistStore } from '~/stores/watchlist'

definePageMeta({ ssr: false })

const { apiBase } = useRuntimeConfig().public
const router = useRouter()
const watchlistStore = useWatchlistStore()
const { addAndSync, removeAndSync } = useWatchlistSync()
const { isDataAdmin } = useAuth()

// ── New reach modal (admin only) ──────────────────────────────────────────────
const authorModalOpen = ref(false)

function onAuthorCreated(slug: string) {
  authorModalOpen.value = false
  router.push(`/reaches/${slug}/edit`)
}

// ── Demo banner ───────────────────────────────────────────────────────────────
const showDemoBanner = ref(false)
onMounted(() => {
  showDemoBanner.value = localStorage.getItem('demo-banner-dismissed') !== 'true'
})
function dismissBanner() {
  showDemoBanner.value = false
  localStorage.setItem('demo-banner-dismissed', 'true')
}

// ── Data loading ──────────────────────────────────────────────────────────────
const loading   = ref(true)
const loadError = ref('')
const reaches   = ref<ReachListItem[]>([])

onMounted(async () => {
  try {
    const res = await fetch(`${apiBase}/api/v1/reaches`)
    if (!res.ok) throw new Error(`${res.status}`)
    reaches.value = await res.json()
  } catch {
    loadError.value = 'Failed to load reaches.'
  } finally {
    loading.value = false
  }
})

// ── Search ────────────────────────────────────────────────────────────────────
const query = ref('')

// ── Tree grouping ─────────────────────────────────────────────────────────────
interface RiverGroup { name: string; reaches: ReachListItem[] }
interface BasinGroup  { name: string; reachCount: number; rivers: RiverGroup[] }
interface StateGroup  { name: string; totalCount: number; basins: BasinGroup[] }

function cleanBasinName(name: string | null | undefined): string | null {
  if (!name) return null
  const cleaned = name
    .replace(/^(Upper|Middle|Lower)\s+/i, '')
    .replace(/\s+(River|Rivers|Basin)s?$/i, '')
    .trim()
  return cleaned || null
}

function buildTree(items: ReachListItem[]): StateGroup[] {
  // state → basin → river → reaches
  const stateMap = new Map<string, Map<string, Map<string, ReachListItem[]>>>()
  for (const r of items) {
    const state = r.state_abbr ?? '—'
    const basin = cleanBasinName(r.basin) ?? 'Other'
    const river = r.river_name ?? 'Unknown River'
    if (!stateMap.has(state)) stateMap.set(state, new Map())
    const basinMap = stateMap.get(state)!
    if (!basinMap.has(basin)) basinMap.set(basin, new Map())
    const riverMap = basinMap.get(basin)!
    if (!riverMap.has(river)) riverMap.set(river, [])
    riverMap.get(river)!.push(r)
  }
  return [...stateMap.entries()]
    .sort(([a], [b]) => a === '—' ? 1 : b === '—' ? -1 : a.localeCompare(b))
    .map(([state, basinMap]) => {
      const basins = [...basinMap.entries()]
        .sort(([a], [b]) => a === 'Other' ? 1 : b === 'Other' ? -1 : a.localeCompare(b))
        .map(([basin, riverMap]) => {
          const rivers = [...riverMap.entries()]
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([name, reaches]) => ({ name, reaches }))
          const reachCount = rivers.reduce((s, r) => s + r.reaches.length, 0)
          return { name: basin, reachCount, rivers }
        })
      const totalCount = basins.reduce((s, b) => s + b.reachCount, 0)
      return { name: state, totalCount, basins }
    })
}

const allStates = computed(() => buildTree(reaches.value))

// Viewport-filtered tree: only reaches currently visible on the map
const viewportStates = computed(() => {
  if (mapReaches.value.length === 0) return allStates.value
  const visibleSlugs = new Set(mapReaches.value.map(r => r.slug))
  return buildTree(reaches.value.filter(r => visibleSlugs.has(r.slug)))
})

const filteredStates = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (q.length < 2) return viewportStates.value
  // Search spans ALL reaches, not just viewport
  const filtered = reaches.value.filter(r =>
    (r.common_name?.toLowerCase().includes(q)) ||
    (r.put_in_name?.toLowerCase().includes(q)) ||
    (r.take_out_name?.toLowerCase().includes(q)) ||
    (r.river_name?.toLowerCase().includes(q)) ||
    (r.basin?.toLowerCase().includes(q)) ||
    (r.state_abbr?.toLowerCase().includes(q)) ||
    (r.slug.toLowerCase().includes(q))
  )
  return buildTree(filtered)
})

// ── Collapse state ────────────────────────────────────────────────────────────
const collapsed = ref<{ states: Set<string>; basins: Set<string>; rivers: Set<string> }>({
  states: new Set(),
  basins: new Set(),
  rivers: new Set(),
})

function toggleState(name: string) {
  const s = new Set(collapsed.value.states)
  if (s.has(name)) s.delete(name); else s.add(name)
  collapsed.value = { ...collapsed.value, states: s }
}

function toggleBasin(key: string) {
  const s = new Set(collapsed.value.basins)
  if (s.has(key)) s.delete(key); else s.add(key)
  collapsed.value = { ...collapsed.value, basins: s }
}

function toggleRiver(state: string, basin: string, river: string) {
  const key = `${state}::${basin}::${river}`
  const s = new Set(collapsed.value.rivers)
  if (s.has(key)) s.delete(key); else s.add(key)
  collapsed.value = { ...collapsed.value, rivers: s }
}

// ── Two-way interaction: list ↔ map ───────────────────────────────────────────
const mapRef      = ref<{ flyToSlug: (slug: string) => void } | null>(null)
const hoveredSlug = ref<string | null>(null)
const mapReaches  = ref<MapReachItem[]>([])
const mapZoom     = ref(4)

const reachRefs = new Map<string, HTMLElement>()
function setReachRef(slug: string, el: HTMLElement | null) {
  if (el) reachRefs.set(slug, el)
  else    reachRefs.delete(slug)
}

function onReachesUpdated(r: MapReachItem[]) {
  mapReaches.value = r
}

// Map hover → highlight + scroll list
function onMapHover(slug: string | null) {
  hoveredSlug.value = slug
  if (slug) {
    nextTick(() => {
      reachRefs.get(slug)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    })
  }
}

// Map reach click → navigate to reach page
function onReachClick(slug: string) {
  navigateTo(`/reaches/${slug}`)
}

// List row click → fly map to reach, hide list on mobile
function onReachRowClick(reach: ReachListItem) {
  mapRef.value?.flyToSlug(reach.slug)
  listVisible.value = false
}

function flowStatusColor(status: string): string {
  const map: Record<string, string> = { runnable: '#22c55e', caution: '#ef4444', flood: '#38bdf8' }
  return map[status] ?? '#9ca3af'
}

// ── Mobile list/map toggle ────────────────────────────────────────────────────
const listVisible = ref(false)

// ── Dashboard watchlist integration ───────────────────────────────────────────
function buildWatchedGauge(reach: ReachListItem): WatchedGauge {
  return {
    id: reach.gauge_id!,
    externalId: reach.gauge_external_id ?? '',
    source: reach.gauge_source ?? 'usgs',
    name: reach.gauge_name ?? null,
    contextReachSlug: reach.slug,
    contextReachCommonName: reach.common_name ?? null,
    contextReachFullName: reach.put_in_name && reach.take_out_name
      ? `${reach.put_in_name} to ${reach.take_out_name}` : null,
    contextReachRiverName: reach.river_name ?? null,
    contextReachBasinGroup: reach.basin ?? null,
    contextReachCenterLng: null,
    contextReachRiverOrder: null,
    contextReachPermitRequired: false,
    contextReachMultiDayDays: 0,
    reachId: null, reachName: null, reachNames: [],
    reachSlug: reach.slug,
    reachSlugs: [reach.slug],
    reachCommonNames: reach.common_name ? [reach.common_name] : [],
    reachRelationship: 'primary',
    watershedName: null, basinName: reach.basin ?? null,
    riverName: reach.river_name ?? null, stateAbbr: reach.state_abbr ?? null,
    lat: null, lng: null,
    currentCfs: reach.current_cfs ?? null,
    flowStatus: reach.flow_status ?? 'unknown',
    flowBandLabel: reach.flow_label ?? null,
    lastReadingAt: null,
    watchState: 'saved', activeSince: null,
  }
}

function isOnDashboard(reach: ReachListItem): boolean {
  if (!reach.gauge_id) return false
  return watchlistStore.gauges.some(g => g.id === reach.gauge_id && g.contextReachSlug === reach.slug)
}

function toggleDashboard(reach: ReachListItem) {
  if (!reach.gauge_id) return
  if (isOnDashboard(reach)) {
    removeAndSync(reach.gauge_id, reach.slug)
  } else {
    addAndSync(buildWatchedGauge(reach))
  }
}

function addAllRiverReaches(reaches: ReachListItem[]) {
  for (const reach of reaches) {
    if (reach.gauge_id && !isOnDashboard(reach)) {
      addAndSync(buildWatchedGauge(reach))
    }
  }
}

// ── Gauge detail modal ────────────────────────────────────────────────────────
const detailOpen  = ref(false)
const detailGauge = ref<WatchedGauge | null>(null)

function openGaugeModal(reach: ReachListItem) {
  if (!reach.gauge_id) return
  detailGauge.value = buildWatchedGauge(reach)
  detailOpen.value = true
}

</script>
