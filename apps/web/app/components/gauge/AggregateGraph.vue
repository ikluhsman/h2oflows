<template>
  <div ref="container" class="w-full h-48" />
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import type { WatchedGauge } from '~/stores/watchlist'

const props = defineProps<{ gauges: WatchedGauge[] }>()
const container = ref<HTMLElement | null>(null)

const { apiBase } = useRuntimeConfig().public

let chart: uPlot | null = null

// Palette for multi-series lines (up to 8 gauges shown simultaneously)
const COLORS = ['#22c55e', '#3b82f6', '#f59e0b', '#ec4899', '#8b5cf6', '#06b6d4', '#ef4444', '#84cc16']

async function buildChart() {
  if (!container.value) return
  chart?.destroy()
  chart = null

  // Fetch 48h history for each gauge in parallel
  const series = await Promise.all(
    props.gauges.map(async (g, i) => {
      try {
        const res = await fetch(`${apiBase}/api/v1/gauges/${g.id}/readings?limit=96`)
        if (!res.ok) return null
        const rows: { cfs: number; timestamp: string }[] = await res.json()
        // uPlot wants ascending timestamps
        rows.reverse()
        return {
          label: g.reachName ?? g.name ?? g.externalId,
          color: COLORS[i % COLORS.length],
          xs: rows.map(r => Math.floor(new Date(r.timestamp).getTime() / 1000)),
          ys: rows.map(r => r.cfs),
        }
      } catch {
        return null
      }
    })
  )

  const valid = series.filter(Boolean) as NonNullable<typeof series[0]>[]
  if (valid.length === 0) return

  // Align all series onto a shared time axis (union of all timestamps, fill with null)
  const allXs = Array.from(new Set(valid.flatMap(s => s!.xs))).sort((a, b) => a - b)
  const xsMap = new Map(allXs.map((x, i) => [x, i]))

  const data: uPlot.AlignedData = [
    new Float64Array(allXs),
    ...valid.map(s => {
      const arr = new Float64Array(allXs.length).fill(NaN)
      s!.xs.forEach((x, i) => { arr[xsMap.get(x)!] = s!.ys[i] })
      return arr
    }),
  ]

  const opts: uPlot.Options = {
    width:  container.value!.clientWidth,
    height: 192,
    cursor: { sync: { key: 'h2oflow-agg' } },
    axes: [
      { stroke: '#6b7280', ticks: { stroke: '#374151' } },
      { label: 'cfs', stroke: '#6b7280', ticks: { stroke: '#374151' } },
    ],
    series: [
      {},
      ...valid.map((s, i) => ({
        label: s!.label,
        stroke: COLORS[i % COLORS.length],
        width: 2,
        spanGaps: false,
      })),
    ],
  }

  chart = new uPlot(opts, data, container.value!)
}

onMounted(buildChart)
watch(() => props.gauges, buildChart, { deep: true })

let resizeObserver: ResizeObserver | null = null
onMounted(() => {
  resizeObserver = new ResizeObserver(() => {
    if (chart && container.value) {
      chart.setSize({ width: container.value.clientWidth, height: 192 })
    }
  })
  if (container.value) resizeObserver.observe(container.value)
})
onUnmounted(() => { resizeObserver?.disconnect(); chart?.destroy() })
</script>
