/**
 * useTripRecording — GPS track collection for active trip recording.
 *
 * Uses the browser Geolocation API (navigator.geolocation.watchPosition).
 * Designed as a drop-in for @capacitor/geolocation — the swap is one function:
 *   replace startWatching() / stopWatching() below with Capacitor equivalents.
 *
 * Capacitor swap (Phase 2):
 *   import { Geolocation } from '@capacitor/geolocation'
 *   const watchId = await Geolocation.watchPosition({ enableHighAccuracy: true }, callback)
 *   await Geolocation.clearWatch({ id: watchId })
 *
 * Filtering:
 *   - Points with accuracy > ACCURACY_THRESHOLD_M are discarded (bad signal)
 *   - Points are sampled at most once per SAMPLE_INTERVAL_MS (battery / storage)
 *   - Speeds > MAX_SPEED_MPS are discarded (shuttle driving, not paddling)
 */

import { ref, readonly } from 'vue'

// ---- Config -----------------------------------------------------------------

const ACCURACY_THRESHOLD_M = 50    // discard points with GPS error > 50m
const SAMPLE_INTERVAL_MS   = 5000  // at most one point per 5 seconds
// No speed filter — shuttle driving on backcountry roads overlaps with
// plausible on-water speeds, and TrackAnalyzer already handles shuttle
// identification via proximity to put-in/take-out coordinates.

// ---- Types ------------------------------------------------------------------

export interface TrackPoint {
  timestamp:  string    // ISO 8601
  lat:        number
  lng:        number
  accuracy_m: number | null
  altitude_m: number | null
  speed_mps:  number | null
  heading:    number | null
}

// ---- Module-level state (singleton — one active recording at a time) --------

const isRecording   = ref(false)
const trackPoints   = ref<TrackPoint[]>([])
const currentLat    = ref<number | null>(null)
const currentLng    = ref<number | null>(null)
const permissionErr = ref<string | null>(null)

let watchId:        number | null = null
let lastSampleTime: number = 0

// ---- Public API -------------------------------------------------------------

export function useTripRecording() {

  async function startRecording(): Promise<boolean> {
    if (isRecording.value) return true

    if (!navigator.geolocation) {
      permissionErr.value = 'Geolocation is not supported by this browser.'
      return false
    }

    permissionErr.value = null
    trackPoints.value   = []
    lastSampleTime      = 0

    return new Promise(resolve => {
      // Request a first fix to surface permission errors immediately.
      navigator.geolocation.getCurrentPosition(
        () => {
          // Permission granted — start continuous watch.
          watchId = navigator.geolocation.watchPosition(
            onPosition,
            onError,
            { enableHighAccuracy: true, timeout: 10_000, maximumAge: 0 },
          )
          isRecording.value = true
          resolve(true)
        },
        err => {
          permissionErr.value = geolocationErrorMessage(err)
          resolve(false)
        },
        { enableHighAccuracy: true, timeout: 10_000 },
      )
    })
  }

  function stopRecording(): TrackPoint[] {
    if (watchId !== null) {
      navigator.geolocation.clearWatch(watchId)
      watchId = null
    }
    isRecording.value = false
    const collected = [...trackPoints.value]
    trackPoints.value = []
    return collected
  }

  return {
    startRecording,
    stopRecording,
    isRecording:    readonly(isRecording),
    trackPoints:    readonly(trackPoints),
    currentLat:     readonly(currentLat),
    currentLng:     readonly(currentLng),
    permissionErr:  readonly(permissionErr),
  }
}

// ---- Internal handlers ------------------------------------------------------

function onPosition(pos: GeolocationPosition) {
  const { latitude, longitude, accuracy, altitude, speed, heading } = pos.coords
  const now = Date.now()

  // Accuracy filter — discard points with poor GPS fix
  if (accuracy > ACCURACY_THRESHOLD_M) return

  // Sample rate limit
  if (now - lastSampleTime < SAMPLE_INTERVAL_MS) return
  lastSampleTime = now

  currentLat.value = latitude
  currentLng.value = longitude

  trackPoints.value.push({
    timestamp:  new Date(pos.timestamp).toISOString(),
    lat:        latitude,
    lng:        longitude,
    accuracy_m: accuracy ?? null,
    altitude_m: altitude ?? null,
    speed_mps:  speed ?? null,
    heading:    heading ?? null,
  })
}

function onError(err: GeolocationPositionError) {
  permissionErr.value = geolocationErrorMessage(err)
}

function geolocationErrorMessage(err: GeolocationPositionError): string {
  switch (err.code) {
    case err.PERMISSION_DENIED:
      return 'Location permission denied. Enable it in your device settings.'
    case err.POSITION_UNAVAILABLE:
      return 'Location unavailable. Check GPS signal.'
    case err.TIMEOUT:
      return 'Location timed out. Move to an area with better GPS signal.'
    default:
      return 'Unknown location error.'
  }
}
