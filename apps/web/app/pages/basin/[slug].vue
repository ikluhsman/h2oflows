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
        <!-- Basin Map -->
        <section>
          <ClientOnly>
            <BasinMap
              ref="basinMapRef"
              :reaches="mapData"
              @select="selectedSlug = $event"
            />
          </ClientOnly>
        </section>

        <!-- D3 River Tree placeholder — wired in PR 6 -->
        <section class="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-6">
          <div class="flex items-center gap-2 mb-1">
            <svg class="w-4 h-4 text-gray-400" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="8" cy="3" r="1.5"/><circle cx="4" cy="13" r="1.5"/><circle cx="12" cy="13" r="1.5"/>
              <line x1="8" y1="4.5" x2="8" y2="7"/><path d="M8 7 Q4 9 4 11.5"/><path d="M8 7 Q12 9 12 11.5"/>
            </svg>
            <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide">River Tree</h2>
          </div>
          <p class="text-sm text-gray-400 italic">Coming soon — dendritic reach tree.</p>
        </section>
      </template>

    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from '#app'
import { useWatchlistStore } from '~/stores/watchlist'
import { cleanBasinName, slugifyBasin } from '~/utils/basin'
import type { BasinReach } from '~/components/basin/BasinMap.vue'

definePageMeta({ ssr: false })

const route  = useRoute()
const router = useRouter()
const slug   = route.params.slug as string
const store  = useWatchlistStore()
const { apiBase } = useRuntimeConfig().public

const mapData      = ref<BasinReach[]>([])
const loading      = ref(false)
const selectedSlug = ref<string | null>(null)
const basinMapRef  = ref<{ flyToReach: (s: string) => void } | null>(null)

// Derive basin display name and reach slugs from the watchlist store.
// Reproduces the same cleanBasinName chain used in dashboard.vue.
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

async function fetchMapData() {
  if (reachSlugs.value.length === 0) return
  loading.value = true
  try {
    const params = new URLSearchParams({ slugs: reachSlugs.value.join(',') })
    const res = await fetch(`${apiBase}/api/v1/reaches/basin/${slug}/map?${params}`)
    if (!res.ok) return
    const body = await res.json()
    mapData.value = body.reaches ?? []
  } catch (e) {
    console.warn('[basin page] fetchMapData:', e)
  } finally {
    loading.value = false
  }
}

// Watch reachSlugs rather than onMounted — store may not be hydrated yet at mount.
// immediate: true fires synchronously if slugs are already available.
watch(reachSlugs, (slugs) => {
  if (slugs.length > 0 && mapData.value.length === 0) fetchMapData()
}, { immediate: true })
</script>
