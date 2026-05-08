<template>
  <UModal v-model:open="open" title="Add a reach or gauge" :ui="{ width: 'max-w-2xl' }">
    <template #body>
      <div class="space-y-4">

        <!-- Dashboard picker — shown when user has multiple dashboards -->
        <div
          v-if="db.dashboards.value.length > 1"
          class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg bg-primary-50 dark:bg-primary-950/30 border border-primary-100 dark:border-primary-900/40"
        >
          <div class="flex items-center gap-2 min-w-0">
            <svg class="w-4 h-4 text-primary-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="4" rx="1"/><rect x="14" y="10" width="7" height="11" rx="1"/><rect x="3" y="13" width="7" height="8" rx="1"/>
            </svg>
            <span class="text-xs text-neutral-600 dark:text-neutral-300 shrink-0">Add to:</span>
          </div>
          <select
            v-model="selectedDashboardId"
            class="flex-1 max-w-[60%] rounded-md border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-sm text-neutral-700 dark:text-neutral-200 px-2 py-1"
          >
            <option v-for="d in db.dashboards.value" :key="d.id" :value="d.id">{{ d.name }}</option>
          </select>
        </div>

        <!-- Tabs + import button -->
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-1 bg-neutral-100 dark:bg-neutral-800 rounded-lg p-1">
            <button
              v-for="t in TABS" :key="t.key"
              class="px-3 py-1 text-xs font-medium rounded-md transition-colors"
              :class="activeTab === t.key
                ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-white shadow-sm'
                : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200'"
              @click="setTab(t.key)"
            >{{ t.label }}</button>
          </div>
          <button
            class="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium rounded-lg border border-neutral-200 dark:border-neutral-700 text-neutral-500 dark:text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400 hover:border-primary-300 dark:hover:border-primary-700 transition-colors"
            @click="importOpen = true"
          >
            <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clip-rule="evenodd"/></svg>
            Import
          </button>
        </div>

        <!-- ── Curated tab ── -->
        <template v-if="activeTab === 'curated'">
          <UInput
            v-model="query"
            placeholder="Search by river, section, or gauge ID…"
            icon="i-heroicons-magnifying-glass"
            size="lg"
            autofocus
            @input="onInput"
          />

          <div class="flex gap-4">
            <div class="flex-1 min-w-0">
              <div v-if="loading" class="space-y-2 py-2">
                <div v-for="i in 4" :key="i" class="flex items-center gap-3 px-2 py-2.5">
                  <div class="flex-1 space-y-2">
                    <div class="h-4 w-3/4 rounded bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
                    <div class="h-3 w-1/2 rounded bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
                  </div>
                  <div class="h-7 w-14 rounded bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
                </div>
              </div>
              <div v-else-if="results.length === 0 && query.length >= 2" class="text-center py-10 text-neutral-400 text-sm">
                No gauges found for "{{ query }}"
              </div>
              <div v-else-if="results.length === 0" class="text-center py-10 text-neutral-400 text-sm">
                Type to search rivers, sections, or gauge IDs
              </div>
              <ul v-else class="divide-y divide-neutral-100 dark:divide-neutral-800 max-h-[60vh] overflow-y-auto">
                <template v-for="g in results" :key="g.id">
                  <template v-if="g.reachSlugs.length">
                    <li
                      v-for="(slug, i) in g.reachSlugs"
                      :key="slug"
                      class="flex items-center justify-between gap-3 py-2.5 px-2 hover:bg-primary-50 dark:hover:bg-primary-950/30 rounded-lg transition-colors cursor-pointer"
                      @click="selectWithContext(g, slug, g.reachCommonNames[i] ?? g.reachNames[i] ?? null)"
                    >
                      <div class="min-w-0 flex-1">
                        <p class="text-sm font-medium text-neutral-900 dark:text-neutral-100 truncate">{{ g.reachCommonNames[i] ?? g.reachNames[i] ?? slug }}</p>
                        <div class="flex items-center gap-1.5 mt-0.5">
                          <span v-if="g.riverName" class="text-xs text-neutral-500 dark:text-neutral-400 truncate">{{ g.riverName }}</span>
                          <span v-if="g.riverName" class="text-neutral-300 dark:text-neutral-600 text-xs">·</span>
                          <span class="text-xs text-neutral-400 truncate">
                            {{ g.source.toUpperCase() }} {{ g.externalId }}<template v-if="g.stateAbbr">, {{ g.stateAbbr }}</template>
                          </span>
                        </div>
                      </div>
                      <div class="flex items-center gap-2 shrink-0">
                        <span v-if="g.currentCfs != null" class="text-sm font-semibold tabular-nums text-neutral-700 dark:text-neutral-300">
                          {{ g.currentCfs.toLocaleString() }}<span class="text-xs font-normal text-neutral-400 ml-0.5">cfs</span>
                        </span>
                        <UButton size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                          @click.stop="selectWithContext(g, slug, g.reachCommonNames[i] ?? g.reachNames[i] ?? null)"
                        >Add</UButton>
                      </div>
                    </li>
                  </template>
                  <li
                    v-else
                    class="flex items-center justify-between gap-3 py-2.5 px-2 hover:bg-primary-50 dark:hover:bg-primary-950/30 rounded-lg transition-colors cursor-pointer"
                    @click="selectWithContext(g, null, null)"
                  >
                    <div class="min-w-0 flex-1">
                      <p class="text-sm font-medium text-neutral-900 dark:text-neutral-100 truncate">{{ g.name ?? g.externalId }}</p>
                      <div class="flex items-center gap-1.5 mt-0.5">
                        <span v-if="g.riverName" class="text-xs text-neutral-500 dark:text-neutral-400 truncate">{{ g.riverName }}</span>
                        <span v-if="g.riverName" class="text-neutral-300 dark:text-neutral-600 text-xs">·</span>
                        <span class="text-xs text-neutral-400 truncate">
                          {{ g.source.toUpperCase() }} {{ g.externalId }}<template v-if="g.stateAbbr">, {{ g.stateAbbr }}</template>
                        </span>
                      </div>
                    </div>
                    <div class="flex items-center gap-2 shrink-0">
                      <span v-if="g.currentCfs != null" class="text-sm font-semibold tabular-nums text-neutral-700 dark:text-neutral-300">
                        {{ g.currentCfs.toLocaleString() }}<span class="text-xs font-normal text-neutral-400 ml-0.5">cfs</span>
                      </span>
                      <UButton size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                        @click.stop="selectWithContext(g, null, null)"
                      >Add</UButton>
                    </div>
                  </li>
                </template>
              </ul>
            </div>
          </div>
        </template>

        <!-- ── My Reaches & Gauges tab ── -->
        <template v-else-if="activeTab === 'mine'">
          <div class="max-h-[60vh] overflow-y-auto space-y-5">

            <!-- Reaches sub-section -->
            <div>
              <p class="text-[10px] font-semibold uppercase tracking-wider text-neutral-400 dark:text-neutral-500 mb-2">My Reaches</p>
              <div v-if="reachesLoading" class="space-y-2">
                <div v-for="i in 3" :key="i" class="flex items-center gap-3 px-2 py-2">
                  <div class="flex-1 h-4 rounded bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
                  <div class="h-7 w-20 rounded bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
                </div>
              </div>
              <div v-else-if="myReaches.length === 0" class="px-2 py-3 text-sm text-neutral-400">
                No personal reaches yet.
                <NuxtLink to="/my/reaches/new" class="ml-1 text-primary-500 hover:text-primary-700 transition-colors font-medium" @click="open = false">Create one →</NuxtLink>
              </div>
              <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                <div
                  v-for="r in myReaches" :key="r.id"
                  class="rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 p-3 space-y-2"
                >
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-1.5">
                        <svg class="w-3 h-3 shrink-0 text-neutral-400/60" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd"/></svg>
                        <p class="text-sm font-medium text-neutral-900 dark:text-neutral-100 truncate">{{ r.name }}</p>
                      </div>
                      <p v-if="r.river_name" class="text-xs text-neutral-400 truncate mt-0.5">{{ r.river_name }}</p>
                    </div>
                    <span v-if="r.current_cfs != null" class="text-sm font-semibold tabular-nums text-neutral-700 dark:text-neutral-300 shrink-0">
                      {{ r.current_cfs.toLocaleString() }}<span class="text-xs font-normal text-neutral-400 ml-0.5">cfs</span>
                    </span>
                  </div>
                  <div class="flex items-center gap-1.5">
                    <UButton
                      v-if="r.gauge_id || r.custom_gauge_id"
                      size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                      :loading="adding === r.id"
                      @click="addUserReach(r)"
                    >Add</UButton>
                    <NuxtLink
                      :to="`/my/reaches/${r.slug}`"
                      class="inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs font-medium text-neutral-500 dark:text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
                      @click="open = false"
                    >Open</NuxtLink>
                  </div>
                </div>
              </div>
              <NuxtLink to="/my/reaches/new" class="flex items-center gap-1.5 px-2 py-1.5 mt-2 text-xs text-primary-500 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors font-medium" @click="open = false">
                <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd"/></svg>
                New reach
              </NuxtLink>
            </div>

            <!-- Custom gauges sub-section -->
            <div>
              <p class="text-[10px] font-semibold uppercase tracking-wider text-neutral-400 dark:text-neutral-500 mb-2">Custom Gauges</p>
              <div v-if="gaugesLoading" class="space-y-2">
                <div v-for="i in 3" :key="i" class="flex items-center gap-3 px-2 py-2">
                  <div class="flex-1 h-4 rounded bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
                  <div class="h-7 w-20 rounded bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
                </div>
              </div>
              <div v-else-if="myGauges.length === 0" class="px-2 py-3 text-sm text-neutral-400">
                No custom gauges yet.
                <NuxtLink to="/my/gauges/new" class="ml-1 text-primary-500 hover:text-primary-700 transition-colors font-medium" @click="open = false">Create one →</NuxtLink>
              </div>
              <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                <div
                  v-for="cg in myGauges" :key="cg.id"
                  class="rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 p-3 space-y-2"
                >
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-1.5">
                        <svg class="w-3.5 h-3.5 text-neutral-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="2" width="16" height="20" rx="2"/><line x1="8" y1="6" x2="16" y2="6"/><line x1="8" y1="10" x2="10" y2="10"/><line x1="14" y1="10" x2="16" y2="10"/><line x1="8" y1="14" x2="10" y2="14"/><line x1="14" y1="14" x2="16" y2="14"/><line x1="8" y1="18" x2="10" y2="18"/><line x1="14" y1="18" x2="16" y2="18"/></svg>
                        <p class="text-sm font-medium text-neutral-900 dark:text-neutral-100 truncate">{{ cg.name }}</p>
                      </div>
                      <p v-if="cg.description" class="text-xs text-neutral-400 truncate mt-0.5">{{ cg.description }}</p>
                    </div>
                    <span v-if="cg.last_value_cfs != null" class="text-sm font-semibold tabular-nums text-neutral-700 dark:text-neutral-300 shrink-0">
                      {{ cg.last_value_cfs.toLocaleString() }}<span class="text-xs font-normal text-neutral-400 ml-0.5">{{ cg.unit }}</span>
                    </span>
                  </div>
                  <div class="flex items-center gap-1.5">
                    <UButton
                      size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                      :loading="adding === cg.id"
                      @click="addCustomGauge(cg)"
                    >Add</UButton>
                    <NuxtLink
                      :to="`/my/gauges/${cg.slug}`"
                      class="inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs font-medium text-neutral-500 dark:text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
                      @click="open = false"
                    >Open</NuxtLink>
                  </div>
                </div>
              </div>
              <NuxtLink to="/my/gauges/new" class="flex items-center gap-1.5 px-2 py-1.5 mt-2 text-xs text-primary-500 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors font-medium" @click="open = false">
                <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd"/></svg>
                New gauge
              </NuxtLink>
            </div>

          </div>
        </template>

      </div>
    </template>
    <template #footer>
      <div class="flex justify-end">
        <UButton variant="ghost" color="neutral" size="sm" @click="open = false">Cancel</UButton>
      </div>
    </template>
  </UModal>

  <!-- Import dialog -->
  <UModal v-model:open="importOpen" title="Import from share code">
    <template #body>
      <div class="space-y-3">
        <p class="text-sm text-neutral-500 dark:text-neutral-400">Paste a share code to import a reach someone shared with you.</p>
        <UTextarea v-model="importPayload" placeholder="Paste share code here…" :rows="6" autofocus />
        <p v-if="importError" class="text-xs text-red-500">{{ importError }}</p>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton variant="ghost" color="neutral" size="sm" @click="importOpen = false">Cancel</UButton>
        <UButton color="primary" size="sm" :loading="importLoading" :disabled="!importPayload.trim()" @click="doImport">Import</UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { featureToWatchedGauge } from '~/composables/useWatchlistSync'

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{
  (e: 'add', gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>, dashboardId: string | null): void
  (e: 'addedExternal'): void
}>()

