<template>
  <div class="min-h-screen flex flex-col bg-white dark:bg-gray-950">

    <!-- Demo banner -->
    <div v-if="showDemoBanner" class="bg-amber-50 dark:bg-amber-950 border-b border-amber-200 dark:border-amber-800 px-4 py-2.5 flex items-center justify-between gap-4 text-sm">
      <p class="text-amber-800 dark:text-amber-200 text-center flex-1">
        <span class="font-semibold">Demo only.</span>
        River data is AI-seeded and unverified — do not use for trip planning or safety decisions.
      </p>
      <button @click="dismissBanner" class="shrink-0 text-amber-600 dark:text-amber-400 hover:text-amber-900 dark:hover:text-amber-100 font-medium transition-colors">Dismiss</button>
    </div>

    <AppHeader />

    <!-- Hero -->
    <main class="flex-1 flex flex-col items-center px-4 sm:px-6 pt-10 pb-12">
      <div class="w-full max-w-4xl flex flex-col items-center">

        <!-- Wave animation -->
        <div class="flex justify-center mb-6">
          <svg ref="waveRef" class="w-48 h-14" viewBox="0 0 250 80" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M0 22 C16 4 32 4 48 22 C64 40 80 40 96 22 C112 4 128 4 144 22 C160 40 176 40 192 22 C208 4 224 4 240 22 C256 40 272 40 272 22" stroke="#3b82f6" stroke-width="3" stroke-linecap="round" fill="none"/>
            <path d="M0 40 C16 22 32 22 48 40 C64 58 80 58 96 40 C112 22 128 22 144 40 C160 58 176 58 192 40 C208 22 224 22 240 40 C256 58 272 58 272 40" stroke="#3b82f6" stroke-width="2.5" stroke-linecap="round" fill="none" opacity="0.55"/>
            <path d="M0 58 C16 40 32 40 48 58 C64 76 80 76 96 58 C112 40 128 40 144 58 C160 76 176 76 192 58 C208 40 224 40 240 58 C256 76 272 76 272 58" stroke="#3b82f6" stroke-width="2" stroke-linecap="round" fill="none" opacity="0.22"/>
          </svg>
        </div>

        <!-- Wordmark -->
        <p class="text-2xl font-extrabold tracking-tight text-blue-600 dark:text-blue-400 mb-4">h2oflows</p>

        <!-- Badge -->
        <div class="inline-flex items-center gap-1.5 text-xs font-medium text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950 rounded-full px-3 py-1 mb-5">
          <span class="relative flex h-1.5 w-1.5">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"/>
            <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-blue-500"/>
          </span>
          Live Streamflow · Community-driven
        </div>

        <!-- Headline -->
        <h1 class="text-4xl sm:text-5xl font-extrabold tracking-tight text-gray-900 dark:text-white leading-tight mb-3 text-center">
          Know before<br>you go.
        </h1>

        <!-- Feature pills — plain text -->
        <div class="flex items-center gap-2 sm:gap-3 mb-6 text-sm font-medium text-gray-500 dark:text-gray-400">
          <span>Real-time Gauges</span>
          <span class="text-gray-300 dark:text-gray-700">·</span>
          <span>Smart Sharing</span>
          <span class="text-gray-300 dark:text-gray-700">·</span>
          <span>AI Intel</span>
        </div>

        <!-- Primary nav buttons -->
        <div class="flex items-center gap-3 mb-8">
          <NuxtLink
            to="/dashboard"
            class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold transition-colors shadow-sm"
          >
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
            Dashboard
          </NuxtLink>
          <NuxtLink
            to="/map"
            class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-200 text-sm font-semibold transition-colors shadow-sm"
          >
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="1 6 1 22 8 18 16 22 23 18 23 2 16 6 8 2 1 6"/><line x1="8" y1="2" x2="8" y2="18"/><line x1="16" y1="6" x2="16" y2="22"/></svg>
            Map
          </NuxtLink>
        </div>

        <!-- Ask anything -->
        <div class="w-full max-w-xl mb-4">
          <form @submit.prevent="askQuestion" class="flex gap-2 mb-3">
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              placeholder='Ask anything — e.g. "Browns Canyon at 800 cfs?"'
              class="flex-1 px-4 py-2.5 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              :disabled="searching"
            />
            <button
              v-if="!searching"
              type="submit"
              :disabled="!searchQuery.trim()"
              class="px-4 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-semibold transition-colors shrink-0"
            >Ask</button>
            <button
              v-else
              type="button"
              class="px-4 py-2.5 rounded-lg bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200 text-sm font-semibold transition-colors shrink-0 flex items-center gap-1.5"
              @click="cancelSearch"
            >
              <span class="w-3.5 h-3.5 border-2 border-gray-400 border-t-gray-700 dark:border-t-gray-200 rounded-full animate-spin"/>
              Stop
            </button>
          </form>

          <p v-if="searching" class="text-sm text-gray-400 dark:text-gray-500 animate-pulse text-center -mt-1 mb-2">{{ loadingVerb }}…</p>

          <!-- Answer cards — one per matched reach -->
          <div v-if="searchResult && searchResult.results.length > 0" class="mb-3 flex flex-col gap-2">
            <div
              v-for="result in searchResult.results"
              :key="result.reach_slug"
              class="rounded-xl border border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-4 text-left"
            >
              <div class="flex items-center justify-between mb-2">
                <span class="text-xs font-semibold uppercase tracking-wide text-blue-500">{{ result.reach_name }}</span>
                <NuxtLink :to="`/reaches/${result.reach_slug}`" class="text-xs text-blue-600 dark:text-blue-400 hover:underline font-medium shrink-0">View reach →</NuxtLink>
              </div>
              <p class="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{{ result.answer }}</p>
            </div>
          </div>
          <p v-else-if="searchResult && searchResult.answer" class="mb-3 text-sm text-gray-500 dark:text-gray-400">{{ searchResult.answer }}</p>
          <p v-if="searchError" class="mb-3 text-sm text-red-500">{{ searchError }}</p>
        </div>

        <!-- Framed reach map -->
        <div class="relative w-full rounded-xl overflow-hidden border border-gray-200 dark:border-gray-700 shadow-sm" style="height: 420px;">
          <ClientOnly>
            <ReachesMap
              :hovered-slug="heroHoveredSlug"
              @reaches-updated="onReachesUpdated"
              @bounds-updated="onBoundsUpdated"
              @hover-changed="slug => heroHoveredSlug = slug"
              @reach-click="slug => navigateTo(`/reaches/${slug}`)"
              @gauge-add="addGaugeById"
            />
          </ClientOnly>
          <NuxtLink
            to="/map"
            class="absolute top-3 right-12 z-10 inline-flex items-center justify-center w-9 h-9 rounded-md bg-white/95 dark:bg-gray-900/95 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 hover:bg-white dark:hover:bg-gray-900 hover:text-blue-600 dark:hover:text-blue-400 shadow-sm transition-colors"
            title="Open full screen map"
            aria-label="Open full screen map"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/></svg>
          </NuxtLink>
        </div>

        <!-- App store buttons -->
        <div class="flex flex-col items-center gap-3 mt-10">
          <div class="flex items-center gap-4">
            <!-- Apple App Store -->
            <a href="#" class="inline-flex items-center gap-2.5 px-5 py-2.5 rounded-xl bg-black text-white hover:bg-gray-800 transition-colors">
              <svg class="w-6 h-6" viewBox="0 0 24 24" fill="currentColor">
                <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 21.99 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.1 21.99C7.79 22.03 6.8 20.68 5.96 19.47C4.25 16.97 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.89C10.1 6.87 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.1 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5ZM13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z"/>
              </svg>
              <div class="text-left">
                <div class="text-[10px] leading-none opacity-80">Download on the</div>
                <div class="text-sm font-semibold leading-tight">App Store</div>
              </div>
            </a>
            <!-- Google Play Store -->
            <a href="#" class="inline-flex items-center gap-2.5 px-5 py-2.5 rounded-xl bg-black text-white hover:bg-gray-800 transition-colors">
              <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
                <path d="M3.609 1.814L13.792 12 3.61 22.186a.996.996 0 01-.61-.92V2.734a1 1 0 01.609-.92zm10.89 10.893l2.302 2.302-10.937 6.333 8.635-8.635zm3.199-3.199l2.608 1.51a.999.999 0 010 1.764l-2.608 1.509-2.544-2.544 2.544-2.239zM5.864 2.658L16.8 8.99l-2.302 2.302-8.634-8.634z"/>
              </svg>
              <div class="text-left">
                <div class="text-[10px] leading-none opacity-80">Get it on</div>
                <div class="text-sm font-semibold leading-tight">Google Play</div>
              </div>
            </a>
          </div>
          <p class="text-xs text-gray-400 dark:text-gray-500">Mobile apps coming soon</p>
        </div>

      </div>
    </main>

    <!-- Footer -->
    <footer class="shrink-0 px-6 py-4 border-t border-gray-100 dark:border-gray-800 flex items-center justify-between text-xs text-gray-400">
      <span>H2OFlows</span>
      <div class="flex items-center gap-4">
        <span>Data: USGS · OSM</span>
        <a href="https://github.com/ikluhsman/h2oflows/issues" target="_blank" rel="noopener noreferrer" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">Support</a>
        <a href="https://github.com/ikluhsman/h2oflows" target="_blank" rel="noopener noreferrer" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors" title="View on GitHub">
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2C6.477 2 2 6.484 2 12.021c0 4.428 2.865 8.185 6.839 9.504.5.092.682-.217.682-.483 0-.237-.009-.868-.013-1.703-2.782.605-3.369-1.342-3.369-1.342-.454-1.154-1.11-1.462-1.11-1.462-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844a9.59 9.59 0 012.504.337c1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.021C22 6.484 17.522 2 12 2z"/>
          </svg>
        </a>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useWatchlistStore, type WatchedGauge } from '~/stores/watchlist'

