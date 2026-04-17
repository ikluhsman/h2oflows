<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-3xl mx-auto px-4 py-6 pb-20 sm:pb-6 space-y-4">

      <!-- Page header -->
      <div>
        <h1 class="text-xl font-bold tracking-tight">Rivers</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
          Browse Colorado's whitewater by basin and river.
        </p>
      </div>

      <!-- Search -->
      <div>
        <UInput
          v-model="query"
          placeholder="Search reaches, rivers, or basins…"
          icon="i-heroicons-magnifying-glass"
          size="sm"
          class="max-w-md"
        />
      </div>

      <!-- Loading state -->
      <div v-if="loading" class="py-12 text-center text-sm text-gray-400">Loading reaches…</div>

      <!-- Error state -->
      <div v-else-if="error" class="py-12 text-center text-sm text-red-400">{{ error }}</div>

      <!-- Empty search -->
      <div v-else-if="query.length >= 2 && filteredBasins.length === 0" class="py-12 text-center text-sm text-gray-400">
        No results for "{{ query }}"
      </div>

      <!-- Tree -->
      <template v-else>
        <section v-for="basin in filteredBasins" :key="basin.name" class="mb-2">
          <!-- Basin header -->
          <button
            class="flex items-center gap-2 mb-1 w-full text-left py-1"
            @click="toggleBasin(basin.name)"
          >
            <svg
              class="w-3 h-3 text-gray-400 transition-transform duration-200"
              :class="{ 'rotate-90': !collapsed.basins.has(basin.name) }"
              viewBox="0 0 20 20" fill="currentColor"
            >
              <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
            </svg>
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white uppercase tracking-wide">{{ basin.name }}</h2>
            <span class="text-xs text-gray-400">({{ basin.reachCount }})</span>
            <div class="flex-1 h-px bg-gray-200 dark:bg-gray-800" />
          </button>

          <template v-if="!collapsed.basins.has(basin.name)">
            <div v-for="river in basin.rivers" :key="river.name" class="ml-2 mb-1">
              <!-- River header -->
              <button
                class="flex items-center gap-2 w-full text-left py-1"
                @click="toggleRiver(basin.name, river.name)"
              >
                <svg
                  class="w-2.5 h-2.5 text-gray-400 transition-transform duration-200"
                  :class="{ 'rotate-90': !collapsed.rivers.has(`${basin.name}::${river.name}`) }"
                  viewBox="0 0 20 20" fill="currentColor"
                >
                  <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
                </svg>
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ river.name }}</span>
                <span class="text-xs text-gray-400">({{ river.reaches.length }})</span>
              </button>

              <!-- Reach rows -->
              <div
                v-if="!collapsed.rivers.has(`${basin.name}::${river.name}`)"
                class="ml-3 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden divide-y divide-gray-100 dark:divide-gray-800"
              >
                <ReachBrowseRow
                  v-for="reach in river.reaches"
                  :key="reach.slug"
                  :reach="reach"
                  @open-gauge="openGaugeModal"
                />
              </div>
            </div>
          </template>
        </section>
      </template>
    </main>

    <!-- Gauge detail modal -->
    <GaugeDetailModal
      v-if="detailGauge"
      v-model:open="detailOpen"
      :gauge="detailGauge"
      mode="reach"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { ReachListItem } from '~/components/reach/ReachBrowseRow.vue'
import type { WatchedGauge } from '~/stores/watchlist'

definePageMeta({ ssr: false })

const { apiBase } = useRuntimeConfig().public

// ── Data loading ─────────────────────────────────────────────────────────────

const loading = ref(true)
const error = ref('')
const reaches = ref<ReachListItem[]>([])

onMounted(async () => {
  try {
    const res = await fetch(`${apiBase}/api/v1/reaches`)
    if (!res.ok) throw new Error(`${res.status}`)
    reaches.value = await res.json()
  } catch {
    error.value = 'Failed to load reaches.'
  } finally {
    loading.value = false
  }
})

// ── Search ───────────────────────────────────────────────────────────────────

const query = ref('')

// ── Grouping: basin → river → reaches ────────────────────────────────────────

