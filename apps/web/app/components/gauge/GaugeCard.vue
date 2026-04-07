<template>
  <div
    class="relative rounded-xl border p-4 transition-all duration-200 cursor-pointer"
    :class="cardClass"
    @click="emit('open')"
  >
    <!-- Gauge name + reach subtitle -->
    <div class="flex items-start justify-between gap-2 mb-3">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-1.5">
          <UTooltip :text="tierTooltip">
            <span class="text-xs shrink-0" :class="tierIconClass">{{ tierIcon }}</span>
          </UTooltip>
          <UTooltip :text="displayName" :delay-duration="500">
            <span class="font-medium text-sm truncate">{{ displayName }}</span>
          </UTooltip>
        </div>
        <p v-if="!hideReachSubtitle" class="text-xs truncate mt-0.5 pl-4">
          <span v-if="gauge.riverName" class="text-blue-400 dark:text-blue-500 font-medium">{{ gauge.riverName }}</span>
          <span v-if="gauge.riverName && gauge.reachName" class="text-gray-300 dark:text-gray-600"> · </span>
          <span v-if="gauge.reachName" class="text-gray-400">{{ gauge.reachName }}</span>
        </p>
      </div>

      <!-- Card actions -->
      <div class="flex items-center gap-1 shrink-0">
        <!-- Run it / Stop recording -->
        <UTooltip :text="watchTooltip">
          <button
            class="rounded-lg p-1.5 transition-all duration-150"
            :class="watchButtonClass"
            :aria-label="watchTooltip"
            @click.stop="handleWatchClick"
          >
            <component :is="watchIcon" class="w-4 h-4" />
          </button>
        </UTooltip>

        <!-- More actions -->
        <UDropdownMenu :items="cardMenuItems" @click.stop>
          <button
            class="rounded-lg p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 transition-all duration-150"
            aria-label="More actions"
            @click.stop
          >
            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <circle cx="4" cy="10" r="1.5" /><circle cx="10" cy="10" r="1.5" /><circle cx="16" cy="10" r="1.5" />
            </svg>
          </button>
        </UDropdownMenu>
      </div>
    </div>

    <!-- Current flow reading -->
    <div class="flex items-end gap-2 mb-2">
      <span class="text-3xl font-bold tabular-nums" :class="cfsClass">
        {{ currentCfs != null ? currentCfs.toLocaleString() : '—' }}
      </span>
      <span class="text-sm text-gray-500 mb-1">cfs</span>
      <TrendArrow v-if="currentCfs != null" :gauge-id="gauge.id" class="mb-1" />
    </div>

    <!-- 12-hour sparkline -->
    <GaugeSparkline :gauge-id="gauge.id" :flow-status="gauge.flowStatus" class="mb-2" />

    <!-- Flow status badge — shown when status is known or a named band is available -->
    <div v-if="gauge.flowStatus !== 'unknown' || gauge.flowBandLabel" class="flex items-center gap-2">
      <UBadge :color="statusColor" variant="subtle" size="sm">{{ statusLabel }}</UBadge>
      <!-- Flood warning pulse -->
      <span v-if="gauge.flowStatus === 'flood'" class="relative flex h-2 w-2">
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75" />
        <span class="relative inline-flex rounded-full h-2 w-2 bg-blue-500" />
      </span>
    </div>

    <!-- Source + external ID (+ featured medal) -->
    <p class="text-xs text-gray-400 mt-2 truncate flex items-center gap-1">
      <UTooltip v-if="gauge.featured" text="Community-verified gauge — trusted data">
        <span class="leading-none" style="filter: drop-shadow(0 0 2px rgba(217,170,0,0.6))">🥇</span>
      </UTooltip>
      {{ gauge.source.toUpperCase() }} · {{ gauge.externalId }}
    </p>
    <p v-if="lastUpdatedLabel" class="text-xs text-gray-400 mt-0.5">{{ lastUpdatedLabel }}</p>

    <!-- GPS permission error -->
    <p v-if="permissionErr && !isActive" class="mt-2 text-xs text-red-500">
      {{ permissionErr }}
    </p>

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

    <!-- Post-trip consent banner — shown briefly after a trip is stopped -->
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
  // When true, suppresses the reach name subtitle (e.g. it's already shown as a section header)
  hideReachSubtitle?: boolean
}>()
const emit = defineEmits<{ (e: 'open'): void }>()

const { startRecording, stopRecording, permissionErr } = useTripRecording()
const { enqueue, flush } = useOfflineQueue()

const store = useWatchlistStore()

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
    // "too_low" → "Too Low", "optimal" → "Optimal", etc.
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
  'text-emerald-500': props.gauge.flowStatus === 'runnable',
  'text-amber-400':   props.gauge.flowStatus === 'caution',
  'text-red-500':     props.gauge.flowStatus === 'low',
  'text-blue-500':    props.gauge.flowStatus === 'flood',
  'text-gray-400':    props.gauge.flowStatus === 'unknown',
}))

// --- Card chrome ------------------------------------------------------------

const cardClass = computed(() => ({
  'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900': !isActive.value,
  'border-emerald-400 dark:border-emerald-500 bg-emerald-50/40 dark:bg-emerald-950/30 shadow-emerald-100 dark:shadow-emerald-900/20 shadow-md':
    isActive.value,
}))

// --- Track button -----------------------------------------------------------

// saved  → "Track it" (play icon) — tap to start a flow tracking session
// active → "Stop" (stop icon)     — tap to end the session
const watchIcon = computed(() =>
  isActive.value ? resolveComponent('IconStop') : resolveComponent('IconPlay')
)
const watchButtonClass = computed(() => ({
  'text-gray-400 hover:text-emerald-500 hover:bg-emerald-50 dark:hover:bg-emerald-950': isSaved.value,
  'text-emerald-600 bg-emerald-100 dark:bg-emerald-900 hover:bg-emerald-200':           isActive.value,
}))
const watchTooltip = computed(() =>
  isActive.value ? 'Stop tracking' : 'Track it — start a flow tracking session'
)

// --- Card menu (⋯) ----------------------------------------------------------

const cardMenuItems = computed(() => [[
  {
    label: isActive.value ? 'Stop tracking' : 'Track it',
    icon: isActive.value ? 'i-heroicons-stop-circle' : 'i-heroicons-play-circle',
    onSelect: () => handleWatchClick(),
  },
], [
  {
    label: 'Remove gauge',
    icon: 'i-heroicons-trash',
    class: 'text-red-500',
    onSelect: () => store.removeGauge(props.gauge.id),
  },
]])

// deviceId — random UUID generated once per install, stored in localStorage.
// Identifies anonymous devices for trip attribution without requiring login.
function getDeviceId(): string {
  const key = 'h2oflow_device_id'
  let id = localStorage.getItem(key)
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem(key, id)
  }
  return id
}

// Post-trip consent — held here until user responds, then enqueued + flushed.
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
    // End trip — stop GPS, collect track, hold for consent before upload.
    const track = stopRecording()
    const trip = store.activeTrip
    store.endTrip()

    if (trip) {
      pendingTrip.value = {
        queuedAt:     new Date().toISOString(),
        gaugeId:      trip.gaugeId,
        reachId:      null,
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
    // Start trip — request GPS permission and begin recording.
    const ok = await startRecording()
    if (ok) {
      store.startTrip(props.gauge.id)
    }
    // If permission denied, permissionErr is set — show it below the card.
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
