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

    <AppHeader class="shrink-0" />

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

  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted } from 'vue'
import type { ReachListItem } from '~/components/map/ReachesMap.vue'

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

</script>
