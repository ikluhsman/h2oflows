<template>
  <UModal v-model:open="open" title="Add a gauge" :ui="{ width: 'max-w-lg' }">
    <template #body>
      <div class="space-y-4">
        <UInput
          v-model="query"
          placeholder="Search by river, section, or gauge ID…"
          icon="i-heroicons-magnifying-glass"
          autofocus
          @input="onInput"
        />

        <!-- Results -->
        <div v-if="loading" class="text-center py-6 text-gray-400 text-sm">Searching…</div>

        <div v-else-if="results.length === 0 && query.length >= 2" class="text-center py-6 text-gray-400 text-sm">
          No gauges found for "{{ query }}"
        </div>

        <ul v-else class="divide-y divide-gray-100 dark:divide-gray-800 max-h-72 overflow-y-auto">
          <li
            v-for="g in results"
            :key="g.id"
            class="flex items-center justify-between py-2.5 px-1 hover:bg-gray-50 dark:hover:bg-gray-900 rounded"
          >
            <div class="min-w-0 flex-1">
              <!-- Gauge name — always the primary title -->
              <p class="text-sm font-medium truncate">{{ g.name ?? g.externalId }}</p>
              <!-- Reach association + relationship context -->
              <p v-if="g.reachName" class="text-xs text-gray-500 dark:text-gray-400 truncate">
                {{ g.reachName }}
                <span v-if="g.reachRelationship && g.reachRelationship !== 'primary'" class="text-gray-400 dark:text-gray-500">
                  · {{ relationshipLabel(g.reachRelationship) }}
                </span>
              </p>
              <!-- Source ID + location -->
              <p class="text-xs text-gray-400 truncate">
                {{ g.source.toUpperCase() }} · {{ g.externalId }}<template v-if="g.stateAbbr">, {{ g.stateAbbr }}</template>
              </p>
            </div>
            <div class="flex items-center gap-2 ml-2 flex-shrink-0">
              <UBadge :color="tierColor(g.pollTier)" variant="subtle" size="xs">
                {{ g.pollTier }}
              </UBadge>
              <UButton size="xs" color="primary" variant="soft" icon="i-heroicons-plus" @click="select(g)">
                Add
              </UButton>
            </div>
          </li>
        </ul>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ (e: 'add', gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>): void }>()

const query = ref('')
const loading = ref(false)
const results = ref<Omit<WatchedGauge, 'watchState' | 'activeSince'>[]>([])

const { apiBase } = useRuntimeConfig().public

let debounceTimer: ReturnType<typeof setTimeout>

function onInput() {
  clearTimeout(debounceTimer)
  if (query.value.length < 2) {
    results.value = []
    return
  }
  debounceTimer = setTimeout(search, 300)
}

async function search() {
  loading.value = true
  try {
    const url = `${apiBase}/api/v1/gauges/search?q=${encodeURIComponent(query.value)}&limit=20`
    const res = await fetch(url)
    if (!res.ok) return
    const data = await res.json()
    // API returns GeoJSON FeatureCollection — map to WatchedGauge shape
    results.value = (data.features ?? []).map((f: any) => ({
      id:           f.properties.id,
      externalId:   f.properties.external_id,
      source:       f.properties.source,
      name:         f.properties.name ?? null,
      featured:     f.properties.featured ?? false,
      reachId:           f.properties.reach_id ?? null,
      reachName:         f.properties.reach_name ?? null,
      reachSlug:         f.properties.reach_slug ?? null,
      reachRelationship: f.properties.reach_relationship ?? null,
      pollTier:     f.properties.poll_tier,
      watershedName: f.properties.watershed_name ?? null,
      basinName:    f.properties.basin_name ?? null,
      stateAbbr:    f.properties.state_abbr ?? null,
      currentCfs:    f.properties.current_cfs ?? null,
      flowStatus:    f.properties.flow_status ?? 'unknown',
      flowBandLabel: f.properties.flow_band_label ?? null,
      lastReadingAt: f.properties.last_reading_at ?? null,
    }))
  } catch {
    results.value = []
  } finally {
    loading.value = false
  }
}

function select(gauge: Omit<WatchedGauge, 'watchState' | 'activeSince'>) {
  emit('add', gauge)
  open.value = false
  query.value = ''
  results.value = []
}

function tierColor(tier: WatchedGauge['pollTier']) {
  return { trusted: 'success', demand: 'info', cold: 'neutral' }[tier] as any
}

function relationshipLabel(rel: string | null): string {
  switch (rel) {
    case 'upstream_indicator':   return '↑ upstream'
    case 'downstream_indicator': return '↓ downstream'
    case 'tributary':            return '⤷ tributary'
    default:                     return ''
  }
}
</script>
