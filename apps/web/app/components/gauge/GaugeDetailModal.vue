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

        <!-- Current CFS -->
        <div class="shrink-0 text-right">
          <span class="text-2xl font-bold tabular-nums" :class="cfsClass">
            {{ gauge.currentCfs != null ? gauge.currentCfs.toLocaleString() : '—' }}
          </span>
          <span class="text-xs text-gray-500 ml-1">cfs</span>
        </div>
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

        <!-- Reach links — one per reach that uses this gauge -->
        <div v-if="reachLinks.length > 0" class="border-t border-gray-100 dark:border-gray-800 pt-3 space-y-1.5">
          <p class="text-xs text-gray-400 mb-2">{{ reachLinks.length === 1 ? 'Reach' : 'Reaches on this gauge' }}</p>
          <NuxtLink
            v-for="r in reachLinks"
            :key="r.slug"
            :to="`/reaches/${r.slug}`"
            class="flex items-center justify-between gap-2 rounded-lg px-3 py-2 bg-gray-50 dark:bg-gray-900 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            @click="open = false"
          >
            <span class="text-sm font-medium">{{ r.name }}</span>
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 18l6-6-6-6"/>
            </svg>
          </NuxtLink>
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
  props.gauge.contextReachCommonName
  ?? props.gauge.reachName
  ?? props.gauge.name
  ?? props.gauge.externalId
)

// Zip reachNames + reachSlugs into link objects. Falls back to the single
// reachName/reachSlug fields for gauges loaded from an older persisted state.
const reachLinks = computed(() => {
  const names = props.gauge.reachNames ?? []
  const slugs = props.gauge.reachSlugs ?? []
  if (names.length > 0 && slugs.length > 0) {
    return slugs.map((slug: string, i: number) => ({ slug, name: names[i] ?? slug }))
  }
  if (props.gauge.reachSlug && props.gauge.reachName) {
    return [{ slug: props.gauge.reachSlug, name: props.gauge.reachName }]
  }
  return []
})

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
