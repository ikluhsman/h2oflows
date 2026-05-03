<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-5xl mx-auto px-4 py-6 space-y-6">

      <!-- Header -->
      <div class="flex items-center gap-3">
        <button class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors" @click="router.back()">
          <svg class="w-5 h-5" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M12.79 5.23a.75.75 0 01-.02 1.06L8.832 10l3.938 3.71a.75.75 0 11-1.04 1.08l-4.5-4.25a.75.75 0 010-1.08l4.5-4.25a.75.75 0 011.06.02z" clip-rule="evenodd" />
          </svg>
        </button>
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ displayName ? `${displayName} Basin` : slug }}
          </h1>
          <p class="text-sm text-gray-500 mt-0.5">
            {{ reachSlugs.length }} reach{{ reachSlugs.length === 1 ? '' : 'es' }} from your dashboard
          </p>
        </div>
      </div>

      <!-- Empty state: no dashboard reaches for this basin -->
      <div v-if="reachSlugs.length === 0" class="py-16 text-center text-gray-400 text-sm">
        No dashboard reaches found for this basin. Add reaches from the
        <NuxtLink to="/" class="text-blue-500 hover:underline">dashboard</NuxtLink>.
      </div>

      <template v-else>
        <!-- NLDI unavailable banner -->
        <div
          v-if="nldiError"
          class="flex items-start gap-3 rounded-lg border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-950/40 px-4 py-3 text-sm text-amber-800 dark:text-amber-300"
        >
          <svg class="w-4 h-4 mt-0.5 shrink-0" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
          </svg>
          <span>{{ nldiError }}</span>
          <button class="ml-auto shrink-0 text-amber-600 dark:text-amber-400 hover:text-amber-800 dark:hover:text-amber-200 transition-colors" @click="nldiError = null">
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/></svg>
          </button>
        </div>

        <!-- Basin Map -->
        <section>
          <ClientOnly>
            <BasinMap
              ref="basinMapRef"
              :reaches="mapData"
              :network="networkData"
              @select="selectedSlug = $event"
            />
          </ClientOnly>
        </section>

        <!-- Reach list -->
        <section class="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 divide-y divide-gray-100 dark:divide-gray-800 overflow-hidden">
          <div
            v-for="reach in mapData"
            :key="reach.slug"
            class="flex items-center gap-3 px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/40 transition-colors cursor-pointer"
            @click="basinMapRef?.flyToReach(reach.slug)"
          >
            <span
              class="w-2 h-2 rounded-full shrink-0"
              :style="{ background: classColor(reach.class_max ?? 0) }"
            />
            <div class="flex-1 min-w-0">
              <p class="text-sm font-semibold text-gray-800 dark:text-gray-100 truncate">
                {{ reach.common_name ?? reach.name }}
              </p>
              <p v-if="reach.river_name" class="text-xs text-gray-400 truncate">{{ reach.river_name }}</p>
            </div>
            <span class="text-xs font-mono text-gray-400 shrink-0">{{ classRange(reach.class_min, reach.class_max) || '—' }}</span>
            <span
              v-if="reach.flow_status !== 'unknown'"
              class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-bold shrink-0"
              :class="flowBandBadgeClass(null, reach.flow_status)"
            >{{ flowBandLabel(null, reach.flow_status) }}</span>
          </div>
          <div v-if="mapData.length === 0 && reachSlugs.length > 0" class="px-4 py-6 text-center text-sm text-gray-400">
            Loading reaches…
          </div>
        </section>

        <!-- D3 River Tree -->
        <section class="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-4">
          <div class="flex items-center gap-2 mb-3">
            <svg class="w-4 h-4 text-gray-400" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="8" cy="3" r="1.5"/><circle cx="4" cy="13" r="1.5"/><circle cx="12" cy="13" r="1.5"/>
              <line x1="8" y1="4.5" x2="8" y2="7"/><path d="M8 7 Q4 9 4 11.5"/><path d="M8 7 Q12 9 12 11.5"/>
            </svg>
            <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide">River Tree</h2>
          </div>
          <ClientOnly>
            <BasinTree
              :reaches="mapData"
              :selected-slug="selectedSlug"
              @select="onTreeSelect"
            />
          </ClientOnly>
        </section>
      </template>

    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from '#app'
