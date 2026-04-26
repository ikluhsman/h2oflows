<template>
  <div class="relative rounded-xl overflow-hidden border border-gray-200 dark:border-gray-800">
    <div ref="container" class="w-full bg-gray-100 dark:bg-gray-900" style="height:480px" />

    <!-- Loading overlay -->
    <div v-if="!mapReady" class="absolute inset-0 flex items-center justify-center text-sm text-gray-400 pointer-events-none">
      Loading map…
    </div>

    <!-- Pick-mode crosshair hint -->
    <div v-if="mapReady && pickMode && !comidSelectMode" class="absolute top-2 left-1/2 -translate-x-1/2 z-10 bg-blue-600 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow pointer-events-none">
      Click the map to snap to NHD
    </div>

    <!-- ComID select mode hint -->
    <div v-if="mapReady && comidSelectMode" class="absolute top-2 left-1/2 -translate-x-1/2 z-10 flex gap-1.5 pointer-events-none">
      <span v-if="comidSelectSlot === 'up'" class="bg-green-600 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow">
        Click a flow line to set the reach start
      </span>
      <span v-else-if="comidSelectSlot === 'down'" class="bg-red-600 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow">
        Click a flow line to set the reach end
      </span>
      <span v-else-if="!selectedUpComID" class="bg-green-600 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow">
        Click a flow line to set the reach start
      </span>
      <span v-else-if="!selectedDownComID" class="bg-red-600 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow">
        Click a flow line to set the reach end
      </span>
      <span v-else class="bg-gray-800 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow">
        Both flow lines selected — adjust or continue
      </span>
    </div>

    <!-- Basemap switcher -->
    <div v-if="mapReady" class="absolute bottom-7 left-2 z-10">
      <div class="flex rounded-md shadow overflow-hidden border border-gray-200 dark:border-gray-600 text-xs font-medium">
        <button
          v-for="opt in BASEMAP_OPTIONS" :key="opt.value"
          class="px-2 py-1 transition-colors"
          :class="basemap === opt.value
            ? 'bg-blue-600 text-white'
            : 'bg-white/90 dark:bg-gray-800/90 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700'"
          @click="setBasemap(opt.value)"
        >{{ opt.label }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, watchEffect, onMounted, onUnmounted, nextTick } from 'vue'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

export interface NHDFeatureCollection {
  type: string
  features: any[]
}

export interface AuthoringPin {
  lat: number
  lng: number
  label?: string
}

const props = defineProps<{
  upstreamFlowlines:   NHDFeatureCollection | null
  downstreamFlowlines: NHDFeatureCollection | null
  upstreamGauges:      NHDFeatureCollection | null
  snapLat?:            number | null
  snapLng?:            number | null
  pickMode?:           boolean
  putInPin?:           AuthoringPin | null
  takeOutPin?:         AuthoringPin | null
  // ComID selection mode: click a flowline segment to emit its ComID
  comidSelectMode?:    boolean
  // Which slot the next ComID click fills — drives the prompt pill text.
  comidSelectSlot?:    'up' | 'down' | null
  selectedUpComID?:    string | null
  selectedDownComID?:  string | null
  // Suppresses all auto-fit behaviour — user controls the viewport entirely.
  disableAutoFit?:     boolean
}>()

const emit = defineEmits<{
  pick:           [lat: number, lng: number]
  'comid-select': [comid: string, lat: number, lng: number]
}>()

const BASEMAP_OPTIONS = [
  { value: 'street',    label: 'Street'    },
  { value: 'topo',      label: 'Topo'      },
  { value: 'satellite', label: 'Satellite' },
] as const

const container = ref<HTMLDivElement>()
const mapReady  = ref(false)
const basemap   = ref<'street' | 'topo' | 'satellite'>('street')

let map: maplibregl.Map | null = null
let snapMarker: maplibregl.Marker | null = null
let putInMarker: maplibregl.Marker | null = null
let takeOutMarker: maplibregl.Marker | null = null

// Auto-fit gating — see shouldFit() for the policy.
let prevUpSet = false
let prevDownSet = false
let initialFitDone = false

const BASEMAP_LAYER_IDS = { street: 'street-tiles', topo: 'topo-tiles', satellite: 'esri-tiles' } as const

function setBasemap(val: 'street' | 'topo' | 'satellite') {
  if (!map) return
  basemap.value = val
  for (const [key, id] of Object.entries(BASEMAP_LAYER_IDS)) {
    map.setLayoutProperty(id, 'visibility', key === val ? 'visible' : 'none')
  }
}

