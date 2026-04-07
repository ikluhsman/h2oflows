<template>
  <UModal v-model:open="open" :ui="{ width: 'max-w-2xl' }">
    <template #header>
      <div class="flex items-center justify-between gap-3 w-full min-w-0">
        <div class="min-w-0">
          <h2 class="font-semibold truncate">{{ title }}</h2>
          <p class="text-xs text-gray-400 mt-0.5">{{ dateLabel }}</p>
        </div>
        <!-- Stats chips -->
        <div class="flex items-center gap-2 shrink-0 text-xs text-gray-500">
          <span v-if="detail?.duration_min != null">{{ durationLabel }}</span>
          <span v-if="detail?.distance_mi   != null" class="text-gray-300 dark:text-gray-600">·</span>
          <span v-if="detail?.distance_mi   != null">{{ detail.distance_mi.toFixed(1) }} mi</span>
        </div>
      </div>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center h-40 text-gray-400 text-sm">Loading…</div>
      <div v-else-if="detail" class="space-y-4">

        <!-- Track map -->
        <div
          v-if="detail.track"
          ref="mapContainer"
          class="w-full rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-800"
          style="height: 240px;"
        />
        <div v-else class="w-full rounded-lg bg-gray-50 dark:bg-gray-800 flex items-center justify-center text-xs text-gray-400" style="height: 80px;">
          No GPS track recorded
        </div>

        <!-- Flow at time -->
        <div v-if="detail.start_cfs != null" class="flex items-center gap-3 text-sm">
          <span class="text-gray-500">Flow at put-in</span>
          <span class="font-semibold tabular-nums">{{ detail.start_cfs.toLocaleString() }} cfs</span>
          <span v-if="detail.end_cfs != null" class="text-gray-400">→ {{ detail.end_cfs.toLocaleString() }} cfs at take-out</span>
        </div>

        <!-- GPS points count -->
        <p v-if="detail.point_count > 0" class="text-xs text-gray-400">
          {{ detail.point_count.toLocaleString() }} GPS points · {{ detail.track ? 'track available' : 'processing…' }}
        </p>

        <!-- Notes editor -->
        <div class="border-t border-gray-100 dark:border-gray-800 pt-4">
          <label class="block text-xs font-medium text-gray-500 mb-1.5">Notes</label>
          <textarea
            v-model="notesEdit"
            rows="3"
            placeholder="Add notes about this trip…"
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm px-3 py-2 text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
          />
          <div class="mt-2 flex justify-end gap-2">
            <UButton
              v-if="notesEdit !== (detail.notes ?? '')"
              size="xs" color="primary" variant="soft"
              :loading="saving"
              @click="saveNotes"
            >Save notes</UButton>
          </div>
        </div>

        <!-- Share consent toggle -->
        <div class="border-t border-gray-100 dark:border-gray-800 pt-4 flex items-start justify-between gap-4">
          <div class="min-w-0">
            <p class="text-sm font-medium">Share anonymously</p>
            <p class="text-xs text-gray-400 mt-0.5 leading-relaxed">
              Help improve reach data by sharing this trip's GPS track and flow readings with the H2OFlows community. No personal information is included.
            </p>
          </div>
          <button
            class="shrink-0 relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none"
            :class="consentValue ? 'bg-blue-600' : 'bg-gray-200 dark:bg-gray-700'"
            @click="toggleConsent"
          >
            <span
              class="inline-block h-4 w-4 rounded-full bg-white shadow transform transition-transform"
              :class="consentValue ? 'translate-x-6' : 'translate-x-1'"
            />
          </button>
        </div>

        <!-- Reach link -->
        <div v-if="detail.reach_slug" class="border-t border-gray-100 dark:border-gray-800 pt-4">
          <NuxtLink
            :to="`/reaches/${detail.reach_slug}`"
            class="flex items-center justify-between gap-2 rounded-lg px-3 py-2 bg-gray-50 dark:bg-gray-900 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            @click="open = false"
          >
            <span class="text-sm font-medium">{{ detail.reach_name }}</span>
            <svg class="w-3.5 h-3.5 text-gray-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18l6-6-6-6"/></svg>
          </NuxtLink>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useTrips, type TripDetail } from '~/composables/useTrips'

const props = defineProps<{ tripId: string | null }>()
const open  = defineModel<boolean>('open', { default: false })

const { getTrip, patchTrip } = useTrips()

const detail    = ref<TripDetail | null>(null)
const loading   = ref(false)
const saving    = ref(false)
const notesEdit = ref('')
const mapContainer = ref<HTMLDivElement | null>(null)

