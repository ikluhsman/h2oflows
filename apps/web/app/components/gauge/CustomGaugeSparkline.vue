<template>
  <div class="w-full" @click.stop>
    <div class="relative w-full" :class="compact ? 'h-6' : 'h-10'">
      <span v-if="compact" class="absolute top-0 right-0 text-[9px] leading-none text-neutral-400 dark:text-neutral-500 font-mono z-10 pointer-events-none">48h</span>
      <div v-if="loading" class="w-full h-full rounded animate-pulse bg-neutral-100 dark:bg-neutral-800" />
      <template v-else-if="points.length >= 2">
        <svg viewBox="0 0 100 40" preserveAspectRatio="none" class="w-full h-full overflow-visible">
          <path :d="areaPath" fill="#3b82f6" fill-opacity="0.12" />
          <path :d="linePath" stroke="#3b82f6" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" fill="none" />
        </svg>
        <span class="absolute top-0 left-0 text-[9px] leading-none text-neutral-400 font-mono">{{ maxLabel }}</span>
        <span class="absolute bottom-0 left-0 text-[9px] leading-none text-neutral-400 font-mono">{{ minLabel }}</span>
      </template>
      <svg v-else viewBox="0 0 100 40" preserveAspectRatio="none" class="w-full h-full opacity-20">
        <line x1="0" y1="20" x2="100" y2="20" stroke="currentColor" stroke-width="1.5" />
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

const props = defineProps<{
  gaugeSlug: string
  compact?: boolean
}>()

const { apiBase } = useRuntimeConfig().public
const { getToken } = useAuth()

const loading  = ref(true)
const readings = ref<{ cfs: number; timestamp: string }[]>([])

async function fetchReadings() {
  loading.value = true
  try {
    const token = await getToken()
    const since = new Date(Date.now() - 48 * 3_600_000).toISOString()
    const res = await fetch(
      `${apiBase}/api/v1/me/custom-gauges/${props.gaugeSlug}/readings?since=${since}&limit=500`,
      { headers: token ? { Authorization: `Bearer ${token}` } : {} },
    )
    if (res.ok) readings.value = ([...(await res.json())]).reverse()
    else readings.value = []
  } catch {
    readings.value = []
  } finally {
    loading.value = false
  }
}

onMounted(fetchReadings)

const points = computed(() => readings.value.map(r => r.cfs))

const minVal  = computed(() => Math.min(...points.value))
const maxVal  = computed(() => Math.max(...points.value))
const range   = computed(() => maxVal.value - minVal.value || 1)

const minLabel = computed(() => minVal.value < 10_000 ? minVal.value.toLocaleString(undefined, { maximumFractionDigits: 0 }) : `${(minVal.value / 1000).toFixed(0)}k`)
const maxLabel = computed(() => maxVal.value < 10_000 ? maxVal.value.toLocaleString(undefined, { maximumFractionDigits: 0 }) : `${(maxVal.value / 1000).toFixed(0)}k`)

function px(i: number) { return (i / (points.value.length - 1)) * 100 }
function py(v: number) { return 40 - ((v - minVal.value) / range.value) * 38 }

const linePath = computed(() => {
  if (points.value.length < 2) return ''
  return points.value.map((v, i) => `${i === 0 ? 'M' : 'L'}${px(i).toFixed(1)},${py(v).toFixed(1)}`).join(' ')
})

const areaPath = computed(() => {
  if (points.value.length < 2) return ''
  const line = points.value.map((v, i) => `${i === 0 ? 'M' : 'L'}${px(i).toFixed(1)},${py(v).toFixed(1)}`).join(' ')
  return `${line} L100,40 L0,40 Z`
})
</script>
