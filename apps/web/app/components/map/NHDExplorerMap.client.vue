<template>
  <div class="relative rounded-xl overflow-hidden border border-gray-200 dark:border-gray-800" style="height:480px">
    <div ref="container" class="w-full h-full bg-gray-100 dark:bg-gray-900" />

    <!-- Loading overlay -->
    <div v-if="!mapReady" class="absolute inset-0 flex items-center justify-center text-sm text-gray-400 pointer-events-none">
      Loading map…
    </div>

    <!-- Pick-mode crosshair hint -->
    <div v-if="mapReady && pickMode" class="absolute top-2 left-1/2 -translate-x-1/2 z-10 bg-blue-600 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow pointer-events-none">
      Click the map to snap to NHD
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
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
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
}>()

const emit = defineEmits<{
  pick: [lat: number, lng: number]
}>()

const BASEMAP_OPTIONS = [
  { value: 'street',    label: 'Street'    },
  { value: 'topo',      label: 'Topo'      },
  { value: 'satellite', label: 'Satellite' },
] as const

const container = ref<HTMLDivElement>()
const mapReady  = ref(false)
const basemap   = ref<'street' | 'topo' | 'satellite'>('topo')

let map: maplibregl.Map | null = null
let snapMarker: maplibregl.Marker | null = null
let putInMarker: maplibregl.Marker | null = null
let takeOutMarker: maplibregl.Marker | null = null

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
  if (allCoords.length < 2) return
  const bounds = allCoords.reduce(
    (b, [lng, lat]) => b.extend([lng, lat] as [number, number]),
    new maplibregl.LngLatBounds(allCoords[0], allCoords[0]),
  )
  map.fitBounds(bounds, { padding: 40, maxZoom: 13 })
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

  // ── Upstream flowlines — blue ─────────────────────────────────────────
  map.addLayer({
    id: 'nhd-upstream-line',
    type: 'line',
    source: 'nhd-upstream',
    paint: {
      'line-color': '#93c5fd',
      'line-width': 1.5,
      'line-opacity': 0.7,
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
    const props = f.properties ?? {}
    const coords = (f.geometry as GeoJSON.Point).coordinates as [number, number]
    new maplibregl.Popup({ closeButton: false, maxWidth: '220px' })
      .setLngLat(coords)
      .setHTML(`<div style="font-size:12px;line-height:1.5">
        <b>${props.name || props.identifier || 'USGS gauge'}</b>
        <div style="color:#6b7280;margin-top:2px">${props.identifier ?? ''}</div>
      </div>`)
      .addTo(map)
  })

  map.on('mouseenter', 'nhd-gauges-circle', () => { if (map) map.getCanvas().style.cursor = 'pointer' })
  map.on('mouseleave', 'nhd-gauges-circle', () => { if (map) map.getCanvas().style.cursor = props.pickMode ? 'crosshair' : '' })
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
        { id: 'street-tiles',    type: 'raster', source: 'street',    layout: { visibility: 'none'    } },
        { id: 'topo-tiles',      type: 'raster', source: 'topo',      layout: { visibility: 'visible' } },
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
    mapReady.value = true
    addLayers()
    if (props.upstreamFlowlines?.features.length || props.downstreamFlowlines?.features.length) {
      updateData()
    }
  })

  map.on('click', (e) => {
    if (props.pickMode) emit('pick', e.lngLat.lat, e.lngLat.lng)
  })

  map.on('error', (e) => console.warn('[NHDExplorerMap]', e.error?.message ?? e))
}

watch(() => props.pickMode, (active) => {
  if (!map) return
  map.getCanvas().style.cursor = active ? 'crosshair' : ''
})

watch(
  () => [props.upstreamFlowlines, props.downstreamFlowlines, props.upstreamGauges,
         props.snapLat, props.snapLng, props.putInPin, props.takeOutPin],
  () => { if (mapReady.value) updateData() },
  { deep: true },
)

onMounted(async () => {
  await nextTick()
  await new Promise<void>(r => requestAnimationFrame(() => r()))
  initMap()
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