function empty(): NHDFeatureCollection {
  return { type: 'FeatureCollection', features: [] }
}

function setLayerData(sourceId: string, fc: NHDFeatureCollection | null) {
  if (!map) return
  const src = map.getSource(sourceId) as maplibregl.GeoJSONSource | undefined
  src?.setData(fc ?? empty())
}

function makePin(color: string, label?: string): HTMLElement {
  const el = document.createElement('div')
  el.style.cssText = `width:14px;height:14px;border-radius:50%;background:${color};border:2.5px solid #fff;box-shadow:0 1px 4px rgba(0,0,0,.45);cursor:default`
  if (label) el.title = label
  return el
}

function updateAuthoringPins() {
  putInMarker?.remove()
  putInMarker = null
  takeOutMarker?.remove()
  takeOutMarker = null
  if (!map) return
  if (props.putInPin) {
    putInMarker = new maplibregl.Marker({ element: makePin('#16a34a', props.putInPin.label ?? 'Put-in') })
      .setLngLat([props.putInPin.lng, props.putInPin.lat])
      .addTo(map)
  }
  if (props.takeOutPin) {
    takeOutMarker = new maplibregl.Marker({ element: makePin('#dc2626', props.takeOutPin.label ?? 'Take-out') })
      .setLngLat([props.takeOutPin.lng, props.takeOutPin.lat])
      .addTo(map)
  }
}

function updateData() {
  setLayerData('nhd-upstream',   props.upstreamFlowlines)
  setLayerData('nhd-downstream', props.downstreamFlowlines)
  setLayerData('nhd-gauges',     props.upstreamGauges)
  updateSnapMarker()
  updateAuthoringPins()
  fitToData()
}

function updateSnapMarker() {
  snapMarker?.remove()
  snapMarker = null
  if (props.snapLat != null && props.snapLng != null && map) {
    const el = document.createElement('div')
    el.style.cssText = 'width:12px;height:12px;border-radius:50%;background:#2563eb;border:2px solid #fff;box-shadow:0 1px 4px rgba(0,0,0,.4)'
    snapMarker = new maplibregl.Marker({ element: el })
      .setLngLat([props.snapLng, props.snapLat])
      .addTo(map)
  }
}

function updateComIDFilters() {
  if (!map || !mapReady.value) return
  const up   = props.selectedUpComID   ?? ''
  const down = props.selectedDownComID ?? ''
  map.setFilter('nhd-upstream-selected',   up   ? ['==', ['get', 'nhdplus_comid'], up]   : ['==', ['literal', true], false])
  map.setFilter('nhd-downstream-selected', down ? ['==', ['get', 'nhdplus_comid'], down] : ['==', ['literal', true], false])
}

// Returns true only on transitions where we actually want to re-frame the map:
// initial load, reset back to no ComIDs, or both ComIDs just became set.
// While exactly one ComID is selected (mid-pick), never auto-fit — that's the
// jump that happens when the downstream mainstem loads after the first pick.
function shouldFit(): boolean {
  const upSet = !!props.selectedUpComID
  const downSet = !!props.selectedDownComID
  const wasUpSet = prevUpSet
  const wasDownSet = prevDownSet
  prevUpSet = upSet
  prevDownSet = downSet

  if (upSet !== downSet) return false

  if (upSet && downSet) {
    return !(wasUpSet && wasDownSet)
  }

  if (!initialFitDone) {
    initialFitDone = true
    return true
  }
  return wasUpSet || wasDownSet
}

function fitToData() {
  if (!map) return
  const allCoords: [number, number][] = []
  for (const fc of [props.upstreamFlowlines, props.downstreamFlowlines]) {
    if (!fc) continue
    for (const f of fc.features) {
      const geom = f.geometry
      if (!geom) continue
      if (geom.type === 'LineString') allCoords.push(...geom.coordinates)
      else if (geom.type === 'MultiLineString') allCoords.push(...geom.coordinates.flat())
    }
  }
  // Include authoring pins so re-pin mode zooms to existing access points
  // even before any flowlines are loaded.
  if (props.putInPin)   allCoords.push([props.putInPin.lng,   props.putInPin.lat])
  if (props.takeOutPin) allCoords.push([props.takeOutPin.lng, props.takeOutPin.lat])
  if (allCoords.length < 2) return
  if (props.disableAutoFit) return
  if (!shouldFit()) return
  const bounds = allCoords.reduce(
    (b, [lng, lat]) => b.extend([lng, lat] as [number, number]),
    new maplibregl.LngLatBounds(allCoords[0], allCoords[0]),
  )
  map.fitBounds(bounds, { padding: 60, maxZoom: 14 })
}