import { useWatchlistStore } from '~/stores/watchlist'
import { cleanBasinName, slugifyBasin } from '~/utils/basin'
import { classColor, classRange } from '~/utils/classRating'
import { flowBandBadgeClass, flowBandLabel } from '~/utils/flowBand'
import type { BasinReach, BasinNetwork } from '~/components/basin/BasinMap.vue'

definePageMeta({ ssr: false })

const route  = useRoute()
const router = useRouter()
const slug   = route.params.slug as string
const store  = useWatchlistStore()
const { apiBase } = useRuntimeConfig().public

const mapData      = ref<BasinReach[]>([])
const networkData  = ref<BasinNetwork | null>(null)
const nldiError    = ref<string | null>(null)
const selectedSlug = ref<string | null>(null)
const basinMapRef  = ref<{ flyToReach: (s: string) => void } | null>(null)

function onTreeSelect(slug: string) {
  selectedSlug.value = slug
  basinMapRef.value?.flyToReach(slug)
}

function gaugeBasinSlug(g: ReturnType<typeof store.gauges>[number]): string | null {
  const name = cleanBasinName(g.contextReachBasinGroup)
    ?? cleanBasinName(g.watershedName)
    ?? cleanBasinName(g.basinName)
    ?? cleanBasinName(g.contextReachRiverName)
    ?? cleanBasinName(g.riverName)
    ?? 'Other'
  return slugifyBasin(name)
}

const basinGauges = computed(() =>
  store.gauges.filter(g => gaugeBasinSlug(g) === slug)
)

const displayName = computed<string | null>(() => {
  const g = basinGauges.value[0]
  if (!g) return null
  return cleanBasinName(g.contextReachBasinGroup)
    ?? cleanBasinName(g.watershedName)
    ?? cleanBasinName(g.basinName)
    ?? cleanBasinName(g.contextReachRiverName)
    ?? cleanBasinName(g.riverName)
    ?? null
})

const reachSlugs = computed<string[]>(() => {
  const seen = new Set<string>()
  const out: string[] = []
  for (const g of basinGauges.value) {
    if (g.contextReachSlug && !seen.has(g.contextReachSlug)) {
      seen.add(g.contextReachSlug)
      out.push(g.contextReachSlug)
    }
  }
  return out
})

async function fetchMapData(params: URLSearchParams) {
  const res = await fetch(`${apiBase}/api/v1/reaches/basin/${slug}/map?${params}`)
  if (!res.ok) return
  const body = await res.json()
  mapData.value = body.reaches ?? []
}

async function fetchNetworkData(params: URLSearchParams) {
  const res = await fetch(`${apiBase}/api/v1/reaches/basin/${slug}/network?${params}`)
  if (!res.ok) return
  const body = await res.json()
  networkData.value = { tributaries: body.tributaries, mainstem: body.mainstem }
  if (!body.nldi_available && body.nldi_error) {
    nldiError.value = body.nldi_error
  }
}

async function fetchAll() {
  if (reachSlugs.value.length === 0) return
  const params = new URLSearchParams({ slugs: reachSlugs.value.join(',') })
  try {
    await Promise.all([fetchMapData(params), fetchNetworkData(params)])
  } catch (e) {
    console.warn('[basin page] fetch:', e)
  }
}

// Watch the raw gauges array. When persistedstate (or @pinia/nuxt payload
// restore) patches the store, this fires and triggers the fetch.
// immediate:true handles the case where gauges are already in memory (SPA nav).
const { gauges } = storeToRefs(store)
watch(gauges, () => {
  if (reachSlugs.value.length > 0 && mapData.value.length === 0) fetchAll()
}, { immediate: true })
</script>
