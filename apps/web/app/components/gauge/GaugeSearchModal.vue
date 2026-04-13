<template>
  <UModal v-model:open="open" title="Add a reach or gauge" :ui="{ width: 'max-w-lg' }">
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

        <ul v-else class="divide-y divide-gray-100 dark:divide-gray-800 max-h-[60vh] overflow-y-auto">
          <template v-for="g in results" :key="g.id">
            <!-- Gauge has reaches: one flat row per reach -->
            <template v-if="g.reachSlugs.length">
              <li
                v-for="(slug, i) in g.reachSlugs"
                :key="slug"
                class="flex items-center justify-between gap-2 py-2.5 px-1 hover:bg-gray-50 dark:hover:bg-gray-900 rounded"
              >
                <div class="min-w-0 flex-1">
                  <p class="text-sm font-medium truncate">{{ g.reachCommonNames[i] ?? g.reachNames[i] ?? slug }}</p>
                  <p class="text-xs text-gray-400 truncate">
                    {{ g.source.toUpperCase() }} · {{ g.externalId }}<template v-if="g.stateAbbr">, {{ g.stateAbbr }}</template>
                    <template v-if="g.reachRelationship && g.reachRelationship !== 'primary' && i === 0">
                      · {{ relationshipLabel(g.reachRelationship) }}
                    </template>
                  </p>
                </div>
                <UButton size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                  @click="selectWithContext(g, slug, g.reachCommonNames[i] ?? g.reachNames[i] ?? null)"
                >Add</UButton>
              </li>
            </template>

            <!-- Gauge has no reaches: standalone row -->
            <li
              v-else
              class="flex items-center justify-between gap-2 py-2.5 px-1 hover:bg-gray-50 dark:hover:bg-gray-900 rounded"
            >
              <div class="min-w-0 flex-1">
                <p class="text-sm font-medium truncate">{{ g.name ?? g.externalId }}</p>
                <p class="text-xs text-gray-400 truncate">
                  {{ g.source.toUpperCase() }} · {{ g.externalId }}<template v-if="g.stateAbbr">, {{ g.stateAbbr }}</template>
                </p>
              </div>
              <UButton size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                @click="selectWithContext(g, null, null)"
              >Add</UButton>
            </li>
          </template>
        </ul>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end">
        <UButton variant="ghost" color="neutral" size="sm" @click="open = false">Cancel</UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { featureToWatchedGauge } from '~/composables/useWatchlistSync'

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
  // Build the watchlist item with the chosen reach context.
  // For a specific reach, look up full name and river name from the gauge's reach data.
  const idx = reachSlug ? gauge.reachSlugs.indexOf(reachSlug) : -1
  const enriched: Omit<WatchedGauge, 'watchState' | 'activeSince'> = {
    ...gauge,
    contextReachSlug:       reachSlug,
    contextReachCommonName: reachCommonName,
    contextReachFullName:   null,  // populated on next batch refresh from API
    contextReachRiverName:  idx >= 0 ? (gauge.riverName ?? null) : null,
  }
  emit('add', enriched)
  open.value = false
  query.value = ''
  results.value = []
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