const waveRef = ref<SVGSVGElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)

function focusAsk() {
  searchInputRef.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  searchInputRef.value?.focus({ preventScroll: true })
}
const { apiBase } = useRuntimeConfig().public
const store = useWatchlistStore()
const { addAndSync } = useWatchlistSync()
const showDemoBanner = ref(false)
onMounted(() => {
  showDemoBanner.value = localStorage.getItem('demo-banner-dismissed') !== 'true'
})
function dismissBanner() {
  showDemoBanner.value = false
  localStorage.setItem('demo-banner-dismissed', 'true')
}

// ── Wave animation ────────────────────────────────────────────────────────────

onMounted(async () => {
  const { gsap } = await import('gsap')
  const svg = waveRef.value
  if (!svg) return
  const paths = Array.from(svg.querySelectorAll<SVGPathElement>('path'))
  for (const path of paths) {
    const len = path.getTotalLength()
    path.style.strokeDasharray = String(len)
    path.style.strokeDashoffset = String(len)
  }
  gsap.to(paths, { strokeDashoffset: 0, duration: 1.1, ease: 'power2.inOut', stagger: 0.18, delay: 0.1 })
})

// ── Map callbacks ─────────────────────────────────────────────────────────────

const heroHoveredSlug = ref<string | null>(null)

