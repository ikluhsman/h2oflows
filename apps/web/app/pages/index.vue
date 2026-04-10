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

    <!-- Header -->
    <header class="shrink-0 flex items-center justify-between px-6 py-4 border-b border-gray-100 dark:border-gray-800">
      <div class="flex items-center gap-2">
        <svg class="w-6 h-6 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M2 12c2-4 4-6 6-6s4 6 6 6 4-6 6-6" stroke-linecap="round"/>
          <path d="M2 18c2-4 4-6 6-6s4 6 6 6 4-6 6-6" stroke-linecap="round" opacity="0.4"/>
        </svg>
        <span class="text-lg font-bold tracking-tight">H2OFlows</span>
      </div>
      <nav class="flex items-center gap-2">
        <NuxtLink
          to="/dashboard"
          class="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-900 text-gray-700 dark:text-gray-300 font-semibold text-sm transition-colors"
        >Dashboard</NuxtLink>
        <NuxtLink
          to="/trips"
          class="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-900 text-gray-700 dark:text-gray-300 font-semibold text-sm transition-colors"
        >My Trips</NuxtLink>
        <ClientOnly>
          <button
            v-if="isAuthenticated"
            class="text-sm text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors px-2"
            @click="handleSignOut"
          >Sign out</button>
          <NuxtLink
            v-else
            to="/login"
            class="inline-flex items-center px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-semibold text-sm transition-colors"
          >Sign in</NuxtLink>
        </ClientOnly>
      </nav>
    </header>

    <!-- Admin bar -->
    <ClientOnly>
      <div v-if="isAdmin" class="bg-gray-900 dark:bg-black border-b border-gray-700 px-4 py-2 flex items-center gap-3">
        <span class="text-xs font-semibold text-gray-400 uppercase tracking-wide">Admin</span>
        <button
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-gray-700 hover:bg-gray-600 text-white text-xs font-medium transition-colors"
          :disabled="importing"
          @click="triggerKmlUpload"
        >
          <span v-if="importing" class="w-3 h-3 border-2 border-gray-400 border-t-white rounded-full animate-spin"/>
          <svg v-else class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
          {{ importing ? 'Importing…' : 'Upload KML' }}
        </button>
        <button
          class="inline-flex items-center justify-center w-5 h-5 rounded-full border border-gray-600 text-gray-400 hover:text-white hover:border-gray-400 text-xs font-bold transition-colors leading-none"
          title="KML format guide"
          @click="showKmlGuide = true"
        >?</button>
        <span v-if="importMsg" class="text-xs" :class="importError ? 'text-red-400' : 'text-green-400'">{{ importMsg }}</span>
        <input ref="kmlInputRef" type="file" accept=".kml,.kmz" class="hidden" @change="onKmlSelected" />
      </div>

      <!-- KML format guide modal -->
      <Teleport to="body">
        <div
          v-if="showKmlGuide"
          class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60"
          @click.self="showKmlGuide = false"
        >
          <div class="w-full max-w-lg bg-gray-900 border border-gray-700 rounded-xl shadow-2xl text-sm text-gray-200 overflow-y-auto max-h-[90vh]">
            <div class="flex items-center justify-between px-5 py-4 border-b border-gray-700">
              <h2 class="font-semibold text-white">KML Format Guide</h2>
              <button class="text-gray-400 hover:text-white transition-colors text-lg leading-none" @click="showKmlGuide = false">&times;</button>
            </div>
            <div class="px-5 py-4 space-y-4">

              <section>
                <h3 class="text-xs font-semibold uppercase tracking-wide text-gray-400 mb-1">Map structure</h3>
                <ul class="space-y-1 text-gray-300">
                  <li><span class="text-white font-medium">Document name</span> — used as the river name (e.g. <code class="text-blue-400">South Platte River</code>)</li>
                  <li><span class="text-white font-medium">Each folder</span> — one reach. Use the plain display name (e.g. <code class="text-blue-400">Buffalo Creek to Foxton</code>)</li>
                </ul>
              </section>

              <section>
                <h3 class="text-xs font-semibold uppercase tracking-wide text-gray-400 mb-1">Pin name prefixes</h3>
                <div class="grid grid-cols-2 gap-x-4 gap-y-0.5 text-gray-300">
                  <span><code class="text-blue-400">Rapid: Name</code></span><span class="text-gray-500">rapid location</span>
                  <span><code class="text-blue-400">Wave: Name</code></span><span class="text-gray-500">surf wave</span>
                  <span><code class="text-blue-400">Put-in: Name</code></span><span class="text-gray-500">put-in access</span>
                  <span><code class="text-blue-400">Take-out: Name</code></span><span class="text-gray-500">take-out access</span>
                  <span><code class="text-blue-400">Parking: Name</code></span><span class="text-gray-500">parking area</span>
                  <span><code class="text-blue-400">Hazard: Name</code></span><span class="text-gray-500">permanent hazard</span>
                  <span><code class="text-blue-400">Campsite: Name</code></span><span class="text-gray-500">campsite</span>
                </div>
              </section>

              <section>
                <h3 class="text-xs font-semibold uppercase tracking-wide text-gray-400 mb-1">Metadata placemarks <span class="text-gray-600 normal-case font-normal">(no pin — add via folder data table)</span></h3>
                <div class="grid grid-cols-2 gap-x-4 gap-y-0.5 text-gray-300">
                  <span><code class="text-blue-400">common_name</code></span><span class="text-gray-500">short name used in slug, e.g. <code class="text-blue-300">Foxton</code></span>
                  <span><code class="text-blue-400">min_class</code></span><span class="text-gray-500">minimum difficulty, e.g. <code class="text-blue-300">3</code></span>
                  <span><code class="text-blue-400">max_class</code></span><span class="text-gray-500">maximum difficulty, e.g. <code class="text-blue-300">4</code></span>
                  <span><code class="text-blue-400">gauge</code></span><span class="text-gray-500">USGS site number, e.g. <code class="text-blue-300">09058000</code></span>
                  <span><code class="text-blue-400">below</code></span><span class="text-gray-500">max CFS for below-recommended, e.g. <code class="text-blue-300">200</code></span>
                  <span><code class="text-blue-400">low</code></span><span class="text-gray-500">min,max CFS e.g. <code class="text-blue-300">200,400</code> (optional)</span>
                  <span><code class="text-blue-400">med</code></span><span class="text-gray-500">min,max CFS, e.g. <code class="text-blue-300">400,800</code></span>
                  <span><code class="text-blue-400">high</code></span><span class="text-gray-500">min,max CFS e.g. <code class="text-blue-300">800,1200</code> (optional)</span>
                  <span><code class="text-blue-400">above</code></span><span class="text-gray-500">min CFS for above-recommended, e.g. <code class="text-blue-300">1200</code></span>
                </div>
                <p class="mt-1.5 text-gray-500 text-xs">Slug: <code class="text-blue-300">river-name-common-name</code> if common_name set, else <code class="text-blue-300">river-name-folder-name</code>. 3-tier (below/med/above) and 5-tier both work.</p>
              </section>

            </div>
          </div>
        </div>
      </Teleport>
    </ClientOnly>

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

        <!-- Badge -->
        <div class="inline-flex items-center gap-1.5 text-xs font-medium text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950 rounded-full px-3 py-1 mb-5">
          <span class="relative flex h-1.5 w-1.5">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"/>
            <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-blue-500"/>
          </span>
          Live USGS Streamflow Data
        </div>

        <!-- Headline -->
        <h1 class="text-4xl sm:text-5xl font-extrabold tracking-tight text-gray-900 dark:text-white leading-tight mb-3 text-center">
          Know before<br>you go.
        </h1>
        <p class="text-base sm:text-lg text-gray-500 dark:text-gray-400 mb-8 text-center leading-relaxed">
          Live streamflow data. AI-assisted flow intel. Community knowledge.
        </p>

        <!-- Ask anything + action buttons -->
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

        <!-- Feature tiles -->
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-left mt-10 w-full">
          <NuxtLink
            to="/dashboard"
            class="rounded-xl border border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-4 hover:border-blue-200 dark:hover:border-blue-800 hover:bg-blue-50 dark:hover:bg-blue-950/30 transition-colors group"
          >
            <div class="text-2xl mb-2">📡</div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-1 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">Gauge dashboard</h3>
            <p class="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">USGS streamflow updated every 15 minutes. Build a personal dashboard of the runs you care about.</p>
          </NuxtLink>
          <NuxtLink
            to="/map"
            class="rounded-xl border border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-4 hover:border-blue-200 dark:hover:border-blue-800 hover:bg-blue-50 dark:hover:bg-blue-950/30 transition-colors group"
          >
            <div class="text-2xl mb-2">🗺️</div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-1 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">Interactive map</h3>
            <p class="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">Explore every reach in Colorado. Colored by difficulty — green, blue, black — just like the slopes.</p>
          </NuxtLink>
          <button
            type="button"
            class="rounded-xl border border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-4 text-left hover:border-blue-200 dark:hover:border-blue-800 hover:bg-blue-50 dark:hover:bg-blue-950/30 transition-colors group"
            @click="focusAsk"
          >
            <div class="text-2xl mb-2">🏄</div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-1 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">Whitewater intel</h3>
            <p class="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">Put-in/take-out access, rapid info, and flow bands so you know if your run is on.</p>
          </button>
        </div>

      </div>
    </main>

    <!-- Footer -->
    <footer class="shrink-0 px-6 py-4 border-t border-gray-100 dark:border-gray-800 flex items-center justify-between text-xs text-gray-400">
      <span>H2OFlows · Open source whitewater tools</span>
      <div class="flex items-center gap-4">
        <span>Data: USGS · OSM</span>
        <a href="https://github.com/brettcvz/h2oflow" target="_blank" rel="noopener noreferrer" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors" title="View on GitHub">
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

