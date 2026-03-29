<template>
  <div class="space-y-3">
    <!-- Time window toggle -->
    <div class="flex justify-end">
      <div class="flex text-xs rounded overflow-hidden border border-gray-200 dark:border-gray-700">
        <button
          v-for="h in ([12, 24, 48] as const)"
          :key="h"
          class="px-2 py-1 transition-colors"
          :class="hours === h ? 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200' : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'"
          @click="hours = h"
        >{{ h }}h</button>
      </div>
    </div>

    <!-- Seasonal context banner -->
    <GaugeSeasonalBanner :gauge-id="gaugeId" :current-cfs="currentCfs" />

    <!-- Diurnal cycle banner -->
    <div
      v-if="diurnal.detected"
      class="flex items-center gap-2 rounded-lg px-3 py-2 text-xs bg-sky-50 dark:bg-sky-950 text-sky-700 dark:text-sky-300"
    >
      <span class="text-base">🌡</span>
      <span>
        <strong>Diurnal cycle</strong> —
        {{ diurnalPhaseLabel }}
        <template v-if="diurnal.estimatedPeakHour != null">
          · Est. peak {{ formatHour(diurnal.estimatedPeakHour) }}
          (~{{ diurnal.peakCfs?.toLocaleString() }} cfs)
        </template>
        <template v-if="diurnal.swingPct != null">
          · {{ diurnal.swingPct }}% daily swing
        </template>
      </span>
    </div>

    <!-- Chart container — always mounted so the ref is never torn down mid-update.
         Overlay states sit on top without removing the canvas from the DOM. -->
    <div class="relative w-full" style="height:200px">
      <div ref="container" class="w-full h-full" />
      <!-- Hover tooltip -->
      <div
        ref="tooltipEl"
        class="absolute pointer-events-none text-xs bg-gray-900/90 text-white rounded px-1.5 py-0.5 whitespace-nowrap translate-x-2 -translate-y-full"
        style="display:none; z-index:10"
      />
      <div
        v-if="loading"
        class="absolute inset-0 flex items-center justify-center text-gray-400 text-sm bg-white/70 dark:bg-gray-900/70"
      >Loading…</div>
      <div
        v-else-if="readings.length === 0"
        class="absolute inset-0 flex items-center justify-center text-gray-400 text-sm"
      >No readings in this window</div>
    </div>

    <!-- Flow range legend + provenance -->
    <div v-if="flowRanges.length > 0" class="space-y-1">
      <div class="flex flex-wrap gap-x-4 gap-y-1.5 text-xs text-gray-500">
        <span
          v-for="fr in flowRanges"
          :key="fr.label"
          class="flex items-center gap-1.5"
        >
          <span class="inline-block w-2.5 h-2.5 rounded-sm flex-shrink-0" :style="{ background: bandColor(fr.label) }" />
          <span class="font-medium">{{ labelDisplay(fr.label) }}</span>
          <span class="text-gray-400">
            {{ fr.min_cfs != null ? fr.min_cfs.toLocaleString() : '—' }}–{{ fr.max_cfs != null ? fr.max_cfs.toLocaleString() : '∞' }} cfs
          </span>
          <DataSourceBadge
            :source="(fr.data_source as any) ?? 'manual'"
            :verified="fr.verified"
            :confidence="fr.ai_confidence ?? undefined"
          />
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onMounted, onUnmounted, nextTick } from 'vue'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import { useDiurnalPattern } from '~/composables/useDiurnalPattern'

// ---- Types ------------------------------------------------------------------

interface Reading {
  cfs:       number
  timestamp: string
}

interface FlowRange {
  label:         string
  min_cfs:       number | null
  max_cfs:       number | null
  craft_type:    string
  class_modifier: number | null
  source_url?:   string
  data_source:   string   // 'manual' | 'ai_seed' | 'ai_web' | 'community'
  ai_confidence: number | null
  verified:      boolean
}

// ---- Props ------------------------------------------------------------------

const props = defineProps<{
  gaugeId: string
  currentCfs?: number | null
}>()

// ---- State ------------------------------------------------------------------