function onReachesUpdated(_reaches: { slug: string; name: string; class_max: number | null }[]) {}
function onBoundsUpdated(_bbox: string) {}

async function addGaugeById(gaugeId: string) {
  try {
    const res = await fetch(`${apiBase}/api/v1/gauges/batch?ids=${gaugeId}`)
    if (!res.ok) return
    const data = await res.json()
    const f = data.features?.[0]
    if (!f) return
    const p = f.properties
    const coords = f.geometry?.coordinates as [number, number] | undefined
    addAndSync({
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

type AskResult = { results: { answer: string; reach_slug: string; reach_name: string }[]; answer?: string }

const searchQuery      = ref('')
const searching        = ref(false)
const searchError      = ref('')
const searchResult     = ref<AskResult | null>(null)
const searchController = ref<AbortController | null>(null)

const LOADING_VERBS = [
  'Eddying', 'Stoking', 'Paddling', 'Navigating', 'Formulating',
  'Hydrating', 'Pumping', 'Rolling', 'Ferrying', 'Scouting',
  'Rowing', 'Boofing', 'Peeling Out', 'Portaging', 'Bracing', 'Surfing',
]
const loadingVerb = ref(LOADING_VERBS[0])
let verbInterval: ReturnType<typeof setInterval> | null = null

watch(searching, (active) => {
  if (active) {
    loadingVerb.value = LOADING_VERBS[Math.floor(Math.random() * LOADING_VERBS.length)]
    verbInterval = setInterval(() => {
      loadingVerb.value = LOADING_VERBS[Math.floor(Math.random() * LOADING_VERBS.length)]
    }, 2000)
  } else {
    if (verbInterval) { clearInterval(verbInterval); verbInterval = null }
  }
})
onUnmounted(() => { if (verbInterval) clearInterval(verbInterval) })

async function askQuestion() {
  // Cancel any in-flight request before starting a new one
  searchController.value?.abort()

  const q = searchQuery.value.trim()
  if (!q) return

  const controller = new AbortController()
  searchController.value = controller
  searching.value = true
  searchError.value = ''
  searchResult.value = null

  try {
    const res = await fetch(`${apiBase}/api/v1/ask`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question: q }),
      signal: controller.signal,
    })
    if (!res.ok) throw new Error(`${res.status}`)
    searchResult.value = await res.json()
  } catch (err: any) {
    if (err?.name !== 'AbortError') {
      searchError.value = 'Something went wrong. Try again.'
    }
  } finally {
    searching.value = false
    searchController.value = null
  }
}

function cancelSearch() {
  searchController.value?.abort()
  searching.value = false
}

</script>