let mapInstance: any = null

// Load detail whenever tripId changes and modal opens
watch([() => props.tripId, open], async ([id, isOpen]) => {
  if (!isOpen || !id) return
  loading.value = true
  detail.value  = null
  try {
    detail.value  = await getTrip(id)
    notesEdit.value = detail.value.notes ?? ''
    await nextTick()
    if (detail.value.track) renderMap(detail.value.track)
  } catch { /* non-fatal */ }
  finally { loading.value = false }
}, { immediate: false })

// Clean up map on close
watch(open, isOpen => {
  if (!isOpen) {
    mapInstance?.remove()
    mapInstance = null
    detail.value = null
  }
})

// ── Map rendering ─────────────────────────────────────────────────────────────

async function renderMap(track: { type: string; coordinates: [number, number][] }) {
  await nextTick()
  if (!mapContainer.value) return

  const maplibregl = (await import('maplibre-gl')).default
  await import('maplibre-gl/dist/maplibre-gl.css')

  const coords = track.coordinates
  if (coords.length < 2) return

  const lngs = coords.map(c => c[0])
  const lats  = coords.map(c => c[1])
  const bounds: [[number, number], [number, number]] = [
    [Math.min(...lngs), Math.min(...lats)],
    [Math.max(...lngs), Math.max(...lats)],
  ]

  mapInstance = new maplibregl.Map({
    container:        mapContainer.value,
    style: {
      version: 8,
      sources: {
        base: {
          type: 'raster',
          tiles: ['https://server.arcgisonline.com/ArcGIS/rest/services/World_Topo_Map/MapServer/tile/{z}/{y}/{x}'],
          tileSize: 256,
          maxzoom: 18,
        },
      },
      layers: [{ id: 'base', type: 'raster', source: 'base' }],
    },
    bounds,
    fitBoundsOptions: { padding: 40 },
    attributionControl: false,
    fadeDuration: 0,
  })

  mapInstance.on('load', () => {
    mapInstance.addSource('trip-track', {
      type: 'geojson',
      data: { type: 'Feature', geometry: track, properties: {} },
    })
    mapInstance.addLayer({
      id: 'track-glow', type: 'line', source: 'trip-track',
      paint: { 'line-color': '#6366f1', 'line-width': 8, 'line-opacity': 0.18, 'line-blur': 4 },
    })
    mapInstance.addLayer({
      id: 'track-line', type: 'line', source: 'trip-track',
      paint: { 'line-color': '#6366f1', 'line-width': 3, 'line-opacity': 0.9 },
    })
    // Start + end markers
    const startEl = document.createElement('div')
    startEl.style.cssText = 'width:10px;height:10px;border-radius:50%;background:#22c55e;border:2px solid white;box-shadow:0 1px 3px rgba(0,0,0,0.4)'
    new maplibregl.Marker({ element: startEl }).setLngLat(coords[0]).addTo(mapInstance)
    const endEl = document.createElement('div')
    endEl.style.cssText = 'width:10px;height:10px;border-radius:50%;background:#ef4444;border:2px solid white;box-shadow:0 1px 3px rgba(0,0,0,0.4)'
    new maplibregl.Marker({ element: endEl }).setLngLat(coords[coords.length - 1]).addTo(mapInstance)
  })
}

// ── Computed ──────────────────────────────────────────────────────────────────

const title = computed(() =>
  detail.value?.reach_name || detail.value?.gauge_name || 'Trip details'
)

const dateLabel = computed(() => {
  if (!detail.value) return ''
  const d = new Date(detail.value.started_at)
  return d.toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' })
})

const durationLabel = computed(() => {
  const m = detail.value?.duration_min
  if (m == null) return ''
  if (m < 60) return `${m}m`
  return `${Math.floor(m / 60)}h ${m % 60}m`
})

const consentValue = computed(() => detail.value?.share_consent ?? false)

// ── Actions ───────────────────────────────────────────────────────────────────

async function saveNotes() {
  if (!detail.value) return
  saving.value = true
  try {
    await patchTrip(detail.value.id, { notes: notesEdit.value })
    detail.value.notes = notesEdit.value
  } catch { /* non-fatal */ }
  finally { saving.value = false }
}

async function toggleConsent() {
  if (!detail.value) return
  const next = !consentValue.value
  try {
    await patchTrip(detail.value.id, { share_consent: next })
    detail.value.share_consent = next
  } catch { /* non-fatal */ }
}
</script>
