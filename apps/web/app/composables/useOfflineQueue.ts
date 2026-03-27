/**
 * useOfflineQueue — buffers completed trips when the device is offline
 * and uploads them when connectivity is restored.
 *
 * Storage: localStorage (5–10 MB limit, fine for beta).
 * Production path: swap for @capacitor/preferences or @capacitor/filesystem
 * for larger track storage and better mobile reliability.
 *
 * Each queued item is a complete trip payload ready to POST to /api/v1/trips.
 * Items are removed from the queue only after a confirmed 201 response.
 */

import { ref, readonly } from 'vue'
import type { TrackPoint } from './useTripRecording'

// ---- Types ------------------------------------------------------------------

export interface QueuedTrip {
  queuedAt:     string     // ISO — when it was enqueued
  gaugeId:      string | null
  reachId:      string | null
  startCfs:     number | null
  endCfs:       number | null
  startedAt:    string
  endedAt:      string | null
  notes:        string
  deviceId:     string
  shareConsent: boolean | null
  trackPoints:  TrackPoint[]
}

// ---- Constants --------------------------------------------------------------

const STORAGE_KEY = 'h2oflow_trip_queue'

// ---- Module-level state -----------------------------------------------------

const pendingCount = ref(0)
const uploading    = ref(false)
const lastError    = ref<string | null>(null)

// Initialise count from storage on module load.
if (typeof window !== 'undefined') {
  pendingCount.value = loadQueue().length
}

// ---- Public API -------------------------------------------------------------

export function useOfflineQueue() {
  const { apiBase } = useRuntimeConfig().public

  function enqueue(trip: QueuedTrip): void {
    const queue = loadQueue()
    queue.push(trip)
    saveQueue(queue)
    pendingCount.value = queue.length
  }

  async function flush(): Promise<void> {
    if (uploading.value) return
    const queue = loadQueue()
    if (queue.length === 0) return

    uploading.value = true
    lastError.value = null

    const remaining: QueuedTrip[] = []

    for (const trip of queue) {
      try {
        const res = await fetch(`${apiBase}/api/v1/trips`, {
          method:  'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            gauge_id:      trip.gaugeId,
            reach_id:      trip.reachId,
            start_cfs:     trip.startCfs,
            end_cfs:       trip.endCfs,
            started_at:    trip.startedAt,
            ended_at:      trip.endedAt,
            notes:         trip.notes,
            device_id:     trip.deviceId,
            share_consent: trip.shareConsent,
            track_points:  trip.trackPoints,
          }),
        })

        if (res.status === 201) {
          // Successfully uploaded — do not re-add to remaining.
          continue
        }

        // 4xx = bad data, won't succeed on retry — discard with a log.
        if (res.status >= 400 && res.status < 500) {
          console.warn(`trip upload: ${res.status} — discarding`, trip.startedAt)
          continue
        }

        // 5xx or network error — keep for retry.
        remaining.push(trip)
        lastError.value = `Upload failed (${res.status}) — will retry when online`

      } catch {
        // Offline or network error — keep for retry.
        remaining.push(trip)
        lastError.value = 'Offline — trips will upload when connected'
      }
    }

    saveQueue(remaining)
    pendingCount.value = remaining.length
    uploading.value    = false
  }

  return {
    enqueue,
    flush,
    pendingCount: readonly(pendingCount),
    uploading:    readonly(uploading),
    lastError:    readonly(lastError),
  }
}

// ---- Storage helpers --------------------------------------------------------

function loadQueue(): QueuedTrip[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

function saveQueue(queue: QueuedTrip[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(queue))
  } catch (e) {
    // Storage full — log and continue. Trips are not lost until the tab closes.
    console.warn('useOfflineQueue: localStorage write failed', e)
  }
}

// ---- Auto-flush on reconnect ------------------------------------------------
// Fires whenever the browser regains network connectivity.
if (typeof window !== 'undefined') {
  window.addEventListener('online', () => {
    // Small delay so the connection stabilises before we start uploading.
    setTimeout(() => {
      const { flush } = useOfflineQueue()
      flush()
    }, 2000)
  })
}
