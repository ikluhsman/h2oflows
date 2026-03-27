<template>
  <UModal v-model:open="open" :ui="{ width: 'max-w-xl' }">
    <template #header>
      <div class="flex items-center justify-between gap-3 w-full">
        <div class="min-w-0">
          <h2 class="font-semibold truncate">{{ displayName }}</h2>
          <p class="text-xs text-gray-400 truncate mt-0.5">
            <a :href="sourceUrl" target="_blank" rel="noopener" class="hover:text-blue-400 underline underline-offset-2">
              {{ gauge.source.toUpperCase() }} · {{ gauge.externalId }}
            </a>
            <span v-if="gauge.watershedName"> · {{ gauge.watershedName }}</span>
          </p>
        </div>

        <!-- Current CFS + badge -->
        <div class="shrink-0 text-right">
          <span class="text-2xl font-bold tabular-nums" :class="cfsClass">
            {{ gauge.currentCfs != null ? gauge.currentCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-500 ml-1">cfs</span>
          <div class="mt-1">
            <UBadge :color="statusColor" variant="subtle" size="xs">{{ statusLabel }}</UBadge>
          </div>
        </div>
      </div>
    </template>

    <template #body>
      <div class="space-y-4">
        <!-- 48-hour graph -->
        <GaugeGraph :gauge-id="gauge.id" :current-cfs="gauge.currentCfs" />

        <!-- Last updated -->
        <p v-if="gauge.lastReadingAt" class="text-xs text-gray-500">
          Last reading {{ lastReadingRelative }}
        </p>

        <!-- Reach link -->
        <div v-if="gauge.reachName" class="border-t border-gray-100 dark:border-gray-800 pt-3">
          <NuxtLink
            :to="`/reaches/${gauge.reachSlug}`"
            class="text-sm font-medium text-blue-500 hover:text-blue-400 flex items-center gap-1"
          >
            {{ gauge.reachName }}
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6M15 3h6v6M10 14L21 3"/>
            </svg>
          </NuxtLink>
          <p class="text-xs text-gray-400 mt-0.5">View reach details, rapids, and access</p>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ gauge: WatchedGauge }>()

const displayName = computed(() =>
  props.gauge.reachName ?? props.gauge.name ?? props.gauge.externalId
)

const sourceUrl = computed(() => {
  switch (props.gauge.source) {
    case 'usgs':
      return `https://waterdata.usgs.gov/monitoring-location/${props.gauge.externalId}/`
    case 'dwr':
      return `https://dwr.state.co.us/Tools/Stations/${props.gauge.externalId}`
    default:
      return '#'
  }
})

const statusColor = computed(() => {
  switch (props.gauge.flowStatus) {
    case 'runnable': return 'success'
    case 'caution':  return 'neutral'
    case 'low':      return 'neutral'
    case 'flood':    return 'info'
    default:         return 'neutral'
  }
})

const statusLabel = computed(() => {
  switch (props.gauge.flowStatus) {
    case 'runnable': return 'Runnable'
    case 'caution':  return 'Caution'
    case 'low':      return 'Too Low'
    case 'flood':    return 'Flood Stage'
    default:         return 'Unknown'
  }
})

const cfsClass = computed(() => ({
  'text-emerald-500':                 props.gauge.flowStatus === 'runnable',
  'text-gray-900 dark:text-gray-100': props.gauge.flowStatus === 'caution',
  'text-gray-400':                    props.gauge.flowStatus === 'low',
  'text-blue-500':                    props.gauge.flowStatus === 'flood',
  'text-gray-300':                    props.gauge.flowStatus === 'unknown',
}))

const lastReadingRelative = computed(() => {
  if (!props.gauge.lastReadingAt) return ''
  const ms = Date.now() - new Date(props.gauge.lastReadingAt).getTime()
  const m = Math.floor(ms / 60_000)
  if (m < 1)  return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m ago`
})
</script>
