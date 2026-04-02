<template>
  <div class="min-h-screen flex flex-col bg-white dark:bg-gray-950">

    <!-- Demo banner -->
    <div v-if="showDemoBanner" class="bg-amber-50 dark:bg-amber-950 border-b border-amber-200 dark:border-amber-800 px-4 py-2.5 flex items-center justify-between gap-4 text-sm">
      <p class="text-amber-800 dark:text-amber-200 text-center flex-1">
        <span class="font-semibold">Demo only.</span>
        This build is for feature demonstration purposes. River data is AI-seeded and unverified — do not use for trip planning or safety decisions.
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
        <UButton size="sm" color="neutral" variant="ghost" disabled title="Coming soon">Sign in</UButton>
        <UButton size="sm" color="primary" variant="soft" disabled title="Coming soon">Get started</UButton>
      </nav>
    </header>

    <!-- Hero -->
    <main class="flex-1 flex flex-col items-center justify-center px-6 text-center">
      <div class="max-w-2xl w-full">

        <!-- Animated wave illustration -->
        <div class="flex justify-center mb-8">
          <svg
            ref="waveRef"
            class="w-64 h-20"
            viewBox="0 0 256 80"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <!-- Wave 1 — full opacity, drawn first -->
            <path
              d="M0 22 C16 4 32 4 48 22 C64 40 80 40 96 22 C112 4 128 4 144 22 C160 40 176 40 192 22 C208 4 224 4 240 22 L256 22"
              stroke="#3b82f6"
              stroke-width="3"
              stroke-linecap="round"
              fill="none"
            />
            <!-- Wave 2 — 55% opacity, drawn second -->
            <path
              d="M0 40 C16 22 32 22 48 40 C64 58 80 58 96 40 C112 22 128 22 144 40 C160 58 176 58 192 40 C208 22 224 22 240 40 L256 40"
              stroke="#3b82f6"
              stroke-width="2.5"
              stroke-linecap="round"
              fill="none"
              opacity="0.55"
            />
            <!-- Wave 3 — 22% opacity, drawn third -->
            <path
              d="M0 58 C16 40 32 40 48 58 C64 76 80 76 96 58 C112 40 128 40 144 58 C160 76 176 76 192 58 C208 40 224 40 240 58 L256 58"
              stroke="#3b82f6"
              stroke-width="2"
              stroke-linecap="round"
              fill="none"
              opacity="0.22"
            />
          </svg>
        </div>

        <!-- Badge -->
        <div class="inline-flex items-center gap-1.5 text-xs font-medium text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950 rounded-full px-3 py-1 mb-6">
          <span class="relative flex h-1.5 w-1.5">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75" />
            <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-blue-500" />
          </span>
          Live USGS streamflow · Colorado
        </div>

        <!-- Headline -->
        <h1 class="text-5xl sm:text-6xl font-extrabold tracking-tight text-gray-900 dark:text-white leading-tight mb-4">
          Know before<br>you go.
        </h1>

        <!-- Subtitle -->
        <p class="text-lg sm:text-xl text-gray-500 dark:text-gray-400 mb-10 leading-relaxed">
          Live streamflow data. AI-assisted flow intel. Community knowledge.
        </p>

        <!-- CTAs -->
        <div class="flex flex-wrap items-center justify-center gap-3 mb-16">
          <NuxtLink
            to="/explore"
            class="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-semibold text-sm transition-colors shadow-sm"
          >
            Try it out
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
          </NuxtLink>
          <NuxtLink
            to="/dashboard"
            class="inline-flex items-center gap-2 px-6 py-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-900 text-gray-700 dark:text-gray-300 font-semibold text-sm transition-colors"
          >
            My Dashboard
          </NuxtLink>
        </div>

        <!-- AI Reach Search -->
        <div class="mb-16 w-full max-w-xl mx-auto">
          <form @submit.prevent="askQuestion" class="flex gap-2">
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Ask anything — e.g. &quot;What's Browns Canyon like at 800 cfs?&quot;"
              class="flex-1 px-4 py-2.5 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              :disabled="searching"
            />
            <button
              type="submit"
              :disabled="searching || !searchQuery.trim()"
              class="px-4 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-semibold transition-colors shrink-0"
            >
              <span v-if="searching" class="flex items-center gap-1.5">
                <span class="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                Asking…
              </span>
              <span v-else>Ask</span>
            </button>
          </form>

          <!-- Answer card -->
          <div v-if="searchResult" class="mt-4 rounded-xl border border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-4 text-left">
            <div v-if="searchResult.reach_name" class="flex items-center justify-between mb-3">
              <span class="text-xs font-semibold uppercase tracking-wide text-blue-500">{{ searchResult.reach_name }}</span>
              <NuxtLink
                :to="`/reaches/${searchResult.reach_slug}`"
                class="text-xs text-blue-600 dark:text-blue-400 hover:underline font-medium"
              >
                View reach →
              </NuxtLink>
            </div>
            <p class="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{{ searchResult.answer }}</p>
          </div>

          <p v-if="searchError" class="mt-3 text-sm text-red-500">{{ searchError }}</p>
        </div>

        <!-- Feature tiles -->
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-left">
          <div class="rounded-xl border border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-4">
            <div class="text-2xl mb-2">📡</div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-1">Gauge dashboard</h3>
            <p class="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">USGS streamflow updated every 15 minutes. Build a personal dashboard of the runs you care about.</p>
          </div>
          <div class="rounded-xl border border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-4">
            <div class="text-2xl mb-2">🗺️</div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-1">Interactive map</h3>
            <p class="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">Explore every reach in Colorado. Colored by difficulty — green, blue, black — just like the slopes.</p>
          </div>
          <div class="rounded-xl border border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-4">
            <div class="text-2xl mb-2">🏄</div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-1">Whitewater intel</h3>
            <p class="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">Put-in/take-out access, rapid info, and flow bands so you know if your run is on.</p>
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="shrink-0 px-6 py-4 border-t border-gray-100 dark:border-gray-800 flex items-center justify-between text-xs text-gray-400">
      <span>H2OFlows · Open source whitewater tools</span>
      <div class="flex items-center gap-4">
        <span>Data: USGS · OSM</span>
        <a
          href="https://github.com/brettcvz/h2oflow"
          target="_blank"
          rel="noopener noreferrer"
          class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          title="View on GitHub"
        >
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

