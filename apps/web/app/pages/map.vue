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

    <!-- Map + Sidebar -->
    <div class="flex-1 overflow-hidden flex">

      <!-- Map -->
      <div class="flex-1 min-w-0 relative">
        <ClientOnly>
          <ReachesMap
            :hovered-slug="hoveredSlug"
            @reaches-updated="onReachesUpdated"
            @bounds-updated="onBoundsUpdated"
            @zoom-updated="onZoomUpdated"
            @hover-changed="onMapHover"
            @reach-click="onReachClick"
          />
        </ClientOnly>
      </div>

      <!-- Reach sidebar -->
      <aside class="w-72 shrink-0 border-l border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 flex flex-col overflow-hidden">

        <!-- Zoom-out prompt -->
        <div v-if="mapZoom < SIDEBAR_ZOOM" class="flex-1 flex flex-col items-center justify-center gap-3 p-6 text-center">
          <svg class="w-8 h-8 text-gray-300 dark:text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
            <path d="M11 8v6M8 11h6" stroke-linecap="round"/>
          </svg>
          <p class="text-sm text-gray-400 leading-relaxed">Zoom in to display<br>river details</p>
        </div>

        <!-- Reach list -->
        <template v-else>
          <div class="px-3 py-2.5 border-b border-gray-100 dark:border-gray-800 flex items-center justify-between">
            <span class="text-xs font-semibold text-gray-500 uppercase tracking-wide">
              {{ mapReaches.length }} reach{{ mapReaches.length === 1 ? '' : 'es' }} in view
            </span>
          </div>

          <div class="flex-1 overflow-y-auto">
            <div
              v-for="r in mapReaches"
              :key="r.slug"
              :ref="(el) => setReachRef(r.slug, el as HTMLElement | null)"
              class="px-3 py-2.5 border-b border-gray-50 dark:border-gray-900 cursor-pointer transition-colors"
              :class="hoveredSlug === r.slug
                ? 'bg-blue-50 dark:bg-blue-950/40'
                : 'hover:bg-gray-50 dark:hover:bg-gray-900/60'"
              @mouseenter="hoveredSlug = r.slug"
              @mouseleave="hoveredSlug = null"
              @click="navigateTo(`/reaches/${r.slug}`)"
            >
              <div class="flex items-center gap-2 min-w-0">
                <!-- Flow status dot -->
                <span
                  class="w-2 h-2 rounded-full shrink-0"
                  :style="{ background: flowStatusColor(r.flow_status) }"
                />
                <span class="text-sm font-medium truncate text-gray-800 dark:text-gray-200">{{ r.name }}</span>
              </div>
              <div class="flex items-center justify-between mt-0.5 pl-4">
                <span class="text-xs text-gray-400">{{ classLabel(r.class_max) }}</span>
                <span
                  v-if="r.current_cfs != null"
                  class="text-xs font-medium tabular-nums"
                  :style="{ color: flowStatusColor(r.flow_status) }"
                >{{ r.current_cfs.toLocaleString() }} cfs</span>
                <span v-else class="text-xs text-gray-300 dark:text-gray-600">no data</span>
              </div>
            </div>

            <!-- Empty inside threshold -->
            <div v-if="mapReaches.length === 0" class="flex items-center justify-center py-12 text-xs text-gray-400">
              No reaches found in this area
            </div>
          </div>
        </template>
      </aside>
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

          <div v-if="searchResult" class="px-4 py-4 space-y-3 max-h-96 overflow-y-auto">
            <div
              v-for="result in (searchResult.results ?? [])"
              :key="result.reach_slug"
              class="rounded-lg border border-gray-100 dark:border-gray-800 p-3 space-y-1"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs font-semibold uppercase tracking-wide text-blue-500">{{ result.reach_name }}</span>
                <NuxtLink
                  :to="`/reaches/${result.reach_slug}`"
                  class="text-xs text-blue-600 dark:text-blue-400 hover:underline font-medium shrink-0"
                  @click="searchOpen = false"
                >View reach →</NuxtLink>
              </div>
              <p class="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{{ result.answer }}</p>
            </div>
            <p v-if="!searchResult.results?.length && searchResult.answer" class="text-sm text-gray-500 leading-relaxed">{{ searchResult.answer }}</p>
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
import type { ReachListItem } from '~/components/map/ReachesMap.vue'

const { apiBase } = useRuntimeConfig().public
const router = useRouter()

const showDemoBanner = ref(false)
onMounted(() => {
  showDemoBanner.value = localStorage.getItem('demo-banner-dismissed') !== 'true'
})
function dismissBanner() {
  showDemoBanner.value = false
  localStorage.setItem('demo-banner-dismissed', 'true')
}

// ── Sidebar ───────────────────────────────────────────────────────────────────

// Zoom level at which the sidebar shows reach details (~state-sized viewport)
const SIDEBAR_ZOOM = 6.5

const mapZoom    = ref(4)
const mapReaches = ref<ReachListItem[]>([])
const hoveredSlug = ref<string | null>(null)

// DOM ref map for scrolling sidebar to hovered reach
const reachRefs = new Map<string, HTMLElement>()
function setReachRef(slug: string, el: HTMLElement | null) {
  if (el) reachRefs.set(slug, el)
  else    reachRefs.delete(slug)
}

function onReachesUpdated(reaches: ReachListItem[]) {
  mapReaches.value = reaches
}
function onBoundsUpdated(_bbox: string) {}
function onZoomUpdated(zoom: number) {
  mapZoom.value = zoom
}

// When the map emits a hover (user moused over a line), update hoveredSlug
// and scroll the sidebar to that row
function onMapHover(slug: string | null) {
  hoveredSlug.value = slug
  if (slug) {
    nextTick(() => {
      reachRefs.get(slug)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    })
  }
}

function onReachClick(slug: string) {
  router.push(`/reaches/${slug}`)
}

// Flow status colors — aligned with GaugeGraph band colors
function flowStatusColor(status: string): string {
  const map: Record<string, string> = {
    runnable: '#22c55e',   // green  — runnable band
    caution:  '#ef4444',   // red    — below_recommended band
    low:      '#ef4444',   // red
    flood:    '#3b82f6',   // blue   — above_recommended band
  }
  return map[status] ?? '#9ca3af'
}

// Difficulty label for sidebar rows
function classLabel(classMax: number | null): string {
  if (classMax == null) return 'Unknown'
  const labels: Record<number, string> = {
    0: 'Class I', 1: 'Class I', 1.5: 'Class I+',
    2: 'Class II', 2.5: 'Class II+',
    3: 'Class III', 3.5: 'Class III+',
    4: 'Class IV', 4.5: 'Class IV+',
    5: 'Class V', 5.5: 'Class V+', 6: 'Class VI',
  }
  return labels[classMax] ?? `Class ${classMax}`
}

// ── AI search ─────────────────────────────────────────────────────────────────

const searchOpen  = ref(false)
const searchInputRef = ref<HTMLInputElement>()
const searchQuery = ref('')
const searching   = ref(false)
const searchError = ref('')
const searchResult = ref<{ results?: { answer: string; reach_slug: string; reach_name: string }[]; answer?: string } | null>(null)

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
