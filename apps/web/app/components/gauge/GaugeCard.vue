<template>
  <!-- ─── LIST row ─────────────────────────────────────────────────────── -->
  <div
    v-if="density === 'list'"
    class="flex items-center gap-3 px-3 py-2 rounded-lg border transition-all duration-200 cursor-pointer"
    :class="cardClass"
    @click="emit('open')"
  >
    <UTooltip :text="tierTooltip">
      <span class="text-xs shrink-0" :class="tierIconClass">{{ tierIcon }}</span>
    </UTooltip>

    <div class="min-w-0 flex-1">
      <span class="text-sm font-medium truncate block">{{ displayName }}</span>
      <span v-if="gauge.riverName" class="text-xs text-blue-400 truncate block">{{ gauge.riverName }}</span>
    </div>

    <span class="text-base font-bold tabular-nums shrink-0" :class="cfsClass">
      {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
      <span class="text-xs font-normal text-gray-400 dark:text-gray-500">cfs</span>
    </span>

    <div class="w-16 shrink-0 hidden sm:block">
      <GaugeSparkline :gauge-id="gauge.id" :flow-status="gauge.flowStatus" compact />
    </div>

    <UBadge v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel" :color="statusColor" variant="subtle" size="sm" class="shrink-0 hidden sm:flex">
      {{ statusLabel }}
    </UBadge>

    <div class="flex items-center gap-0.5 shrink-0">
      <UTooltip :text="watchTooltip">
        <button
          class="rounded p-1 transition-all duration-150"
          :class="watchButtonClass"
          :aria-label="watchTooltip"
          @click.stop="handleWatchClick"
        >
          <component :is="watchIcon" class="w-3.5 h-3.5" />
        </button>
      </UTooltip>
      <button
        class="rounded p-1 text-gray-300 dark:text-gray-600 hover:text-red-400 dark:hover:text-red-400 transition-colors"
        aria-label="Remove gauge"
        @click.stop="removeAndSync(gauge.id)"
      >
        <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm4 0a1 1 0 112 0v6a1 1 0 11-2 0V8z" clip-rule="evenodd"/>
        </svg>
      </button>
    </div>
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
        <div class="flex items-center gap-1.5">
          <UTooltip :text="tierTooltip">
            <span class="text-xs shrink-0" :class="tierIconClass">{{ tierIcon }}</span>
          </UTooltip>
          <UTooltip :text="displayName" :delay-duration="500">
            <span class="font-medium truncate" :class="density === 'compact' ? 'text-xs' : 'text-sm'">{{ displayName }}</span>
          </UTooltip>
        </div>
        <p v-if="!hideReachSubtitle && density === 'full'" class="text-xs truncate mt-0.5 pl-4">
          <span v-if="gauge.riverName" class="text-blue-400 dark:text-blue-500 font-medium">{{ gauge.riverName }}</span>
          <span v-if="gauge.riverName && gauge.reachName" class="text-gray-300 dark:text-gray-600"> · </span>
          <span v-if="gauge.reachName" class="text-gray-400">{{ gauge.reachName }}</span>
        </p>
      </div>

      <!-- Card actions: Record/Stop + Remove -->
      <div class="flex items-center gap-1 shrink-0">
        <UTooltip v-if="density === 'full'" :text="watchTooltip">
          <button
            class="rounded-lg p-1.5 transition-all duration-150"
            :class="watchButtonClass"
            :aria-label="watchTooltip"
            @click.stop="handleWatchClick"
          >
            <component :is="watchIcon" class="w-4 h-4" />
          </button>
        </UTooltip>
        <UTooltip text="Remove gauge">
          <button
            class="rounded-lg transition-all duration-150 text-gray-300 dark:text-gray-600 hover:text-red-400 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/40"
            :class="density === 'compact' ? 'p-1' : 'p-1.5'"
            aria-label="Remove gauge"
            @click.stop="removeAndSync(gauge.id)"
          >
            <svg :class="density === 'compact' ? 'w-3 h-3' : 'w-4 h-4'" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm4 0a1 1 0 112 0v6a1 1 0 11-2 0V8z" clip-rule="evenodd"/>
            </svg>
          </button>
        </UTooltip>
      </div>
    </div>

    <!-- Current flow reading + badge (compact/comfortable: badge right of value on sm+) -->
    <div class="relative flex items-baseline gap-2 flex-wrap" :class="density === 'compact' ? 'mb-6' : 'mb-1.5'">
      <span class="font-bold tabular-nums leading-none" :class="[cfsClass, density === 'compact' ? 'text-xl' : density === 'comfortable' ? 'text-2xl' : 'text-3xl']">
        {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
      </span>
      <span class="text-xs text-gray-500">cfs</span>
      <TrendArrow v-if="currentCfs != null && density === 'full'" :gauge-id="gauge.id" />
      <!-- Compact/comfortable: badge inline with CFS on sm+ -->
      <UBadge
        v-if="density !== 'full' && (gauge.flowStatus !== 'unknown' || gauge.flowBandLabel)"
        :color="statusColor" variant="subtle"
        :size="density === 'compact' ? 'xs' : 'sm'"
        class="hidden sm:inline-flex"
      >{{ statusLabel }}</UBadge>
    </div>

    <!-- Flow status badge — full: always shown; compact/comfortable: mobile only -->
    <div v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel" class="flex items-center gap-2 mb-2" :class="density === 'full' ? '' : 'sm:hidden'">
      <UBadge :color="statusColor" variant="subtle" :size="density === 'full' ? 'md' : 'sm'">{{ statusLabel }}</UBadge>
      <span v-if="gauge.flowStatus === 'flood'" class="relative flex h-2 w-2">
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75" />
        <span class="relative inline-flex rounded-full h-2 w-2 bg-blue-400" />
      </span>
    </div>

    <!-- Sparkline — comfortable gets compact sparkline; full gets full sparkline -->
    <GaugeSparkline v-if="density === 'comfortable'" :gauge-id="gauge.id" :flow-status="gauge.flowStatus" compact class="mb-1" />
    <GaugeSparkline v-else-if="density === 'full'" :gauge-id="gauge.id" :flow-status="gauge.flowStatus" class="mb-2" />

    <!-- Source + external ID — full only -->
    <p v-if="density === 'full'" class="text-xs text-gray-400 mt-2 truncate flex items-center gap-1">
      <UTooltip v-if="gauge.featured" text="Community-verified gauge — trusted data">
        <span class="leading-none" style="filter: drop-shadow(0 0 2px rgba(217,170,0,0.6))">🥇</span>
      </UTooltip>
      {{ gauge.source.toUpperCase() }} · {{ gauge.externalId }}
    </p>
    <p v-if="density === 'full' && lastUpdatedLabel" class="text-xs text-gray-400 mt-0.5">{{ lastUpdatedLabel }}</p>

    <!-- GPS permission error -->
    <p v-if="permissionErr && !isActive" class="mt-2 text-xs text-red-400">{{ permissionErr }}</p>

    <!-- Active trip indicator -->
    <div
      v-if="isActive"
      class="mt-3 flex items-center gap-1.5 text-xs font-medium text-emerald-600 dark:text-emerald-400"
    >
      <span class="relative flex h-2 w-2">
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
        <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
      </span>
      Recording trip · {{ activeDuration }}
    </div>

    <!-- Post-trip consent banner -->
    <div
      v-if="showConsentBanner"
      class="mt-3 rounded-lg border border-blue-100 dark:border-blue-900 bg-blue-50 dark:bg-blue-950/40 p-3 space-y-2"
      @click.stop
    >
      <p class="text-xs text-blue-800 dark:text-blue-200 leading-relaxed">
        Share this trip anonymously to help improve reach data?
      </p>
      <div class="flex gap-2">
        <button
          class="flex-1 text-xs font-medium py-1.5 rounded-md bg-blue-600 text-white hover:bg-blue-700 transition-colors"
          @click.stop="resolveConsent(true)"
        >Share</button>
        <button
          class="flex-1 text-xs font-medium py-1.5 rounded-md border border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
          @click.stop="resolveConsent(false)"
        >Keep private</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { QueuedTrip } from '~/composables/useOfflineQueue'
import type { WatchedGauge } from '~/stores/watchlist'
import { useWatchlistStore } from '~/stores/watchlist'
import { useTripRecording } from '~/composables/useTripRecording'
import { useOfflineQueue } from '~/composables/useOfflineQueue'

const props = defineProps<{
  gauge: WatchedGauge
  hideReachSubtitle?: boolean
  density?: 'compact' | 'comfortable' | 'full' | 'list'
}>()
const emit = defineEmits<{ (e: 'open'): void }>()

const { startRecording, stopRecording, permissionErr } = useTripRecording()
const { enqueue, flush } = useOfflineQueue()

const store = useWatchlistStore()
const { removeAndSync } = useWatchlistSync()

const isActive = computed(() => props.gauge.watchState === 'active')
const isSaved  = computed(() => props.gauge.watchState === 'saved')
const currentCfs = computed(() => props.gauge.currentCfs)

// --- Display name -----------------------------------------------------------

const displayName = computed(() =>
  props.gauge.name ?? props.gauge.externalId
)

// --- Poll tier ---------------------------------------------------------------

const tierIcon = computed(() => {
  switch (props.gauge.pollTier) {
    case 'trusted': return '✓'
    case 'demand':  return '◉'
    default:        return '○'
  }
})
const tierIconClass = computed(() => ({
  'text-emerald-500': props.gauge.pollTier === 'trusted',
  'text-blue-400':    props.gauge.pollTier === 'demand',
  'text-gray-400':    props.gauge.pollTier === 'cold',
}))
const tierTooltip = computed(() => {
  switch (props.gauge.pollTier) {
    case 'trusted': return 'Trusted gauge — community-verified, always live'
    case 'demand':  return 'Active gauge — live while being watched'
    default:        return 'Cold gauge — data fetched on request'
  }
})

// --- Flow status ------------------------------------------------------------

const statusColor = computed(() => {
  switch (props.gauge.flowStatus) {
    case 'runnable': return 'success'
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
const cfsClass = computed(() => ({
  'text-emerald-400 dark:text-emerald-500': props.gauge.flowStatus === 'runnable',
  'text-amber-400':                         props.gauge.flowStatus === 'caution',
  'text-red-400':                           props.gauge.flowStatus === 'low',
  'text-blue-400 dark:text-blue-500':       props.gauge.flowStatus === 'flood',
  'text-gray-400':                          props.gauge.flowStatus === 'unknown',
}))

// --- Card chrome ------------------------------------------------------------

const cardClass = computed(() => ({
  'border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900': !isActive.value,
  'border-emerald-400 dark:border-emerald-500 bg-emerald-50/40 dark:bg-emerald-950/30 shadow-emerald-100 dark:shadow-emerald-900/20 shadow-md':
    isActive.value,
}))

// --- Track button -----------------------------------------------------------

const watchIcon = computed(() =>
  isActive.value ? resolveComponent('IconStop') : resolveComponent('IconRecord')
)
const watchButtonClass = computed(() => ({
  'text-gray-400 hover:text-emerald-500 hover:bg-emerald-50 dark:hover:bg-emerald-950': isSaved.value,
  'text-emerald-600 bg-emerald-100 dark:bg-emerald-900 hover:bg-emerald-200':           isActive.value,
}))
const watchTooltip = computed(() =>
  isActive.value ? 'Stop tracking' : 'Track it — start a flow tracking session'
)

// deviceId — random UUID generated once per install, stored in localStorage.
function getDeviceId(): string {
  const key = 'h2oflow_device_id'
  let id = localStorage.getItem(key)
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem(key, id)
  }
  return id
}

const showConsentBanner = ref(false)
const pendingTrip = ref<QueuedTrip | null>(null)

function resolveConsent(share: boolean) {
  if (!pendingTrip.value) return
  pendingTrip.value.shareConsent = share
  enqueue(pendingTrip.value)
  flush()
  pendingTrip.value  = null
  showConsentBanner.value = false
}

async function handleWatchClick() {
  if (isActive.value) {
    const track = stopRecording()
    const trip = store.activeTrip
    store.endTrip()
    if (trip) {
      pendingTrip.value = {
        queuedAt:     new Date().toISOString(),
        gaugeId:      trip.gaugeId,
        reachId:      props.gauge.reachId,
        startCfs:     trip.startCfs,
        endCfs:       props.gauge.currentCfs ?? null,
        startedAt:    trip.startedAt,
        endedAt:      new Date().toISOString(),
        notes:        '',
        deviceId:     getDeviceId(),
        shareConsent: null,
        trackPoints:  track,
      }
      showConsentBanner.value = true
    }
  } else {
    const ok = await startRecording()
    if (ok) store.startTrip(props.gauge.id)
  }
}

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

// --- Active trip duration ---------------------------------------------------

const activeDuration = computed(() => {
  if (!props.gauge.activeSince) return ''
  const ms = Date.now() - new Date(props.gauge.activeSince).getTime()
  const h = Math.floor(ms / 3_600_000)
  const m = Math.floor((ms % 3_600_000) / 60_000)
  return h > 0 ? `${h}h ${m}m` : `${m}m`
})
</script>
