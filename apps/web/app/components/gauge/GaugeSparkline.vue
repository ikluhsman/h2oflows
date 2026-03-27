<template>
  <div class="w-full" @click.stop>
    <!-- Window toggle -->
    <div class="flex justify-end mb-0.5">
      <div class="flex text-xs rounded overflow-hidden border border-gray-200 dark:border-gray-700">
        <button
          class="px-1.5 py-0.5 transition-colors"
          :class="hours === 12 ? 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200' : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'"
          @click="hours = 12"
        >12h</button>
        <button
          class="px-1.5 py-0.5 transition-colors"
          :class="hours === 24 ? 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200' : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'"
          @click="hours = 24"
        >24h</button>
      </div>
    </div>

    <!-- Chart area -->
    <div class="relative w-full h-10">
      <div v-if="loading" class="w-full h-full rounded animate-pulse bg-gray-100 dark:bg-gray-800" />

      <template v-else-if="points.length >= 2">
        <svg viewBox="0 0 100 40" preserveAspectRatio="none" class="w-full h-full overflow-visible">
          <path :d="areaPath" :fill="strokeColor" fill-opacity="0.12" />
          <path :d="linePath" :stroke="strokeColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" fill="none" />
        </svg>
        <!-- Labels outside the scaled SVG so they render at normal proportions -->
        <span class="absolute top-0 left-0 text-[9px] leading-none text-gray-400 font-mono">{{ maxLabel }}</span>
        <span class="absolute bottom-0 left-0 text-[9px] leading-none text-gray-400 font-mono">{{ minLabel }}</span>
      </template>

      <svg v-else viewBox="0 0 100 40" preserveAspectRatio="none" class="w-full h-full opacity-20">
        <line x1="0" y1="20" x2="100" y2="20" stroke="currentColor" stroke-width="1.5" />
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'

const props = defineProps<{
  gaugeId: string
  flowStatus: WatchedGauge['flowStatus']
}>()

const { apiBase } = useRuntimeConfig().public
const PREF_KEY = 'h2oflow_sparkline_hours'

const hours   = ref<12 | 24>(12)
const loading = ref(true)
const readings = ref<{ cfs: number; timestamp: string }[]>([])

async function fetchReadings() {
  loading.value = true
  try {
    const limit = hours.value * 2
    const res = await fetch(`${apiBase}/api/v1/gauges/${props.gaugeId}/readings?limit=${limit}`)
    if (res.ok) readings.value = ([...(await res.json())]).reverse()
  } catch { /* fall through */ } finally {
    loading.value = false
  }
}

// Watcher declared after fetchReadings — fires on every hours change
watch(hours, (h) => {
  localStorage.setItem(PREF_KEY, String(h))
  fetchReadings()
})

onMounted(() => {
  // Read localStorage after mount (guaranteed client-side, avoids SSR mismatch)
  const saved = localStorage.getItem(PREF_KEY)
  if (saved === '24') {
    hours.value = 24  // triggers the watcher above → fetchReadings()
  } else {
    fetchReadings()   // default 12h, no watcher change needed
  }
})

// ---- Computed ---------------------------------------------------------------

const minCfs = computed(() => readings.value.length ? Math.min(...readings.value.map(r => r.cfs)) : 0)
const maxCfs = computed(() => readings.value.length ? Math.max(...readings.value.map(r => r.cfs)) : 0)

function fmt(n: number) {
  return n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(Math.round(n))
}
const minLabel = computed(() => fmt(minCfs.value))
const maxLabel = computed(() => fmt(maxCfs.value))

const points = computed(() => {
  const data = readings.value
  if (data.length < 2) return []
  const range = maxCfs.value - minCfs.value || 1
  return data.map((r, i) => ({
    x: (i / (data.length - 1)) * 100,
    y: 38 - ((r.cfs - minCfs.value) / range) * 36,
  }))
})

function toPath(pts: { x: number; y: number }[]): string {
  return pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
}

const linePath = computed(() => toPath(points.value))
const areaPath = computed(() => {
  const pts = points.value
  if (!pts.length) return ''
  const last = pts[pts.length - 1]
  return `${toPath(pts)} L${last.x.toFixed(1)},40 L0,40 Z`
})

const strokeColor = computed(() => ({
  runnable: '#10b981', // emerald — go paddle
  caution:  '#1f2937', // near-black — minimum, marginal
  low:      '#9ca3af', // gray — muted, not worth running
  flood:    '#3b82f6', // blue — too much water
  unknown:  '#6b7280',
}[props.flowStatus]))
</script>