interface RiverGroup {
  name: string
  reaches: ReachListItem[]
}

interface BasinGroup {
  name: string
  reachCount: number
  rivers: RiverGroup[]
}

function buildTree(items: ReachListItem[]): BasinGroup[] {
  const basinMap = new Map<string, Map<string, ReachListItem[]>>()

  for (const r of items) {
    const basin = r.basin ?? 'Other'
    const river = r.river_name ?? 'Unknown River'
    if (!basinMap.has(basin)) basinMap.set(basin, new Map())
    const riverMap = basinMap.get(basin)!
    if (!riverMap.has(river)) riverMap.set(river, [])
    riverMap.get(river)!.push(r)
  }

  return [...basinMap.entries()]
    .sort(([a], [b]) => a === 'Other' ? 1 : b === 'Other' ? -1 : a.localeCompare(b))
    .map(([basin, riverMap]) => {
      const rivers = [...riverMap.entries()]
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([name, reaches]) => ({ name, reaches }))
      const reachCount = rivers.reduce((s, r) => s + r.reaches.length, 0)
      return { name: basin, reachCount, rivers }
    })
}

const allBasins = computed(() => buildTree(reaches.value))

const filteredBasins = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (q.length < 2) return allBasins.value

  // Filter reaches that match, then rebuild the tree
  const filtered = reaches.value.filter(r =>
    (r.common_name?.toLowerCase().includes(q)) ||
    (r.put_in_name?.toLowerCase().includes(q)) ||
    (r.take_out_name?.toLowerCase().includes(q)) ||
    (r.river_name?.toLowerCase().includes(q)) ||
    (r.basin?.toLowerCase().includes(q)) ||
    (r.slug.toLowerCase().includes(q))
  )
  return buildTree(filtered)
})

// ── Collapse state ───────────────────────────────────────────────────────────

const collapsed = ref<{ basins: Set<string>; rivers: Set<string> }>({
  basins: new Set(),
  rivers: new Set(),
})

function toggleBasin(name: string) {
  const s = new Set(collapsed.value.basins)
  if (s.has(name)) s.delete(name); else s.add(name)
  collapsed.value = { ...collapsed.value, basins: s }
}

function toggleRiver(basin: string, river: string) {
  const key = `${basin}::${river}`
  const s = new Set(collapsed.value.rivers)
  if (s.has(key)) s.delete(key); else s.add(key)
  collapsed.value = { ...collapsed.value, rivers: s }
}

// ── Gauge modal ──────────────────────────────────────────────────────────────

const detailOpen = ref(false)
const detailGauge = ref<WatchedGauge | null>(null)

function openGaugeModal(reach: ReachListItem) {
  if (!reach.gauge_id) return

  // Construct a minimal WatchedGauge for the modal
  detailGauge.value = {
    id: reach.gauge_id,
    externalId: reach.gauge_external_id ?? '',
    source: reach.gauge_source ?? 'usgs',
    name: reach.gauge_name ?? null,
    contextReachSlug: reach.slug,
    contextReachCommonName: reach.common_name ?? null,
    contextReachFullName: reach.put_in_name && reach.take_out_name
      ? `${reach.put_in_name} to ${reach.take_out_name}`
      : null,
    contextReachRiverName: reach.river_name ?? null,
    contextReachBasinGroup: reach.basin ?? null,
    contextReachPermitRequired: false,
    contextReachMultiDayDays: 0,
    reachId: null,
    reachName: null,
    reachNames: [],
    reachSlug: reach.slug,
    reachSlugs: [reach.slug],
    reachCommonNames: reach.common_name ? [reach.common_name] : [],
    reachRelationship: 'primary',
    watershedName: null,
    basinName: reach.basin ?? null,
    riverName: reach.river_name ?? null,
    stateAbbr: null,
    lat: null,
    lng: null,
    currentCfs: reach.current_cfs ?? null,
    flowStatus: reach.flow_status ?? 'unknown',
    flowBandLabel: reach.flow_label ?? null,
    lastReadingAt: null,
    watchState: 'saved',
    activeSince: null,
  }
  detailOpen.value = true
}
</script>
