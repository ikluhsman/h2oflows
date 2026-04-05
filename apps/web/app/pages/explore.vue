<template>
  <div class="flex flex-col h-screen overflow-hidden bg-gray-50 dark:bg-gray-950">

    <AppHeader>
      <template #actions>
        <NuxtLink
          to="/dashboard"
          class="text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 transition-colors flex items-center gap-1"
        >
          My Dashboard
          <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M5 12h14M12 5l7 7-7 7"/>
          </svg>
        </NuxtLink>
      </template>
    </AppHeader>

    <!-- Hero bar -->
    <div class="shrink-0 px-4 py-4 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950">
      <div class="max-w-7xl mx-auto">
        <h1 class="text-xl font-bold tracking-tight">Find your run.</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
          Real-time streamflow for Colorado's rivers. Search a gauge or pan the map to explore.
        </p>
      </div>
    </div>

    <!-- Body: sidebar + map -->
    <div class="flex flex-1 overflow-hidden">

      <!-- Left panel -->
      <div class="w-72 shrink-0 flex flex-col border-r border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 overflow-y-auto">

        <!-- Search input -->
        <div class="p-3 border-b border-gray-100 dark:border-gray-800">
          <UInput
            v-model="query"
            placeholder="Search river, gauge, section…"
            icon="i-heroicons-magnifying-glass"
            size="sm"
            @input="onInput"
          />
        </div>

        <!-- Search results -->
        <div v-if="searching" class="p-4 text-center text-sm text-gray-400">Searching…</div>
        <div v-else-if="results.length === 0 && query.length >= 2" class="p-4 text-center text-sm text-gray-400">
          No results for "{{ query }}"
        </div>
        <ul v-else-if="results.length > 0" class="divide-y divide-gray-100 dark:divide-gray-800">
          <li
            v-for="g in results"
            :key="g.id"
            class="px-3 py-2.5 hover:bg-gray-50 dark:hover:bg-gray-900 flex items-center gap-2"
          >
            <span class="shrink-0 w-2 h-2 rounded-full" :style="{ background: flowDotColor(g.flowStatus) }" />
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium truncate">{{ g.name ?? g.externalId }}</p>
              <p v-if="g.reachName" class="text-xs text-gray-500 truncate">{{ g.reachName }}</p>
              <p class="text-xs text-gray-400 truncate">
                {{ g.currentCfs != null ? `${Number(g.currentCfs).toLocaleString()} cfs` : '—' }}
                · {{ g.source.toUpperCase() }}
              </p>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <UButton
                v-if="g.reachSlug"
                size="xs" color="neutral" variant="ghost" icon="i-heroicons-map-pin"
                :to="`/reaches/${g.reachSlug}`"
              />
              <UButton
                size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                @click="addGauge(g)"
              />
            </div>
          </li>
        </ul>

        <!-- Visible gauges (map viewport) -->
        <template v-else>
          <div v-if="loadingGauges" class="p-4 text-center text-sm text-gray-400">Loading gauges…</div>
          <div v-else-if="visibleGauges.length === 0" class="p-3 text-xs text-gray-400 dark:text-gray-500">
            Pan or zoom the map to explore gauges.
          </div>
          <ul v-else class="divide-y divide-gray-100 dark:divide-gray-800">
            <li
              v-for="g in visibleGauges"
              :key="g.id"
              class="flex items-center gap-2 px-3 py-2.5 hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors cursor-pointer"
              @click="g.reachSlug && reachesMapRef?.flyToSlug(g.reachSlug)"
            >
              <span class="shrink-0 w-2 h-2 rounded-full" :style="{ background: flowDotColor(g.flowStatus) }" />
              <div class="flex-1 min-w-0">
                <p class="text-sm text-gray-700 dark:text-gray-300 truncate font-medium">{{ g.name ?? g.externalId }}</p>
                <p class="text-xs text-gray-400 truncate">
                  {{ g.currentCfs != null ? `${Number(g.currentCfs).toLocaleString()} cfs` : '—' }}
                  <span v-if="g.reachName"> · {{ g.reachName }}</span>
                </p>
              </div>
              <div class="flex items-center gap-1 shrink-0">
                <UButton
                  v-if="g.reachSlug"
                  size="xs" color="neutral" variant="ghost" icon="i-heroicons-arrow-right"
                  :to="`/reaches/${g.reachSlug}`"
                  @click.stop
                />
                <UButton
                  size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                  @click.stop="addGauge(g)"
                />
              </div>
            </li>
          </ul>
        </template>

        <!-- Dashboard link -->
        <div class="mt-auto p-3 border-t border-gray-100 dark:border-gray-800">
          <NuxtLink
            to="/dashboard"
            class="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
              <rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>
            </svg>
            My Dashboard
          </NuxtLink>
        </div>
      </div>

      <!-- Map -->
      <div class="flex-1 overflow-hidden">
        <ClientOnly>
          <ReachesMap
            ref="reachesMapRef"
            :hovered-slug="hoveredSlug"
            @bounds-updated="onBoundsUpdated"
            @gauge-add="addGaugeById"
          />
        </ClientOnly>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'