const { apiBase } = useRuntimeConfig().public
const { getToken } = useAuth()
const db = useDashboards()
const { addCustomGaugeToWatchlist, addUserReachToWatchlist } = useWatchlistSync()

// Make sure dashboards are loaded for the picker
onMounted(() => { if (!db.loaded.value) db.load() })

// Picker state — defaults to active dashboard, follows it as it changes
const selectedDashboardId = ref<string | null>(db.activeDashboardId.value)
watch(() => db.activeDashboardId.value, (id) => {
  if (!selectedDashboardId.value || !db.dashboards.value.find(d => d.id === selectedDashboardId.value)) {
    selectedDashboardId.value = id
  }
})

// ── Tabs ─────────────────────────────────────────────────────────────────────

type TabKey = 'curated' | 'mine'
const TABS: { key: TabKey; label: string }[] = [
  { key: 'curated', label: 'H2OFlows'          },
  { key: 'mine',    label: 'My Reaches & Gauges' },
]
const activeTab = ref<TabKey>('curated')

function setTab(key: TabKey) {
  activeTab.value = key
  if (key === 'mine') {
    if (myReaches.value.length === 0 && !reachesLoading.value) loadMyReaches()
    if (myGauges.value.length === 0  && !gaugesLoading.value)  loadMyGauges()
  }
}

