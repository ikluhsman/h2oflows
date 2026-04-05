<template>
  <div class="h-screen flex flex-col overflow-hidden bg-white dark:bg-gray-950">

    <!-- Demo banner -->
    <div v-if="showDemoBanner" class="shrink-0 bg-amber-50 dark:bg-amber-950 border-b border-amber-200 dark:border-amber-800 px-4 py-2 flex items-center justify-between gap-4 text-sm">
      <p class="text-amber-800 dark:text-amber-200 text-center flex-1">
        <span class="font-semibold">Demo only.</span>
        River data is AI-seeded and unverified — do not use for trip planning or safety decisions.
      </p>
      <button @click="dismissBanner" class="shrink-0 text-amber-600 dark:text-amber-400 hover:text-amber-900 dark:hover:text-amber-100 font-medium transition-colors">Dismiss</button>
    </div>

    <!-- Header -->
    <header class="shrink-0 flex items-center justify-between px-4 py-3 border-b border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-950 z-10">
      <div class="flex items-center gap-2">
        <NuxtLink to="/" class="flex items-center gap-2 hover:opacity-80 transition-opacity">
          <svg class="w-5 h-5 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M2 12c2-4 4-6 6-6s4 6 6 6 4-6 6-6" stroke-linecap="round"/>
            <path d="M2 18c2-4 4-6 6-6s4 6 6 6 4-6 6-6" stroke-linecap="round" opacity="0.4"/>
          </svg>
          <span class="text-base font-bold tracking-tight">H2OFlows</span>
        </NuxtLink>
        <span class="hidden sm:inline text-xs text-gray-400 ml-1">Colorado · Live streamflow</span>
      </div>
      <nav class="flex items-center gap-2">
        <button
          class="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
          @click="searchOpen = true"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
          Ask anything…
        </button>
        <NuxtLink
          to="/dashboard"
          class="text-xs px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-semibold transition-colors"
        >Dashboard</NuxtLink>
      </nav>
    </header>

    <!-- Map — fills remaining height -->
    <div class="flex-1 overflow-hidden">
      <ClientOnly>
        <ReachesMap
          @reaches-updated="onReachesUpdated"
          @bounds-updated="onBoundsUpdated"
          @gauge-add="addGaugeById"
        />
      </ClientOnly>
    </div>

    <!-- AI search modal -->
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="searchOpen"
        class="fixed inset-0 z-50 flex items-start justify-center pt-[10vh] px-4"
        @click.self="searchOpen = false"
      >
        <div class="w-full max-w-xl bg-white dark:bg-gray-900 rounded-2xl shadow-2xl border border-gray-100 dark:border-gray-800 overflow-hidden">
          <!-- Search input -->
          <form class="flex items-center gap-2 px-4 py-3 border-b border-gray-100 dark:border-gray-800" @submit.prevent="askQuestion">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-gray-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              placeholder='Ask anything — e.g. "Browns Canyon at 800 cfs?"'
              class="flex-1 bg-transparent text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none"
              :disabled="searching"
            />
            <button
              v-if="searchQuery"
              type="button"
              class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              @click="searchQuery = ''; searchResult = null"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18M6 6l12 12"/></svg>
            </button>
            <button
              type="submit"
              :disabled="searching || !searchQuery.trim()"
              class="shrink-0 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-40 text-white text-xs font-semibold transition-colors"
            >
              <span v-if="searching" class="flex items-center gap-1">
                <span class="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin"/>
              </span>
              <span v-else>Ask</span>
            </button>
          </form>

          <!-- Answer -->
          <div v-if="searchResult" class="px-4 py-4">
            <div v-if="searchResult.reach_name" class="flex items-center justify-between mb-2">
              <span class="text-xs font-semibold uppercase tracking-wide text-blue-500">{{ searchResult.reach_name }}</span>
              <NuxtLink
                :to="`/reaches/${searchResult.reach_slug}`"
                class="text-xs text-blue-600 dark:text-blue-400 hover:underline font-medium"
                @click="searchOpen = false"
              >View reach →</NuxtLink>
            </div>
            <p class="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{{ searchResult.answer }}</p>
          </div>
          <p v-else-if="!searching && !searchResult" class="px-4 py-3 text-xs text-gray-400">
            Try: "What's Foxton like at 300 cfs?" or "Best beginner runs near Denver"
          </p>
          <p v-if="searchError" class="px-4 py-3 text-sm text-red-500">{{ searchError }}</p>

          <div class="px-4 py-2.5 border-t border-gray-100 dark:border-gray-800 flex justify-end">
            <button class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" @click="searchOpen = false">Close</button>
          </div>
        </div>
      </div>
    </Transition>

  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, watch, onMounted } from 'vue'
import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'

const { apiBase } = useRuntimeConfig().public
const store = useWatchlistStore()

const showDemoBanner = ref(false)
onMounted(() => {
  showDemoBanner.value = localStorage.getItem('demo-banner-dismissed') !== 'true'
})
function dismissBanner() {
  showDemoBanner.value = false
  localStorage.setItem('demo-banner-dismissed', 'true')
}

// ── Map callbacks ─────────────────────────────────────────────────────────────

function onReachesUpdated(_reaches: { slug: string; name: string; class_max: number | null }[]) {
  // reserved for future sidebar / stats
}
function onBoundsUpdated(_bbox: string) {
  // reserved
}

// ── Add gauge from map popup ──────────────────────────────────────────────────

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
    } satisfies Omit<WatchedGauge, 'watchState' | 'activeSince'>)
  } catch { /* non-fatal */ }
}

// ── AI search ─────────────────────────────────────────────────────────────────

const searchOpen  = ref(false)
const searchInputRef = ref<HTMLInputElement>()
const searchQuery = ref('')
const searching   = ref(false)
const searchError = ref('')
const searchResult = ref<{ answer: string; reach_slug?: string; reach_name?: string } | null>(null)

watch(searchOpen, async (open) => {
  if (open) {
    searchQuery.value = ''
    searchResult.value = null
    searchError.value = ''
    await nextTick()
    searchInputRef.value?.focus()
  }
})

async function askQuestion() {
  const q = searchQuery.value.trim()
  if (!q) return
  searching.value = true
  searchError.value = ''
  searchResult.value = null
  try {
    const res = await fetch(`${apiBase}/api/v1/ask`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question: q }),
    })
    if (!res.ok) throw new Error(`${res.status}`)
    searchResult.value = await res.json()
  } catch {
    searchError.value = 'Something went wrong. Try again.'
  } finally {
    searching.value = false
  }
}
</script>
