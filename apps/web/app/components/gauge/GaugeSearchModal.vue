<template>
  <UModal v-model:open="open" title="Add a reach or gauge" :ui="{ width: 'max-w-2xl' }">
    <template #body>
      <div class="space-y-4">
        <UInput
          v-model="query"
          placeholder="Search by river, section, or gauge ID…"
          icon="i-heroicons-magnifying-glass"
          size="lg"
          autofocus
          @input="onInput"
        />

        <div class="flex gap-4">
          <!-- Results list -->
          <div class="flex-1 min-w-0">
            <!-- Skeleton loader -->
            <div v-if="loading" class="space-y-2 py-2">
              <div v-for="i in 4" :key="i" class="flex items-center gap-3 px-2 py-2.5">
                <div class="flex-1 space-y-2">
                  <div class="h-4 w-3/4 rounded bg-gray-100 dark:bg-gray-800 animate-pulse" />
                  <div class="h-3 w-1/2 rounded bg-gray-100 dark:bg-gray-800 animate-pulse" />
                </div>
                <div class="h-7 w-14 rounded bg-gray-100 dark:bg-gray-800 animate-pulse" />
              </div>
            </div>

            <!-- Empty state -->
            <div v-else-if="results.length === 0 && query.length >= 2" class="text-center py-10 text-gray-400 text-sm">
              No gauges found for "{{ query }}"
            </div>

            <!-- Idle state -->
            <div v-else-if="results.length === 0" class="text-center py-10 text-gray-400 text-sm">
              Type to search rivers, sections, or gauge IDs
            </div>

            <!-- Results -->
            <ul v-else class="divide-y divide-gray-100 dark:divide-gray-800 max-h-[60vh] overflow-y-auto">
              <template v-for="g in results" :key="g.id">
                <!-- Gauge has reaches: one row per reach -->
                <template v-if="g.reachSlugs.length">
                  <li
                    v-for="(slug, i) in g.reachSlugs"
                    :key="slug"
                    class="flex items-center justify-between gap-3 py-2.5 px-2 hover:bg-blue-50 dark:hover:bg-blue-950/30 rounded-lg transition-colors cursor-pointer"
                    @mouseenter="hoverGauge = g"
                    @mouseleave="hoverGauge = null"
                    @click="selectWithContext(g, slug, g.reachCommonNames[i] ?? g.reachNames[i] ?? null)"
                  >
                    <div class="min-w-0 flex-1">
                      <p class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{{ g.reachCommonNames[i] ?? g.reachNames[i] ?? slug }}</p>
                      <div class="flex items-center gap-1.5 mt-0.5">
                        <span v-if="g.riverName" class="text-xs text-gray-500 dark:text-gray-400 truncate">{{ g.riverName }}</span>
                        <span v-if="g.riverName" class="text-gray-300 dark:text-gray-600 text-xs">·</span>
                        <span class="text-xs text-gray-400 truncate">
                          {{ g.source.toUpperCase() }} {{ g.externalId }}<template v-if="g.stateAbbr">, {{ g.stateAbbr }}</template>
                        </span>
                      </div>
                    </div>
                    <div class="flex items-center gap-2 shrink-0">
                      <span v-if="g.currentCfs != null" class="text-sm font-semibold tabular-nums text-gray-700 dark:text-gray-300">
                        {{ g.currentCfs.toLocaleString() }}
                        <span class="text-xs font-normal text-gray-400">cfs</span>
                      </span>
                      <UButton size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                        @click.stop="selectWithContext(g, slug, g.reachCommonNames[i] ?? g.reachNames[i] ?? null)"
                      >Add</UButton>
                    </div>
                  </li>
                </template>

                <!-- Standalone gauge (no reaches) -->
                <li
                  v-else
                  class="flex items-center justify-between gap-3 py-2.5 px-2 hover:bg-blue-50 dark:hover:bg-blue-950/30 rounded-lg transition-colors cursor-pointer"
                  @mouseenter="hoverGauge = g"
                  @mouseleave="hoverGauge = null"
                  @click="selectWithContext(g, null, null)"
                >
                  <div class="min-w-0 flex-1">
                    <p class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{{ g.name ?? g.externalId }}</p>
                    <div class="flex items-center gap-1.5 mt-0.5">
                      <span v-if="g.riverName" class="text-xs text-gray-500 dark:text-gray-400 truncate">{{ g.riverName }}</span>
                      <span v-if="g.riverName" class="text-gray-300 dark:text-gray-600 text-xs">·</span>
                      <span class="text-xs text-gray-400 truncate">
                        {{ g.source.toUpperCase() }} {{ g.externalId }}<template v-if="g.stateAbbr">, {{ g.stateAbbr }}</template>
                      </span>
                    </div>
                  </div>
                  <div class="flex items-center gap-2 shrink-0">
                    <span v-if="g.currentCfs != null" class="text-sm font-semibold tabular-nums text-gray-700 dark:text-gray-300">
                      {{ g.currentCfs.toLocaleString() }}
                      <span class="text-xs font-normal text-gray-400">cfs</span>
                    </span>
                    <UButton size="xs" color="primary" variant="soft" icon="i-heroicons-plus"
                      @click.stop="selectWithContext(g, null, null)"
                    >Add</UButton>
                  </div>
                </li>
              </template>
            </ul>
          </div>

          <!-- Mini-map preview — only visible when results exist, hidden on small screens -->
          <div
            v-if="results.length > 0"
            class="hidden sm:block w-48 shrink-0 rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700 self-start sticky top-0"
          >
            <ClientOnly>
              <GaugeSearchMiniMap
                :gauges="results"
                :highlight-id="hoverGauge?.id ?? null"
              />
            </ClientOnly>
          </div>
        </div>
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
const hoverGauge = ref<Omit<WatchedGauge, 'watchState' | 'activeSince'> | null>(null)

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
  const idx = reachSlug ? gauge.reachSlugs.indexOf(reachSlug) : -1
  const enriched: Omit<WatchedGauge, 'watchState' | 'activeSince'> = {
    ...gauge,
    contextReachSlug:       reachSlug,
    contextReachCommonName: reachCommonName,
    contextReachFullName:   null,
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
