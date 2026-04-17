<template>
  <div class="w-full" @click.stop>
    <!-- Window toggle — hidden in compact mode -->
    <div v-if="!compact" class="flex justify-end mb-0.5">
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
    <div class="relative w-full" :class="compact ? 'h-6' : 'h-10'">
      <span v-if="compact" class="absolute top-0 right-0 text-[9px] leading-none text-gray-400 dark:text-gray-500 font-mono z-10 pointer-events-none">{{ hours }}h</span>
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
  flowBandLabel?: string | null
  reachSlug?: string | null
  compact?: boolean
  color?: string  // override stroke color (e.g. '#3b82f6' for neutral blue)
}>()

const emit = defineEmits<{
  (e: 'latestCfs', cfs: number): void
  (e: 'liveFlowBand', band: { flowBandLabel: string | null; flowStatus: string }): void
}>()

const { apiBase } = useRuntimeConfig().public
const PREF_KEY = 'h2oflow_sparkline_hours'

const hours   = ref<12 | 24>(12)
const loading = ref(true)
const readings = ref<{ cfs: number; timestamp: string }[]>([])
// Flow ranges are fetched once per mount so we can compute the live band
// from the freshest reading and emit it upward, overriding the (potentially
// stale) flowBandLabel coming from the watchlist store.
const flowRanges = ref<{ label: string; min_cfs: number | null; max_cfs: number | null }[]>([])
let rangesLoaded = false

async function loadFlowRanges() {
  if (rangesLoaded) return
  try {
    const url = props.reachSlug
      ? `${apiBase}/api/v1/reaches/${props.reachSlug}/flow-ranges`
      : `${apiBase}/api/v1/gauges/${props.gaugeId}/flow-ranges`
    const res = await fetch(url)
    if (res.ok) flowRanges.value = await res.json()
    rangesLoaded = true
  } catch { /* fall through */ }
}

async function fetchReadings() {
  loading.value = true
  try {
    await loadFlowRanges()
    const since = new Date(Date.now() - hours.value * 3_600_000).toISOString()
    const res = await fetch(`${apiBase}/api/v1/gauges/${props.gaugeId}/readings?since=${since}&limit=500`)
    if (res.ok) {
      readings.value = ([...(await res.json())]).reverse()
      if (readings.value.length > 0) {
        const latestCfs = readings.value[readings.value.length - 1].cfs
        emit('latestCfs', latestCfs)
        const matched = flowRanges.value.find(fr =>
          (fr.min_cfs == null || latestCfs >= fr.min_cfs) &&
          (fr.max_cfs == null || latestCfs <  fr.max_cfs)
        )
        const liveLabel = matched?.label ?? null
        emit('liveFlowBand', { flowBandLabel: liveLabel, flowStatus: flowStatusForBand(liveLabel) })
      }
    }
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

const strokeColor = computed(() => props.color ?? flowBandSolidColor(props.flowBandLabel, props.flowStatus))
</script>