definePageMeta({ ssr: false })

const store = useWatchlistStore()
const { apiBase } = useRuntimeConfig().public

// ── Gauge text search ─────────────────────────────────────────────────────────

const query    = ref('')
const searching = ref(false)
const results  = ref<Omit<WatchedGauge, 'watchState' | 'activeSince'>[]>([])
let searchDebounce: ReturnType<typeof setTimeout>

function onInput() {
  clearTimeout(searchDebounce)
  if (query.value.length < 2) { results.value = []; return }
  searchDebounce = setTimeout(search, 300)
}

async function search() {
  searching.value = true
  try {
    const res = await fetch(`${apiBase}/api/v1/gauges/search?q=${encodeURIComponent(query.value)}&limit=20`)
    if (!res.ok) return
    const data = await res.json()
    results.value = (data.features ?? []).map(mapFeature)
  } catch {
    results.value = []
  } finally {
    searching.value = false
  }
}

// ── Visible gauges (driven by map bounds) ─────────────────────────────────────

const reachesMapRef  = ref<{ flyToSlug: (slug: string) => void } | null>(null)
const hoveredSlug    = ref<string | null>(null)
const visibleGauges  = ref<Omit<WatchedGauge, 'watchState' | 'activeSince'>[]>([])
const loadingGauges  = ref(false)
let bboxDebounce: ReturnType<typeof setTimeout>

function onBoundsUpdated(bbox: string) {
  clearTimeout(bboxDebounce)
  bboxDebounce = setTimeout(() => fetchGaugesByBbox(bbox), 400)
}

async function fetchGaugesByBbox(bbox: string) {
  loadingGauges.value = true
  try {
    const res = await fetch(`${apiBase}/api/v1/gauges/search?bbox=${bbox}&limit=60`)
    if (!res.ok) return
    const data = await res.json()
    visibleGauges.value = (data.features ?? []).map(mapFeature)
  } catch {
    visibleGauges.value = []
  } finally {
    loadingGauges.value = false
  }
}

// ── Shared helpers ────────────────────────────────────────────────────────────

function mapFeature(f: any): Omit<WatchedGauge, 'watchState' | 'activeSince'> {
  return {
    id:               f.properties.id,
    externalId:       f.properties.external_id,
    source:           f.properties.source,
    name:             f.properties.name ?? null,
    featured:         f.properties.featured ?? false,
    reachId:          f.properties.reach_id ?? null,
    reachName:        f.properties.reach_name ?? null,
    reachNames:       f.properties.reach_names ?? [],
    reachSlug:        f.properties.reach_slug ?? null,
    reachSlugs:       f.properties.reach_slugs ?? [],
    reachRelationship: f.properties.reach_relationship ?? null,
    pollTier:         f.properties.poll_tier,
    watershedName:    f.properties.watershed_name ?? null,
    basinName:        f.properties.basin_name ?? null,
    riverName:        f.properties.river_name ?? null,
    stateAbbr:        f.properties.state_abbr ?? null,
    lng:              f.geometry?.coordinates?.[0] ?? null,
    lat:              f.geometry?.coordinates?.[1] ?? null,
    currentCfs:       f.properties.current_cfs ?? null,
    flowStatus:       f.properties.flow_status ?? 'unknown',
    flowBandLabel:    f.properties.flow_band_label ?? null,
    lastReadingAt:    f.properties.last_reading_at ?? null,
  }
}

function addGauge(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
  store.addGauge(gauge)
  query.value = ''
  results.value = []
}

async function addGaugeById(gaugeId: string) {
  try {
    const res = await fetch(`${apiBase}/api/v1/gauges/batch?ids=${gaugeId}`)
    if (!res.ok) return
    const data = await res.json()
    const f = data.features?.[0]
    if (!f) return
    const p = f.properties
    const coords = f.geometry?.coordinates as [number, number] | undefined
    store.addGauge({
      id: p.id, externalId: p.external_id, source: p.source,
      name: p.name ?? null, featured: p.featured ?? false,
      reachId: p.reach_id ?? null, reachName: p.reach_name ?? null,
      reachNames: p.reach_names ?? [], reachSlug: p.reach_slug ?? null,
      reachSlugs: p.reach_slugs ?? [], reachRelationship: p.reach_relationship ?? null,
      pollTier: p.poll_tier, watershedName: p.watershed_name ?? null,
      basinName: p.basin_name ?? null, riverName: p.river_name ?? null,
      stateAbbr: p.state_abbr ?? null,
      lng: coords?.[0] ?? null, lat: coords?.[1] ?? null,
      currentCfs: p.current_cfs ?? null, flowStatus: p.flow_status ?? 'unknown',
      flowBandLabel: p.flow_band_label ?? null, lastReadingAt: p.last_reading_at ?? null,
    })
  } catch { /* non-fatal */ }
}

function flowDotColor(status: string): string {
  return ({ runnable: '#16a34a', caution: '#d97706', low: '#9ca3af', flood: '#dc2626' } as Record<string, string>)[status] ?? '#9ca3af'
}
</script>
