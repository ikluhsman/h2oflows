<template>
  <UModal v-model:open="open" :ui="{ width: 'max-w-xl' }">
    <template #header>
      <div class="flex items-center justify-between gap-3 w-full">
        <div class="min-w-0">
          <NuxtLink
            v-if="primaryReachSlug"
            :to="`/reaches/${primaryReachSlug}`"
            class="font-semibold truncate block hover:text-blue-500 transition-colors"
            @click="open = false"
          >{{ displayName }}</NuxtLink>
          <h2 v-else class="font-semibold truncate">{{ displayName }}</h2>
          <p class="text-xs text-gray-400 truncate mt-0.5">
            <a :href="sourceUrl" target="_blank" rel="noopener" class="hover:text-blue-400 underline underline-offset-2">
              {{ gauge.source.toUpperCase() }} · {{ gauge.externalId }}
            </a>
            <span v-if="gauge.watershedName"> · {{ gauge.watershedName }}</span>
          </p>
        </div>

        <!-- Current CFS -->
        <div class="shrink-0 text-right">
          <span class="text-2xl font-bold tabular-nums" :class="cfsClass">
            {{ gauge.currentCfs != null ? gauge.currentCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-500 ml-1">cfs</span>
        </div>

        <button
          class="shrink-0 p-1 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
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
      <div class="space-y-4">
        <!-- 48-hour graph — use context reach's flow ranges when available -->
        <GaugeGraph
          :gauge-id="gauge.id"
          :current-cfs="gauge.currentCfs"
          :reach-slug="gauge.contextReachSlug ?? gauge.reachSlug"
        />

        <!-- Last updated -->
        <p v-if="gauge.lastReadingAt" class="text-xs text-gray-500">
          Last reading {{ lastReadingRelative }}
        </p>

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
  props.gauge.contextReachCommonName
  ?? props.gauge.reachName
  ?? props.gauge.name
  ?? props.gauge.externalId
)

const primaryReachSlug = computed(() =>
  props.gauge.contextReachSlug ?? props.gauge.reachSlug ?? props.gauge.reachSlugs?.[0] ?? null
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

const cfsClass = computed(() => ({
  'text-emerald-500': props.gauge.flowStatus === 'runnable',
  'text-amber-400':   props.gauge.flowStatus === 'caution',
  'text-red-500':     props.gauge.flowStatus === 'low',
  'text-blue-500':    props.gauge.flowStatus === 'flood',
  'text-gray-400':    props.gauge.flowStatus === 'unknown',
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