// Reset tab when modal closes; default selected dashboard back to active when opening
watch(open, (v) => {
  if (!v) { activeTab.value = 'curated'; query.value = ''; results.value = [] }
  else if (db.activeDashboardId.value) selectedDashboardId.value = db.activeDashboardId.value
})

// ── Curated search ────────────────────────────────────────────────────────────

const query    = ref('')
const loading  = ref(false)
const results  = ref<Omit<WatchedGauge, 'watchState' | 'activeSince'>[]>([])

let debounceTimer: ReturnType<typeof setTimeout>

function onInput() {
  clearTimeout(debounceTimer)
  if (query.value.length < 2) { results.value = []; return }
  debounceTimer = setTimeout(search, 300)
}

async function search() {
  loading.value = true
  try {
    const url = `${apiBase}/api/v1/gauges/search?q=${encodeURIComponent(query.value)}&limit=20`
    const res = await fetch(url)
    if (!res.ok) return
    const data = await res.json()
    results.value = (data.features ?? []).map((f: any) => {
      const coords = f.geometry?.coordinates as [number, number] | undefined
      return featureToWatchedGauge(f.properties, coords)
    })
  } catch {
    results.value = []
  } finally {
    loading.value = false
  }
}

function selectWithContext(
  gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>,
  reachSlug: string | null,
  reachCommonName: string | null,
) {
  const idx = reachSlug ? gauge.reachSlugs.indexOf(reachSlug) : -1
  const enriched: Omit<WatchedGauge, 'watchState' | 'activeSince'> = {
    ...gauge,
    contextReachSlug:       reachSlug,
    contextReachCommonName: reachCommonName,
    contextReachFullName:   null,
    contextReachRiverName:  idx >= 0 ? (gauge.riverName ?? null) : null,
  }
  emit('add', enriched, selectedDashboardId.value)
  open.value = false
  query.value = ''
  results.value = []
}