const waveRef = ref<SVGSVGElement | null>(null)
const { apiBase } = useRuntimeConfig().public

const showDemoBanner = ref(false)
onMounted(() => {
  showDemoBanner.value = localStorage.getItem('demo-banner-dismissed') !== 'true'
})
function dismissBanner() {
  showDemoBanner.value = false
  localStorage.setItem('demo-banner-dismissed', 'true')
}

// AI reach search
const searchQuery = ref('')
const searching = ref(false)
const searchError = ref('')
const searchResult = ref<{ answer: string; reach_slug?: string; reach_name?: string } | null>(null)

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
  } catch (e) {
    searchError.value = 'Something went wrong. Try again.'
  } finally {
    searching.value = false
  }
}

onMounted(async () => {
  const { gsap } = await import('gsap')

  const svg = waveRef.value
  if (!svg) return

  const paths = Array.from(svg.querySelectorAll<SVGPathElement>('path'))

  // Prime each path for draw-on: set dasharray = full length, offset = full length (invisible)
  for (const path of paths) {
    const len = path.getTotalLength()
    path.style.strokeDasharray = String(len)
    path.style.strokeDashoffset = String(len)
  }

  // Draw each wave in left-to-right, staggered
  gsap.to(paths, {
    strokeDashoffset: 0,
    duration: 1.1,
    ease: 'power2.inOut',
    stagger: 0.18,
    delay: 0.1,
  })
})
</script>
