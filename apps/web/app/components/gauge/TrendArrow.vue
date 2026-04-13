<template>
  <span :class="arrowClass" :title="arrowTitle">{{ arrowGlyph }}</span>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const props = withDefaults(defineProps<{ gaugeId: string; size?: 'sm' | 'lg' }>(), { size: 'sm' })

// Trend derived from the two most recent readings fetched from the API.
// Falls back to 'flat' if unavailable (no history yet, API error, etc.).
type Trend = 'rising' | 'falling' | 'flat'
const trend = ref<Trend>('flat')

const { apiBase } = useRuntimeConfig().public

onMounted(async () => {
  try {
    const res = await fetch(`${apiBase}/api/v1/gauges/${props.gaugeId}/readings?limit=2`)
    if (!res.ok) return
    const data = await res.json() as { cfs: number; timestamp: string }[]
    if (data.length < 2) return
    // API returns newest-first
    const [latest, prev] = data
    const delta = latest.cfs - prev.cfs
    if (Math.abs(delta) < 5) trend.value = 'flat'
    else trend.value = delta > 0 ? 'rising' : 'falling'
  } catch {
    // ignore — flat is a safe default
  }
})

const arrowGlyph = computed(() => ({ rising: '↑', falling: '↓', flat: '→' }[trend.value]))
const arrowClass = computed(() => ({
  'text-red-500':    trend.value === 'rising',
  'text-blue-400':   trend.value === 'falling',
  'text-gray-400':   trend.value === 'flat',
  'text-sm font-semibold': props.size === 'sm',
  'text-xl font-bold': props.size === 'lg',
}))
const arrowTitle = computed(() => ({
  rising:  'Rising',
  falling: 'Falling',
  flat:    'Steady',
}[trend.value]))
</script>