// ── My Reaches ────────────────────────────────────────────────────────────────

interface ReachSummary {
  id: string; slug: string; name: string; river_name: string | null
  current_cfs: number | null; flow_band: string | null
  gauge_id: string | null; custom_gauge_id: string | null
}
const myReaches      = ref<ReachSummary[]>([])
const reachesLoading = ref(false)

async function loadMyReaches() {
  reachesLoading.value = true
  try {
    const token = await getToken()
    if (!token) return
    const res = await fetch(`${apiBase}/api/v1/me/reaches`, { headers: { Authorization: `Bearer ${token}` } })
    if (res.ok) myReaches.value = await res.json() ?? []
  } catch { /* non-fatal */ } finally {
    reachesLoading.value = false
  }
}

// ── My Gauges ─────────────────────────────────────────────────────────────────

interface GaugeSummary { id: string; slug: string; name: string; description: string | null; unit: string; last_value_cfs: number | null }
const myGauges      = ref<GaugeSummary[]>([])
const gaugesLoading = ref(false)

async function loadMyGauges() {
  gaugesLoading.value = true
  try {
    const token = await getToken()
    if (!token) return
    const res = await fetch(`${apiBase}/api/v1/me/custom-gauges`, { headers: { Authorization: `Bearer ${token}` } })
    if (res.ok) { const d = await res.json(); myGauges.value = d.items ?? [] }
  } catch { /* non-fatal */ } finally {
    gaugesLoading.value = false
  }
}

