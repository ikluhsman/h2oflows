<template>
  <UModal v-model:open="open" :ui="{ width: 'max-w-xl' }">
    <template #header>
      <div class="flex items-start justify-between gap-3 w-full">
        <div class="min-w-0 flex-1">
          <h2 class="text-lg font-bold text-neutral-900 dark:text-white truncate leading-tight">{{ reachName }}</h2>
          <p class="text-xs text-neutral-400 mt-0.5 flex items-center gap-1">
            <svg class="w-3 h-3 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="4" y="2" width="16" height="20" rx="2"/><line x1="8" y1="6" x2="16" y2="6"/><line x1="8" y1="10" x2="10" y2="10"/><line x1="14" y1="10" x2="16" y2="10"/><line x1="8" y1="14" x2="10" y2="14"/><line x1="14" y1="14" x2="16" y2="14"/>
            </svg>
            Calculated · {{ gaugeName }}
          </p>
        </div>
        <button
          class="shrink-0 p-1 rounded-md text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 transition-colors"
          aria-label="Close"
          @click="open = false"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6 6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>
    </template>

    <template #body>
      <div class="space-y-3">
        <!-- Current value + badge -->
        <div class="flex items-baseline gap-2 flex-wrap">
          <span class="text-3xl font-bold tabular-nums leading-none text-neutral-900 dark:text-white">
            {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-sm text-neutral-500">cfs</span>
          <span
            v-if="flowBand"
            :class="['inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', bandBadgeClass(flowBand, flowStatus)]"
          >{{ flowBandLabel(flowBand, flowStatus) }}</span>
        </div>

        <!-- Time window toggle -->
        <div class="flex justify-end">
          <div class="flex text-xs rounded overflow-hidden border border-neutral-200 dark:border-neutral-700">
            <button
              v-for="h in ([12, 24, 48] as const)"
              :key="h"
              class="px-2 py-1 transition-colors"
              :class="hours === h ? 'bg-neutral-100 dark:bg-neutral-700 text-neutral-700 dark:text-neutral-200' : 'text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300'"
              @click="hours = h; load()"
            >{{ h }}h</button>
          </div>
        </div>

        <!-- Chart -->
        <div class="relative w-full" style="height:200px">
          <div ref="container" class="w-full h-full" />
          <div
            v-if="loading"
            class="absolute inset-0 flex items-center justify-center text-neutral-400 text-sm bg-white/70 dark:bg-neutral-900/70"
          >Loading…</div>
          <div
            v-else-if="readings.length === 0"
            class="absolute inset-0 flex items-center justify-center text-neutral-400 text-sm"
          >No readings in this window</div>
        </div>

        <!-- Link to reach detail page (user reach) or gauge page (standalone) -->
        <div class="pt-1 border-t border-neutral-100 dark:border-neutral-800">
          <NuxtLink
            v-if="reachSlug"
            :to="`/my/reaches/${reachSlug}`"
            class="inline-flex items-center gap-1 text-sm text-primary-500 hover:text-primary-600 dark:text-primary-400 dark:hover:text-primary-300 font-medium transition-colors"
            @click="open = false"
          >
            Edit reach
            <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M5 10h10M11 6l4 4-4 4"/>
            </svg>
          </NuxtLink>
          <NuxtLink
            v-else
            :to="`/my/gauges/${gaugeSlug}`"
            class="inline-flex items-center gap-1 text-sm text-primary-500 hover:text-primary-600 dark:text-primary-400 dark:hover:text-primary-300 font-medium transition-colors"
            @click="open = false"
          >
            Edit gauge
            <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M5 10h10M11 6l4 4-4 4"/>
            </svg>
          </NuxtLink>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onUnmounted, onMounted } from 'vue'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import { flowBandLabel } from '~/utils/flowBand'

const { bandBadgeClass } = useFlowBandPalette()

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  reachName: string
  reachSlug: string
  gaugeName: string
  gaugeSlug: string
  currentCfs: number | null
  flowBand: string | null
  flowStatus: string
}>()

const { apiBase } = useRuntimeConfig().public
const { getToken } = useAuth()

const loading   = ref(false)
const readings  = ref<Array<{ timestamp: string; cfs: number }>>([])
const hours     = ref<12 | 24 | 48>(48)
const container = ref<HTMLElement | null>(null)
let chart: uPlot | null = null

async function load() {
  loading.value = true
  try {
    const token  = await getToken()
    const since  = new Date(Date.now() - hours.value * 3_600_000).toISOString()
    const res    = await fetch(
      `${apiBase}/api/v1/me/custom-gauges/${props.gaugeSlug}/readings?since=${since}`,
      { headers: token ? { Authorization: `Bearer ${token}` } : {} },
    )
    if (res.ok) readings.value = await res.json()
    else readings.value = []
  } catch {
    readings.value = []
  } finally {
    loading.value = false
    await nextTick()
    buildChart()
  }
}

function buildChart() {
  if (!container.value) return
  chart?.destroy()
  chart = null
  if (readings.value.length === 0) return

  const sorted = [...readings.value].reverse()
  const xs = new Float64Array(sorted.map(r => new Date(r.timestamp).getTime() / 1000))
  const ys = new Float64Array(sorted.map(r => r.cfs))

  chart = new uPlot(
    {
      width:   container.value.clientWidth || 400,
      height:  200,
      padding: [8, 0, 0, 0],
      cursor:  { show: true },
      legend:  { show: false },
      axes: [
        { stroke: '#9ca3af', ticks: { stroke: '#374151' }, grid: { stroke: '#1f2937', width: 1 } },
        { stroke: '#9ca3af', ticks: { stroke: '#374151' }, grid: { stroke: '#1f2937', width: 1 } },
      ],
      series: [
        {},
        {
          stroke: '#3b82f6',
          width:  2,
          fill:   'rgba(59,130,246,0.08)',
        },
      ],
    },
    [xs, ys],
    container.value,
  )
}

// immediate: true catches the case where the component mounts with open already
// true (v-if becomes active in the same tick as open = true, so the non-immediate
// watcher never sees the false→true transition on first open).
watch(open, async (v) => {
  if (v) {
    await nextTick()
    load()
  } else {
    chart?.destroy()
    chart = null
    readings.value = []
  }
}, { immediate: true })

// If UModal renders body lazily, container ref becomes non-null after load() resolves.
// Rebuild chart whenever the container element mounts.
watch(container, (el) => {
  if (el && !loading.value && readings.value.length > 0) buildChart()
})

onUnmounted(() => { chart?.destroy() })
</script>
