<template>
  <!-- ─── LIST row ─────────────────────────────────────────────────────── -->
  <div
    v-if="density === 'list'"
    class="flex items-center gap-3 px-3 py-2 rounded-lg border transition-all duration-200 cursor-pointer"
    :class="cardClass"
    @click="emit('open')"
  >
    <div class="min-w-0 flex-1">
      <span class="text-sm font-medium truncate block">{{ displayName }}</span>
      <span v-if="gauge.riverName" class="text-xs text-blue-400 truncate block">{{ gauge.riverName }}</span>
    </div>

    <div class="w-40 shrink-0 hidden sm:block">
      <GaugeSparkline :gauge-id="gauge.id" :flow-status="gauge.flowStatus" compact />
    </div>

    <TrendArrow v-if="currentCfs != null" :gauge-id="gauge.id" size="lg" class="shrink-0 hidden sm:block" />

    <UBadge v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel" :color="statusColor" variant="subtle" size="sm" class="shrink-0 hidden sm:flex">
      {{ statusLabel }}
    </UBadge>

    <span class="text-base font-bold tabular-nums shrink-0 min-w-16 text-right" :class="cfsClass">
      {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
      <span class="text-xs font-normal text-gray-400 dark:text-gray-500">cfs</span>
    </span>

    <button
      class="rounded p-1 text-gray-300 dark:text-gray-600 hover:text-red-400 dark:hover:text-red-400 transition-colors shrink-0"
      aria-label="Remove gauge"
      @click.stop="removeAndSync(gauge.id, gauge.contextReachSlug)"
    >
      <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm4 0a1 1 0 112 0v6a1 1 0 11-2 0V8z" clip-rule="evenodd"/>
      </svg>
    </button>
  </div>

  <!-- ─── CARD views (compact / comfortable / full) ────────────────────── -->
  <div
    v-else
    class="relative rounded-xl border transition-all duration-200 cursor-pointer overflow-hidden"
    :class="[cardClass, density === 'compact' ? 'p-2.5' : density === 'comfortable' ? 'p-3' : 'p-4']"
    @click="emit('open')"
  >
    <!-- Compact: background sparkline along the bottom edge -->
    <div v-if="density === 'compact'" class="absolute bottom-0 left-0 right-0 h-10 pointer-events-none opacity-35">
      <GaugeSparkline :gauge-id="gauge.id" :flow-status="gauge.flowStatus" compact class="h-full w-full" />
    </div>

    <!-- Gauge name + reach subtitle -->
    <div class="relative flex items-start justify-between gap-2" :class="density === 'compact' ? 'mb-1' : 'mb-2'">
      <div class="min-w-0 flex-1">
        <UTooltip :text="displayName" :delay-duration="500">
          <span class="font-medium truncate block" :class="density === 'compact' ? 'text-xs' : 'text-sm'">{{ displayName }}</span>
        </UTooltip>
        <span v-if="gauge.contextReachRiverName ?? gauge.riverName" class="text-xs text-blue-400/70 truncate block">{{ gauge.contextReachRiverName ?? gauge.riverName }}</span>
        <!-- Full: reach full name + gauge source/id as subtitle -->
        <p v-if="!hideReachSubtitle && density === 'full'" class="text-xs truncate mt-0.5">
          <span v-if="contextFullName" class="text-gray-400">{{ contextFullName }}</span>
          <span v-if="contextFullName && gauge.externalId" class="text-gray-300 dark:text-gray-600"> · </span>
          <span class="text-gray-400 dark:text-gray-500">{{ gauge.source.toUpperCase() }} {{ gauge.externalId }}</span>
        </p>
      </div>

      <!-- Remove button -->
      <UTooltip text="Remove gauge">
        <button
          class="rounded-lg transition-all duration-150 text-gray-300 dark:text-gray-600 hover:text-red-400 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/40"
          :class="density === 'compact' ? 'p-1' : 'p-1.5'"
          aria-label="Remove gauge"
          @click.stop="removeAndSync(gauge.id, gauge.contextReachSlug)"
        >
          <svg :class="density === 'compact' ? 'w-3 h-3' : 'w-4 h-4'" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm4 0a1 1 0 112 0v6a1 1 0 11-2 0V8z" clip-rule="evenodd"/>
          </svg>
        </button>
      </UTooltip>
    </div>

    <!-- Current flow reading -->
    <div class="relative flex items-baseline gap-2 flex-wrap" :class="density === 'compact' ? 'mb-6' : 'mb-1.5'">
      <span class="font-bold tabular-nums leading-none" :class="[cfsClass, density === 'compact' ? 'text-xl' : density === 'comfortable' ? 'text-2xl' : 'text-3xl']">
        {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
      </span>
      <span class="text-xs text-gray-500">cfs</span>
      <!-- Trending arrow — shown in comfortable + full -->
      <TrendArrow v-if="currentCfs != null && density !== 'compact'" :gauge-id="gauge.id" class="text-lg" />
      <!-- Badge inline with CFS on sm+ for comfortable mode only -->
      <UBadge
        v-if="density === 'comfortable' && (gauge.flowStatus !== 'unknown' || gauge.flowBandLabel)"
        :color="statusColor" variant="subtle" size="sm"
        class="hidden sm:inline-flex"
      >{{ statusLabel }}</UBadge>
    </div>

    <!-- Flow status badge — full: always shown; comfortable: mobile only -->
    <div
      v-if="(gauge.flowStatus !== 'unknown' || gauge.flowBandLabel) && density !== 'compact'"
      class="flex items-center gap-2 mb-2"
      :class="density === 'full' ? '' : 'sm:hidden'"
    >
      <UBadge :color="statusColor" variant="subtle" :size="density === 'full' ? 'md' : 'sm'">{{ statusLabel }}</UBadge>
      <span v-if="gauge.flowStatus === 'flood'" class="relative flex h-2 w-2">
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75" />
        <span class="relative inline-flex rounded-full h-2 w-2 bg-blue-400" />
      </span>
    </div>

    <!-- Sparkline — comfortable gets compact sparkline; full gets full sparkline -->
    <GaugeSparkline v-if="density === 'comfortable'" :gauge-id="gauge.id" :flow-status="gauge.flowStatus" compact class="mb-1" />
    <GaugeSparkline v-else-if="density === 'full'" :gauge-id="gauge.id" :flow-status="gauge.flowStatus" class="mb-2" />

    <!-- Last updated — full only, when no subtitle already shows source info -->
    <p v-if="density === 'full' && !contextFullName" class="text-xs text-gray-400 mt-1 truncate">
      {{ gauge.source.toUpperCase() }} · {{ gauge.externalId }}
    </p>
    <p v-if="density === 'full' && lastUpdatedLabel" class="text-xs text-gray-400 mt-0.5">{{ lastUpdatedLabel }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'

const props = defineProps<{
  gauge: WatchedGauge
  hideReachSubtitle?: boolean
  density?: 'compact' | 'comfortable' | 'full' | 'list'
}>()
const emit = defineEmits<{ (e: 'open'): void }>()

const { removeAndSync } = useWatchlistSync()

const currentCfs = computed(() => props.gauge.currentCfs)

// --- Display name -----------------------------------------------------------
// Prefer the context reach's common name (e.g. "Foxton") over the raw gauge name.
const displayName = computed(() =>
  props.gauge.contextReachCommonName
  ?? props.gauge.name
  ?? props.gauge.externalId
)

// Full-mode subtitle: reach full name if available (e.g. "Buffalo Creek to South Platte")
const contextFullName = computed(() => props.gauge.contextReachFullName ?? null)

// --- Flow status ------------------------------------------------------------

const statusColor = computed(() => {
  const band = props.gauge.flowBandLabel
  if (band === 'low_runnable')  return 'lime'
  if (band === 'med_runnable')  return 'emerald'
  if (band === 'high_runnable') return 'green'
  switch (props.gauge.flowStatus) {
    case 'runnable': return 'emerald'
    case 'caution':  return 'warning'
    case 'low':      return 'error'
    case 'flood':    return 'info'
    default:         return 'neutral'
  }
})
const statusLabel = computed(() => {
  if (props.gauge.flowBandLabel) {
    return props.gauge.flowBandLabel
      .split('_')
      .map(w => w.charAt(0).toUpperCase() + w.slice(1))
      .join(' ')
  }
  switch (props.gauge.flowStatus) {
    case 'runnable': return 'Runnable'
    case 'caution':  return 'Minimum'
    case 'low':      return 'Too Low'
    case 'flood':    return 'Flood Stage'
    default:         return 'Unknown'
  }
})
const cfsClass = computed(() => {
  const band = props.gauge.flowBandLabel
  if (band === 'low_runnable')  return 'text-lime-500'
  if (band === 'med_runnable')  return 'text-emerald-500'
  if (band === 'high_runnable') return 'text-green-600 dark:text-green-500'
  return {
    'text-emerald-400 dark:text-emerald-500': props.gauge.flowStatus === 'runnable',
    'text-amber-400':                         props.gauge.flowStatus === 'caution',
    'text-red-400':                           props.gauge.flowStatus === 'low',
    'text-blue-400 dark:text-blue-500':       props.gauge.flowStatus === 'flood',
    'text-gray-400':                          !props.gauge.flowStatus || props.gauge.flowStatus === 'unknown',
  }
})

// --- Card chrome ------------------------------------------------------------

const cardClass = computed(() => ({
  'border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900': true,
}))

// --- Last updated -----------------------------------------------------------

const lastUpdatedLabel = computed(() => {
  if (!props.gauge.lastReadingAt) return ''
  const ms = Date.now() - new Date(props.gauge.lastReadingAt).getTime()
  const minutes = Math.floor(ms / 60_000)
  if (minutes < 1)  return 'Updated just now'
  if (minutes < 60) return `Updated ${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24)   return `Updated ${hours}h ago`
  return `Updated ${Math.floor(hours / 24)}d ago`
})
</script>
