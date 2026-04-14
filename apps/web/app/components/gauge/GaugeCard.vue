<template>
  <!-- ─── LIST row ─────────────────────────────────────────────────────── -->
  <div
    v-if="density === 'list'"
    class="flex items-center gap-2 sm:gap-3 px-3 py-2 rounded-lg border transition-all duration-200 cursor-pointer"
    :class="cardClass"
    @click="emit('open')"
  >
    <!-- Name + badge + trend arrow -->
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-1.5 min-w-0">
        <span class="text-sm font-medium truncate">{{ displayName }}</span>
        <span
          v-if="displayFlowStatus !== 'unknown' || displayFlowBand"
          :class="['hidden sm:inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium shrink-0', statusBadgeClass]"
        >{{ statusLabel }}</span>
        <TrendArrow v-if="currentCfs != null" :gauge-id="gauge.id" size="lg" class="shrink-0 hidden sm:block" />
      </div>
      <span v-if="gauge.riverName" class="text-xs text-blue-400 truncate block">{{ gauge.riverName }}</span>
    </div>

    <!-- Sparkline + CFS -->
    <div class="flex items-center gap-2 shrink-0">
      <div class="w-32 shrink-0 hidden sm:block">
        <GaugeSparkline :gauge-id="gauge.id" :flow-status="displayFlowStatus" :flow-band-label="displayFlowBand" :reach-slug="gauge.contextReachSlug ?? gauge.reachSlug" compact @latest-cfs="liveCfs = $event" @live-flow-band="liveFlowBand = $event" />
      </div>
      <span class="text-base font-bold tabular-nums min-w-14 text-right" :class="cfsClass">
        {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
        <span class="text-xs font-normal text-gray-400 dark:text-gray-500">cfs</span>
      </span>
    </div>

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
      <GaugeSparkline :gauge-id="gauge.id" :flow-status="displayFlowStatus" :flow-band-label="displayFlowBand" :reach-slug="gauge.contextReachSlug ?? gauge.reachSlug" compact class="h-full w-full" @latest-cfs="liveCfs = $event" @live-flow-band="liveFlowBand = $event" />
    </div>

    <!-- Gauge name + reach subtitle -->
    <div class="relative flex items-start justify-between gap-2" :class="density === 'compact' ? 'mb-1' : 'mb-2'">
      <div class="min-w-0 flex-1">
        <UTooltip :text="displayName" :delay-duration="500">
          <span class="font-medium truncate block" :class="density === 'compact' ? 'text-xs' : 'text-sm'">{{ displayName }}</span>
        </UTooltip>
        <span v-if="density === 'full' && (gauge.contextReachRiverName ?? gauge.riverName)" class="text-xs text-blue-400/70 truncate block">{{ gauge.contextReachRiverName ?? gauge.riverName }}</span>
        <!-- Permit / multi-day micro-icons (full density only) -->
        <span v-if="density === 'full' && (gauge.contextReachPermitRequired || gauge.contextReachMultiDayDays > 1)" class="inline-flex items-center gap-1.5 mt-0.5">
          <UTooltip v-if="gauge.contextReachPermitRequired" text="Permit Required">
            <svg class="w-3 h-3 text-amber-500 dark:text-amber-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M12 7V5a2 2 0 00-2-2H9a2 2 0 00-2 2v6"/><circle cx="12" cy="16" r="1" fill="currentColor" stroke="none"/></svg>
          </UTooltip>
          <UTooltip v-if="gauge.contextReachMultiDayDays > 1" :text="`${gauge.contextReachMultiDayDays}-Day Trip`">
            <svg class="w-3 h-3 text-blue-400 dark:text-blue-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><path d="M16 2v4M8 2v4M3 10h18"/></svg>
          </UTooltip>
        </span>
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
      <span
        v-if="density === 'comfortable' && (displayFlowStatus !== 'unknown' || displayFlowBand)"
        :class="['hidden sm:inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', statusBadgeClass]"
      >{{ statusLabel }}</span>
    </div>

    <!-- Flow status badge — full: always shown; comfortable: mobile only -->
    <div
      v-if="(displayFlowStatus !== 'unknown' || displayFlowBand) && density !== 'compact'"
      class="flex items-center gap-2 mb-2"
      :class="density === 'full' ? '' : 'sm:hidden'"
    >
      <span :class="['inline-flex items-center rounded-md font-medium', density === 'full' ? 'px-2 py-0.5 text-sm' : 'px-1.5 py-0.5 text-xs', statusBadgeClass]">{{ statusLabel }}</span>
      <span v-if="displayFlowBand === 'above_recommended'" class="relative flex h-2 w-2">
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75" />
        <span class="relative inline-flex rounded-full h-2 w-2 bg-amber-400" />
      </span>
    </div>

    <!-- Sparkline — comfortable gets compact sparkline; full gets full sparkline -->
    <GaugeSparkline v-if="density === 'comfortable'" :gauge-id="gauge.id" :flow-status="displayFlowStatus" :flow-band-label="displayFlowBand" :reach-slug="gauge.contextReachSlug ?? gauge.reachSlug" compact class="mb-1" @latest-cfs="liveCfs = $event" @live-flow-band="liveFlowBand = $event" />
    <GaugeSparkline v-else-if="density === 'full'" :gauge-id="gauge.id" :flow-status="displayFlowStatus" :flow-band-label="displayFlowBand" :reach-slug="gauge.contextReachSlug ?? gauge.reachSlug" class="mb-2" @latest-cfs="liveCfs = $event" @live-flow-band="liveFlowBand = $event" />

    <!-- Diurnal forecast — compact/comfortable: one-liner; full: richer summary -->
    <p v-if="diurnal.detected && diurnal.forecast && density !== 'full'" class="relative text-[10px] text-indigo-500 dark:text-indigo-400 truncate">
      {{ diurnal.forecast.label }}
    </p>
    <div v-if="diurnal.detected && density === 'full'" class="flex items-center gap-2 text-[11px] text-indigo-500 dark:text-indigo-400 mt-0.5 mb-1">
      <svg class="w-3 h-3 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
      <span class="truncate">
        {{ diurnalPhaseLabel }}
        <template v-if="diurnal.forecast"> · {{ diurnal.forecast.label }}</template>
        <template v-if="diurnal.swingPct != null"> · {{ diurnal.swingPct }}% swing</template>
      </span>
    </div>

    <!-- Last updated — full only, when no subtitle already shows source info -->
    <p v-if="density === 'full' && !contextFullName" class="text-xs text-gray-400 mt-1 truncate">
      {{ gauge.source.toUpperCase() }} · {{ gauge.externalId }}
    </p>
    <p v-if="density === 'full' && lastUpdatedLabel" class="text-xs text-gray-400 mt-0.5">{{ lastUpdatedLabel }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { WatchedGauge } from '~/stores/watchlist'
import { useDiurnalCache } from '~/composables/useDiurnalCache'

const props = defineProps<{
  gauge: WatchedGauge
  hideReachSubtitle?: boolean
  density?: 'compact' | 'comfortable' | 'full' | 'list'
}>()
const emit = defineEmits<{ (e: 'open'): void }>()

const { removeAndSync } = useWatchlistSync()

const { pattern: diurnal } = useDiurnalCache(props.gauge.id)

const diurnalPhaseLabel = computed(() => {
  switch (diurnal.value.phase) {
    case 'rising':       return 'Rising'
    case 'falling':      return 'Falling'
    case 'near_peak':    return 'Near peak'
    case 'near_trough':  return 'Near trough'
    case 'stable':       return 'Stable'
    default:             return ''
  }
})

// liveCfs / liveFlowBand are set by GaugeSparkline once it loads fresh readings
// and flow ranges — both supersede the (potentially-stale) watchlist store values.
const liveCfs       = ref<number | null>(null)
const liveFlowBand  = ref<{ flowBandLabel: string | null; flowStatus: string } | null>(null)
const currentCfs        = computed(() => liveCfs.value ?? props.gauge.currentCfs)
const displayFlowBand   = computed(() => liveFlowBand.value?.flowBandLabel ?? props.gauge.flowBandLabel)
const displayFlowStatus = computed(() => liveFlowBand.value?.flowStatus   ?? props.gauge.flowStatus)

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

const statusBadgeClass = computed(() => flowBandBadgeClass(displayFlowBand.value, displayFlowStatus.value))
const statusLabel      = computed(() => flowBandLabel(displayFlowBand.value, displayFlowStatus.value))
const cfsClass         = computed(() => flowBandCfsClass(displayFlowBand.value, displayFlowStatus.value))

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