const container  = ref<HTMLElement | null>(null)
const tooltipEl  = ref<HTMLElement | null>(null)
const loading    = ref(true)
const readings   = ref<Reading[]>([])
const flowRanges = ref<FlowRange[]>([])
let chart: uPlot | null = null

const { apiBase } = useRuntimeConfig().public

// ---- Data fetching ----------------------------------------------------------

const hours = ref<12 | 24 | 48>(48)

async function load() {
  loading.value = true
  try {
    const since = new Date(Date.now() - hours.value * 3_600_000).toISOString()
    const [rdRes, frRes] = await Promise.all([
      fetch(`${apiBase}/api/v1/gauges/${props.gaugeId}/readings?since=${since}&limit=500`),
      fetch(`${apiBase}/api/v1/gauges/${props.gaugeId}/flow-ranges`),
    ])
    if (rdRes.ok) readings.value = await rdRes.json()
    if (frRes.ok) flowRanges.value = await frRes.json()
  } finally {
    loading.value = false
    await nextTick()
    buildChart()
  }
}

// ---- Chart ------------------------------------------------------------------

function buildChart() {
  if (!container.value || readings.value.length === 0) return
  chart?.destroy()
  chart = null

  // API returns newest-first; uPlot needs ascending timestamps.
  const sorted = [...readings.value].reverse()
  const xs = new Float64Array(sorted.map(r => new Date(r.timestamp).getTime() / 1000))
  const ys = new Float64Array(sorted.map(r => r.cfs))

  const ranges = flowRanges.value
  const currentCfs = props.currentCfs ?? null

  const opts: uPlot.Options = {
    width:  container.value!.clientWidth,
    height: 200,
    padding: [8, 0, 0, 0],
    cursor: { show: true },
    legend: { show: false },   // we render our own legend below
    axes: [
      {
        stroke:  '#9ca3af',
        ticks:   { stroke: '#374151' },
        grid:    { stroke: '#1f2937', width: 1 },
      },
      {
        stroke:  '#9ca3af',
        ticks:   { stroke: '#374151' },
        grid:    { stroke: '#1f2937', width: 1 },
      },
    ],
    series: [
      {},
      {
        stroke: lineColor(ranges, currentCfs),
        width:  2,
        fill:   lineColor(ranges, currentCfs) + '18',
        spanGaps: false,
      },
    ],
    hooks: {
      // Draw flow range bands behind the series line.
      drawClear: [u => drawBands(u, ranges)],
      // Draw a horizontal marker for the current reading.
      draw: [u => drawCurrentMarker(u, currentCfs)],
      // Hover tooltip
      setCursor: [u => {
        const el = tooltipEl.value
        if (!el) return
        const idx = u.cursor.idx
        if (idx == null) { el.style.display = 'none'; return }
        const val = u.data[1][idx]
        if (val == null) { el.style.display = 'none'; return }
        const x = u.valToPos(u.data[0][idx], 'x')
        const y = u.valToPos(val, 'y')
        const ts = new Date(u.data[0][idx] * 1000)
        const timeStr = ts.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
        const dateStr = ts.toLocaleDateString([], { month: 'short', day: 'numeric' })
        el.textContent = `${Math.round(val).toLocaleString()} cfs · ${dateStr} ${timeStr}`
        el.style.display = 'block'
        el.style.left = `${x}px`
        el.style.top = `${y}px`
      }],
    },
  }

  chart = new uPlot(opts, [xs, ys], container.value!)
}

// ---- Canvas drawing helpers -------------------------------------------------

const BAND_COLORS: Record<string, string> = {
  too_low: 'rgba(239,68,68,0.22)',
  minimum: 'rgba(249,115,22,0.22)',
  fun:     'rgba(34,197,94,0.30)',
  optimal: 'rgba(16,185,129,0.35)',
  pushy:   'rgba(234,179,8,0.28)',
  high:    'rgba(249,115,22,0.28)',
  flood:   'rgba(239,68,68,0.30)',
}

// bandColor returns the fill color for a flow range label (used by legend too).
function bandColor(label: string): string {
  return BAND_COLORS[label] ?? 'rgba(156,163,175,0.10)'
}