// ── Add handlers (user reach / custom gauge → watchlist) ─────────────────────

const adding = ref<string | null>(null)

async function addUserReach(r: ReachSummary) {
  if (!r.gauge_id) return  // user reach without a real gauge can't be watchlisted yet
  adding.value = r.id
  try {
    await addUserReachToWatchlist(r.gauge_id, r.slug, selectedDashboardId.value)
    emit('addedExternal')
    open.value = false
  } finally {
    adding.value = null
  }
}

async function addCustomGauge(cg: GaugeSummary) {
  adding.value = cg.id
  try {
    await addCustomGaugeToWatchlist(cg.id, selectedDashboardId.value)
    emit('addedExternal')
    open.value = false
  } finally {
    adding.value = null
  }
}

// ── Import ────────────────────────────────────────────────────────────────────

const importOpen    = ref(false)
const importPayload = ref('')
const importLoading = ref(false)
const importError   = ref('')

async function doImport() {
  importError.value = ''
  importLoading.value = true
  try {
    const payload = JSON.parse(importPayload.value.trim())
    const token = await getToken()
    if (!token) { importError.value = 'Sign in to import reaches.'; return }
    const res = await fetch(`${apiBase}/api/v1/me/reaches/import`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      importError.value = err.error ?? `Import failed (${res.status})`
      return
    }
    importOpen.value = false
    importPayload.value = ''
    open.value = false
    myReaches.value = [] // force reload next open
  } catch {
    importError.value = 'Invalid share code — paste the full JSON payload.'
  } finally {
    importLoading.value = false
  }
}
</script>
