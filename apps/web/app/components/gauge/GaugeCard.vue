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
        <span
          v-if="isUnhealthy"
          :class="['hidden sm:inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium shrink-0', healthBadgeClass]"
        >{{ healthBadgeLabel }}</span>
        <TrendArrow v-if="currentCfs != null" :gauge-id="gauge.id" size="lg" class="shrink-0 hidden sm:block" />
        <!-- Shared gauge indicator — list view inline -->
        <UTooltip v-if="sharedWith?.length" :text="`Also: ${sharedWith.join(', ')}`" placement="bottom">
          <span class="inline-flex items-center gap-0.5 shrink-0 text-slate-400 dark:text-slate-500">
            <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
          </span>
        </UTooltip>
      </div>
    </div>

    <!-- Sparkline + CFS -->
    <div class="flex items-center gap-2 shrink-0">
      <!-- H2OFlows origin indicator — list view -->
      <svg class="w-3.5 h-3.5 shrink-0 hidden sm:block text-primary-400/40 dark:text-primary-500/30" viewBox="0 0 32 32" fill="none" aria-hidden="true">
        <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/>
        <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.6"/>
      </svg>
      <div class="w-32 shrink-0 hidden sm:block">
        <GaugeSparkline :gauge-id="gauge.id" :flow-status="displayFlowStatus" :flow-band-label="displayFlowBand" :reach-slug="gauge.contextReachSlug ?? gauge.reachSlug" compact @latest-cfs="liveCfs = $event" @live-flow-band="liveFlowBand = $event" />
      </div>
      <span class="text-base font-bold tabular-nums min-w-14 text-right" :style="{ color: cfsColor }">
        {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
        <span class="text-xs font-normal text-neutral-400 dark:text-neutral-500">cfs</span>
      </span>
    </div>

    <button
      class="rounded p-1 text-neutral-300 dark:text-neutral-600 hover:text-red-400 dark:hover:text-red-400 transition-colors shrink-0"
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
    :class="[cardClass, density === 'compact' ? 'p-2.5' : 'p-3']"
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
        <span v-if="density === 'full' && (gauge.contextReachRiverName ?? gauge.riverName)" class="text-xs text-primary-400/70 truncate block">{{ gauge.contextReachRiverName ?? gauge.riverName }}</span>
        <!-- Permit / multi-day micro-icons (full density only) -->
        <span v-if="density === 'full' && (gauge.contextReachPermitRequired || gauge.contextReachMultiDayDays > 1)" class="inline-flex items-center gap-1.5 mt-0.5">
          <UTooltip v-if="gauge.contextReachPermitRequired" text="Permit Required">
            <svg class="w-3 h-3 text-amber-500 dark:text-amber-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M12 7V5a2 2 0 00-2-2H9a2 2 0 00-2 2v6"/><circle cx="12" cy="16" r="1" fill="currentColor" stroke="none"/></svg>
          </UTooltip>
          <UTooltip v-if="gauge.contextReachMultiDayDays > 1" :text="`${gauge.contextReachMultiDayDays}-Day Trip`">
            <svg class="w-3 h-3 text-primary-400 dark:text-primary-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><path d="M16 2v4M8 2v4M3 10h18"/></svg>
          </UTooltip>
        </span>
        <!-- Full: reach full name + gauge source/id as subtitle -->
        <p v-if="!hideReachSubtitle && density === 'full'" class="text-xs truncate mt-0.5">
          <span v-if="contextFullName" class="text-neutral-400">{{ contextFullName }}</span>
          <span v-if="contextFullName && gauge.externalId" class="text-neutral-300 dark:text-neutral-600"> · </span>
          <span class="text-neutral-400 dark:text-neutral-500">{{ gauge.source.toUpperCase() }} {{ gauge.externalId }}</span>
        </p>
      </div>

      <!-- Remove button -->
      <UTooltip text="Remove gauge">
        <button
          class="rounded-lg transition-all duration-150 text-neutral-300 dark:text-neutral-600 hover:text-red-400 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/40"
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
      <span class="font-bold tabular-nums leading-none" :class="density === 'compact' ? 'text-xl' : 'text-2xl'" :style="{ color: cfsColor }">
        {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
      </span>
      <span class="text-xs text-neutral-500">cfs</span>
      <!-- Trending arrow — shown in comfortable + full -->
      <TrendArrow v-if="currentCfs != null && density !== 'compact'" :gauge-id="gauge.id" class="text-lg" />
      <!-- Badge inline with CFS on sm+ for comfortable + full -->
      <span
        v-if="(density === 'comfortable' || density === 'full') && (displayFlowStatus !== 'unknown' || displayFlowBand)"
        :class="['hidden sm:inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', statusBadgeClass]"
      >{{ statusLabel }}</span>
    </div>

    <!-- Flow status badge — mobile only (sm+ gets inline badge above) -->
    <div
      v-if="(displayFlowStatus !== 'unknown' || displayFlowBand) && density !== 'compact'"
      class="flex items-center gap-2 mb-2 sm:hidden"
    >
      <span :class="['inline-flex items-center rounded-md font-medium px-1.5 py-0.5 text-xs', statusBadgeClass]">{{ statusLabel }}</span>
      <span v-if="displayFlowBand === 'high'" class="relative flex h-2 w-2">
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75" />
        <span class="relative inline-flex rounded-full h-2 w-2 bg-sky-400" />
      </span>
    </div>

    <!-- Sparkline — comfortable + full both use compact sparkline -->
    <GaugeSparkline v-if="density === 'comfortable' || density === 'full'" :gauge-id="gauge.id" :flow-status="displayFlowStatus" :flow-band-label="displayFlowBand" :reach-slug="gauge.contextReachSlug ?? gauge.reachSlug" compact class="mb-1" @latest-cfs="liveCfs = $event" @live-flow-band="liveFlowBand = $event" />

    <!-- Diurnal forecast — compact/comfortable: one-liner; full: styled pill box -->
    <p v-if="diurnal.detected && diurnal.forecast && density !== 'full'" class="relative text-[10px] text-indigo-500 dark:text-indigo-400 truncate">
      {{ diurnal.forecast.label }}
    </p>
    <div
      v-if="diurnal.detected && density === 'full'"
      class="mt-1 mb-1 inline-flex items-start gap-1.5 rounded-md border border-indigo-200/70 dark:border-indigo-800/60 bg-indigo-50 dark:bg-indigo-950/40 px-2 py-1 text-[11px] text-indigo-700 dark:text-indigo-300"
    >
      <svg class="w-3.5 h-3.5 shrink-0 mt-px text-indigo-500 dark:text-indigo-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/>
      </svg>
      <div class="min-w-0 leading-tight">
        <span class="font-medium">{{ diurnalPhaseLabel }}</span>
        <template v-if="diurnal.forecast"> · {{ diurnal.forecast.label }}</template>
        <template v-if="diurnal.swingPct != null"><span class="opacity-75"> · {{ diurnal.swingPct }}% swing</span></template>
      </div>
    </div>

    <!-- Health badge — card views -->
    <div v-if="isUnhealthy && density !== 'list'" class="mt-1">
      <span :class="['inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', healthBadgeClass]">
        {{ healthBadgeLabel }}
      </span>
    </div>

    <!-- Last updated — full only, when no subtitle already shows source info -->
    <p v-if="density === 'full' && !contextFullName" class="text-xs text-neutral-400 mt-1 truncate">
      {{ gauge.source.toUpperCase() }} · {{ gauge.externalId }}
    </p>
    <p v-if="density === 'full' && lastUpdatedLabel" class="text-xs text-neutral-400 mt-0.5">{{ lastUpdatedLabel }}</p>

    <!-- H2OFlows origin badge — comfortable + full -->
    <div class="flex items-center gap-1 mt-1.5 text-primary-400/50 dark:text-primary-500/40">
      <svg class="w-3 h-3 shrink-0" viewBox="0 0 32 32" fill="none" aria-hidden="true">
        <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/>
        <path d="M4 22c3-6 6-9 8-9s5 9 8 9 5-9 8-9" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.6"/>
      </svg>
      <span class="text-[10px] font-medium">H2OFlows</span>
    </div>

    <!-- Shared gauge indicator — card view (comfortable + full) -->
    <UTooltip
      v-if="sharedWith?.length && density !== 'compact'"
      :text="`Also: ${sharedWith.join(', ')}`"
      placement="bottom"
    >
      <div class="flex items-center gap-1 mt-2 pt-1.5 border-t border-neutral-100 dark:border-neutral-800 text-slate-400 dark:text-slate-500">
        <svg class="w-3 h-3 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
        <span class="text-[10px] font-medium">Shared gauge</span>
      </div>
    </UTooltip>
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
  sharedWith?: string[]  // other reach names on the dashboard using the same gauge station
}>()
const emit = defineEmits<{ (e: 'open'): void }>()

const { removeAndSync } = useWatchlistSync()
const { bandBadgeClass, bandSolid } = useFlowBandPalette()

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

const statusBadgeClass = computed(() => bandBadgeClass(displayFlowBand.value, displayFlowStatus.value))
const statusLabel      = computed(() => flowBandLabel(displayFlowBand.value, displayFlowStatus.value))
const cfsColor         = computed(() => bandSolid(displayFlowBand.value, displayFlowStatus.value))

// --- Card chrome ------------------------------------------------------------

const cardClass = computed(() => ({
  'border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-900': true,
}))

// --- Last updated -----------------------------------------------------------

const lastUpdatedLabel = computed(() => {
  const ts = props.gauge.lastReadingAt
  if (!ts) return ''
  const ms = Date.now() - new Date(ts).getTime()
  const minutes = Math.floor(ms / 60_000)
  if (minutes < 1)  return 'Updated just now'
  if (minutes < 60) return `Updated ${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24)   return `Updated ${hours}h ago`
  return `Updated ${Math.floor(hours / 24)}d ago`
})

// --- Poll health ------------------------------------------------------------

const isUnhealthy = computed(() =>
  props.gauge.pollHealth === 'stale' || props.gauge.pollHealth === 'unreachable'
)

const healthBadgeLabel = computed(() => {
  if (props.gauge.pollHealth === 'unreachable') return 'Offline'
  if (props.gauge.pollHealth === 'stale')       return 'Stale'
  return ''
})

const healthBadgeClass = computed(() =>
  props.gauge.pollHealth === 'unreachable'
    ? 'bg-red-100 dark:bg-red-950/60 text-red-700 dark:text-red-300'
    : 'bg-amber-100 dark:bg-amber-950/60 text-amber-700 dark:text-amber-300'
)
</script>