function addLayers() {
  if (!map) return

  // ── Sources ──────────────────────────────────────────────────────────────
  const sources: [string, NHDFeatureCollection | null][] = [
    ['nhd-upstream',   props.upstreamFlowlines],
    ['nhd-downstream', props.downstreamFlowlines],
    ['nhd-gauges',     props.upstreamGauges],
  ]
  for (const [id, data] of sources) {
    map.addSource(id, { type: 'geojson', data: data ?? empty() })
  }

  // ── Upstream flowlines — fat transparent hit-area ─────────────────────
  // Sits beneath the visible line so clicks on a thick area still land on
  // the flowline (the actual rendered line is only 1.5px wide).
  map.addLayer({
    id: 'nhd-upstream-hit',
    type: 'line',
    source: 'nhd-upstream',
    paint: {
      'line-color': '#000',
      'line-width': 14,
      'line-opacity': 0,
    },
  })

  // ── Upstream flowlines — blue ─────────────────────────────────────────
  map.addLayer({
    id: 'nhd-upstream-line',
    type: 'line',
    source: 'nhd-upstream',
    paint: {
      'line-color': '#60a5fa',
      'line-width': 2.5,
      'line-opacity': 0.8,
    },
  })

  // ── Selected upstream ComID — green highlight (on top) ────────────────
  map.addLayer({
    id: 'nhd-upstream-selected',
    type: 'line',
    source: 'nhd-upstream',
    filter: ['==', ['literal', true], false], // nothing highlighted initially
    paint: {
      'line-color': '#16a34a',
      'line-width': 5,
      'line-opacity': 1,
    },
  })

  // ── Selected downstream ComID — red highlight (on top) ───────────────
  map.addLayer({
    id: 'nhd-downstream-selected',
    type: 'line',
    source: 'nhd-upstream', // same UT source — downstream segment also lives here
    filter: ['==', ['literal', true], false],
    paint: {
      'line-color': '#dc2626',
      'line-width': 5,
      'line-opacity': 1,
    },
  })

  // ── Downstream mainstem — fat transparent hit-area ────────────────────
  map.addLayer({
    id: 'nhd-downstream-hit',
    type: 'line',
    source: 'nhd-downstream',
    paint: {
      'line-color': '#000',
      'line-width': 14,
      'line-opacity': 0,
    },
  })

  // ── Downstream mainstem — teal ────────────────────────────────────────
  map.addLayer({
    id: 'nhd-downstream-line',
    type: 'line',
    source: 'nhd-downstream',
    paint: {
      'line-color': '#0d9488',
      'line-width': 3,
      'line-opacity': 0.9,
    },
  })

  // ── Gauge sites — amber circles ───────────────────────────────────────
  map.addLayer({
    id: 'nhd-gauges-circle',
    type: 'circle',
    source: 'nhd-gauges',
    filter: ['==', ['geometry-type'], 'Point'],
    paint: {
      'circle-radius': 6,
      'circle-color': '#f59e0b',
      'circle-stroke-color': '#fff',
      'circle-stroke-width': 1.5,
    },
  })

  // ── Gauge popups ──────────────────────────────────────────────────────
  map.on('click', 'nhd-gauges-circle', (e) => {
    const f = e.features?.[0]
    if (!f || !map) return
    const p = f.properties ?? {}
    const coords = (f.geometry as GeoJSON.Point).coordinates as [number, number]
    new maplibregl.Popup({ closeButton: false, maxWidth: '220px' })
      .setLngLat(coords)
      .setHTML(`<div style="font-size:12px;line-height:1.5">
        <b>${p.name || p.identifier || 'USGS gauge'}</b>
        <div style="color:#6b7280;margin-top:2px">${p.identifier ?? ''}</div>
      </div>`)
      .addTo(map)
  })

  map.on('mouseenter', 'nhd-gauges-circle', () => { if (map) map.getCanvas().style.cursor = 'pointer' })
  map.on('mouseleave', 'nhd-gauges-circle', () => { if (map) map.getCanvas().style.cursor = props.pickMode ? 'crosshair' : '' })

  // ── ComID selection: click on any flowline segment ────────────────────
  // Hit-target layers are transparent + 14px wide so small flowlines stay
  // clickable. Visible layers stay clickable too as a fallback.
  const flowlineClick = (e: maplibregl.MapMouseEvent & { features?: maplibregl.MapGeoJSONFeature[] }) => {
    if (!props.comidSelectMode) return
    const comid = e.features?.[0]?.properties?.nhdplus_comid as string | undefined
    if (comid) {
      e.preventDefault()
      emit('comid-select', comid, e.lngLat.lat, e.lngLat.lng)
    }
  }
  const flowlineHover = () => {
    if (map && props.comidSelectMode) map.getCanvas().style.cursor = 'pointer'
  }
  const flowlineLeave = () => {
    if (map) map.getCanvas().style.cursor = props.pickMode ? 'crosshair' : ''
  }
  for (const id of ['nhd-upstream-hit', 'nhd-upstream-line', 'nhd-downstream-hit', 'nhd-downstream-line']) {
    map.on('click', id, flowlineClick)
    map.on('mouseenter', id, flowlineHover)
    map.on('mouseleave', id, flowlineLeave)
  }
}