const { isAuthenticated, isAdmin, signOut, getToken } = useAuth()
const router = useRouter()
async function handleSignOut() { await signOut(); router.push('/') }

const waveRef = ref<SVGSVGElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)

function focusAsk() {
  searchInputRef.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  searchInputRef.value?.focus({ preventScroll: true })
}
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

type AskResult = { results: { answer: string; reach_slug: string; reach_name: string }[]; answer?: string }

const searchQuery      = ref('')
const searching        = ref(false)
const searchError      = ref('')
const searchResult     = ref<AskResult | null>(null)
const searchController = ref<AbortController | null>(null)

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

// ── Admin KML upload ──────────────────────────────────────────────────────────

const kmlInputRef  = ref<HTMLInputElement | null>(null)
const importing    = ref(false)
const importMsg    = ref('')
const importError  = ref(false)
const showKmlGuide = ref(false)

function triggerKmlUpload() {
  importMsg.value = ''
  importError.value = false
  kmlInputRef.value?.click()
}

async function onKmlSelected(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  ;(event.target as HTMLInputElement).value = ''

  importing.value = true
  importMsg.value = ''
  importError.value = false

  try {
    const token = await getToken()
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(`${apiBase}/api/v1/import/kmz`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    })
    const json = await res.json()
    if (!res.ok) {
      importError.value = true
      importMsg.value = json.error ?? `Error ${res.status}`
    } else {
      const reachCount = Object.keys(json.reaches ?? {}).length
      importMsg.value = `Imported ${reachCount} reach${reachCount !== 1 ? 'es' : ''}`
    }
  } catch (err: any) {
    importError.value = true
    importMsg.value = err?.message ?? 'Upload failed'
  } finally {
    importing.value = false
  }
}
</script>