function drawBands(u: uPlot, ranges: FlowRange[]) {
  if (ranges.length === 0) return
  const { ctx, bbox } = u
  const dpr = devicePixelRatio

  ctx.save()
  // Clip to the plot area so bands don't bleed into axes.
  ctx.beginPath()
  ctx.rect(bbox.left, bbox.top, bbox.width, bbox.height)
  ctx.clip()

  for (const fr of ranges) {
    const color = BAND_COLORS[fr.label]
    if (!color) continue

    // Convert CFS values to canvas Y coordinates.
    // min_cfs null means the band extends to the bottom of the chart (too_low).
    // max_cfs null means it extends to the top (flood).
    const yMin = fr.max_cfs != null
      ? u.valToPos(fr.max_cfs, 'y', true) * dpr
      : bbox.top

    const yMax = fr.min_cfs != null
      ? u.valToPos(fr.min_cfs, 'y', true) * dpr
      : bbox.top + bbox.height

    const height = Math.abs(yMax - yMin)
    if (height <= 0) continue

    ctx.fillStyle = color
    ctx.fillRect(bbox.left, Math.min(yMin, yMax), bbox.width, height)
  }

  ctx.restore()
}

function drawCurrentMarker(u: uPlot, cfs: number | null) {
  if (cfs == null) return
  const { ctx, bbox } = u
  const dpr = devicePixelRatio

  const y = u.valToPos(cfs, 'y', true) * dpr
  if (y < bbox.top || y > bbox.top + bbox.height) return

  ctx.save()
  ctx.beginPath()
  ctx.setLineDash([4 * dpr, 3 * dpr])
  ctx.strokeStyle = 'rgba(255,255,255,0.5)'
  ctx.lineWidth = 1 * dpr
  ctx.moveTo(bbox.left, y)
  ctx.lineTo(bbox.left + bbox.width, y)
  ctx.stroke()
  ctx.restore()
}

// Determine the line color from current CFS and flow ranges.
function lineColor(ranges: FlowRange[], cfs: number | null): string {
  if (cfs == null || ranges.length === 0) return '#6b7280'
  const match = ranges.find(fr =>
    (fr.min_cfs == null || cfs >= fr.min_cfs) &&
    (fr.max_cfs == null || cfs <  fr.max_cfs)
  )
  if (!match) return '#6b7280'
  switch (match.label) {
    case 'fun':
    case 'optimal':   return '#22c55e'
    case 'minimum':
    case 'pushy':     return '#eab308'
    default:          return '#ef4444'
  }
}

// ---- Diurnal cycle ----------------------------------------------------------

const diurnal = computed(() => useDiurnalPattern(readings.value))

const diurnalPhaseLabel = computed(() => {
  switch (diurnal.value.phase) {
    case 'rising':     return 'Rising'
    case 'falling':    return 'Falling'
    case 'near_peak':  return 'Near peak'
    case 'near_trough': return 'Near trough'
    default:           return 'Stable'
  }
})

function formatHour(h: number): string {
  const ampm = h >= 12 ? 'pm' : 'am'
  const display = h % 12 === 0 ? 12 : h % 12
  return `${display}${ampm}`
}

// ---- Flow range legend helpers ----------------------------------------------

const LABEL_DISPLAY: Record<string, string> = {
  too_low: 'Too Low',
  minimum: 'Minimum',
  fun:     'Fun',
  optimal: 'Optimal',
  pushy:   'Pushy',
  high:    'High',
  flood:   'Flood',
}

function labelDisplay(label: string): string {
  return LABEL_DISPLAY[label] ?? label
}

// ---- Lifecycle --------------------------------------------------------------

// Declared after load + buildChart so references are unambiguous at setup time.
watch(hours, load)
watch(() => props.gaugeId, load)
watch(() => props.currentCfs, async () => { await nextTick(); buildChart() })

let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  load()
  resizeObserver = new ResizeObserver(() => {
    if (chart && container.value) {
      chart.setSize({ width: container.value.clientWidth, height: 200 })
    }
  })
  if (container.value) resizeObserver.observe(container.value)
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  chart?.destroy()
})
</script>