function initMap() {
  if (!container.value || map) return

  map = new maplibregl.Map({
    container: container.value,
    style: {
      version: 8,
      glyphs: 'https://demotiles.maplibre.org/font/{fontstack}/{range}.pbf',
      sources: {
        street: {
          type: 'raster',
          tiles: ['https://server.arcgisonline.com/ArcGIS/rest/services/World_Street_Map/MapServer/tile/{z}/{y}/{x}'],
          tileSize: 256,
          attribution: 'Tiles © Esri',
          maxzoom: 18,
        },
        topo: {
          type: 'raster',
          tiles: ['https://server.arcgisonline.com/ArcGIS/rest/services/USA_Topo_Maps/MapServer/tile/{z}/{y}/{x}'],
          tileSize: 256,
          attribution: 'Tiles © Esri — USGS, NPS',
          maxzoom: 15,
        },
        esri: {
          type: 'raster',
          tiles: ['https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}'],
          tileSize: 256,
          attribution: 'Tiles © Esri',
          maxzoom: 18,
        },
      },
      layers: [
        { id: 'street-tiles',    type: 'raster', source: 'street',    layout: { visibility: 'visible' } },
        { id: 'topo-tiles',      type: 'raster', source: 'topo',      layout: { visibility: 'none'    } },
        { id: 'esri-tiles',      type: 'raster', source: 'esri',      layout: { visibility: 'none'    } },
      ],
    },
    center: [-106.0, 39.5],
    zoom: 7,
    attributionControl: false,
    fadeDuration: 0,
  })

  map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-right')
  map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-right')

  map.on('load', () => {
    if (!map) return
    map.resize()
    mapReady.value = true
    addLayers()
    updateComIDFilters()
    if (props.upstreamFlowlines?.features.length || props.downstreamFlowlines?.features.length
        || props.putInPin || props.takeOutPin) {
      updateData()
    }
  })

  map.on('click', (e) => {
    if (props.pickMode && !props.comidSelectMode) emit('pick', e.lngLat.lat, e.lngLat.lng)
  })

  map.on('error', (e) => console.warn('[NHDExplorerMap]', e.error?.message ?? e))
}

watch(() => props.pickMode, (active) => {
  if (!map) return
  map.getCanvas().style.cursor = active ? 'crosshair' : ''
})

watch(() => props.comidSelectMode, () => {
  if (!map) return
  map.getCanvas().style.cursor = props.comidSelectMode ? 'crosshair' : (props.pickMode ? 'crosshair' : '')
})

watch([() => props.selectedUpComID, () => props.selectedDownComID], updateComIDFilters)

watch(
  () => [props.upstreamFlowlines, props.downstreamFlowlines, props.upstreamGauges,
         props.snapLat, props.snapLng, props.putInPin, props.takeOutPin],
  () => { if (mapReady.value) updateData() },
  { deep: true },
)

onMounted(async () => {
  await nextTick()
  // Two rAF passes: first clears the .client.vue / ClientOnly render deferral,
  // second waits for the browser layout so the container has non-zero dimensions.
  await new Promise<void>(r => requestAnimationFrame(() => requestAnimationFrame(r)))
  initMap()
})

// Fallback: if the container ref wasn't ready when onMounted ran (can happen
// when the component is mounted inside a tab v-if that flips after mount),
// watch for it to become available and init then.
watchEffect(() => {
  if (container.value && !map) {
    nextTick(() => initMap())
  }
})

onUnmounted(() => {
  snapMarker?.remove()
  putInMarker?.remove()
  takeOutMarker?.remove()
  map?.remove()
  map = null
  snapMarker = null
  putInMarker = null
  takeOutMarker = null
})
</script>
