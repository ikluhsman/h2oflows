<template>
  <div class="relative w-full rounded-xl overflow-hidden bg-gray-100 dark:bg-gray-900" style="height: 480px;">
    <div ref="container" class="w-full h-full" />

    <div v-if="!mapReady" class="absolute inset-0 flex items-center justify-center text-gray-400 text-sm pointer-events-none">
      Loading map…
    </div>

    <div v-if="mapReady && reaches.length === 0" class="absolute inset-0 flex items-center justify-center pointer-events-none">
      <p class="text-sm text-gray-400 bg-white/80 dark:bg-gray-900/80 rounded-lg px-4 py-2">
        No centerlines available for this basin yet.
      </p>
    </div>

    <div v-if="mapReady" class="absolute bottom-7 left-2 z-10 flex rounded-md shadow overflow-hidden border border-gray-200 dark:border-gray-600 text-xs font-medium">
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
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

export interface BasinReach {
  slug: string
  name: string
  river_name: string | null
  common_name: string | null
  river_order: number | null
  class_min: number | null
  class_max: number | null
  anchor_comid: string | null
  start_comid: string | null
  end_comid: string | null
  centerline: any
  start_point: [number, number] | null
  end_point: [number, number] | null
  flow_status: string
}

export interface BasinNetwork {
  tributaries: { type: string; features: any[] }
  mainstem:    { type: string; features: any[] }
}

const EMPTY_FC = { type: 'FeatureCollection', features: [] }

const props = defineProps<{
  reaches: BasinReach[]
  network?: BasinNetwork | null
}>()
const emit  = defineEmits<{ (e: 'select', slug: string): void }>()

const container = ref<HTMLDivElement>()
const mapReady  = ref(false)
const basemap   = ref<'street' | 'topo' | 'satellite'>('street')

const BASEMAP_OPTIONS = [
  { value: 'street',    label: 'Street'    },
  { value: 'topo',      label: 'Topo'      },
  { value: 'satellite', label: 'Satellite' },
] as const

let map: maplibregl.Map | null = null

const reachTooltip = new maplibregl.Popup({
  closeButton: false, closeOnClick: false, offset: [0, -8],
  className: 'basin-map-tooltip',
})

function setBasemap(value: 'street' | 'topo' | 'satellite') {
  if (!map) return
  basemap.value = value
  map.setLayoutProperty('street-tiles', 'visibility', value === 'street'    ? 'visible' : 'none')
  map.setLayoutProperty('topo-tiles',   'visibility', value === 'topo'      ? 'visible' : 'none')
  map.setLayoutProperty('esri-tiles',   'visibility', value === 'satellite' ? 'visible' : 'none')
}

function classColorExpr(): any {
  return ['step', ['coalesce', ['get', 'class_max'], 0],
    '#6b7280',        // gray  (no class)
    0.5, '#16a34a',   // green (I–II)
    3.0, '#3b82f6',   // blue  (III)
    4.0, '#111827',   // black (IV)
    5.0, '#dc2626',   // red   (V+)
  ]
}

function allCoordsOf(geom: any): [number, number][] {
  if (!geom?.coordinates) return []
  return geom.type === 'LineString'
    ? geom.coordinates
    : (geom.coordinates as [number, number][][]).flat()
}

function buildReachFC(): GeoJSON.FeatureCollection {
  return {
    type: 'FeatureCollection',
    features: props.reaches
      .filter(r => r.centerline)
      .map(r => ({
        type: 'Feature' as const,
        geometry: r.centerline,
        properties: {
          slug:       r.slug,
          name:       r.name,
          river_name: r.river_name,
          common_name: r.common_name,
          class_min:  r.class_min,
          class_max:  r.class_max,
          flow_status: r.flow_status,
        },
      })),
  }
}

function buildEndpointsFC(): GeoJSON.FeatureCollection {
  const features: GeoJSON.Feature[] = []
  for (const r of props.reaches) {
    if (r.start_point) {
      features.push({
        type: 'Feature',
        geometry: { type: 'Point', coordinates: r.start_point },
        properties: { slug: r.slug, kind: 'start', name: r.name },
      })
    }
    if (r.end_point) {
      features.push({
        type: 'Feature',
        geometry: { type: 'Point', coordinates: r.end_point },
        properties: { slug: r.slug, kind: 'end', name: r.name },
      })
    }
  }
  return { type: 'FeatureCollection', features }
}

function fitToReaches() {
  if (!map || props.reaches.length === 0) return
  const coords: [number, number][] = []
  for (const r of props.reaches) {
    if (r.centerline) coords.push(...allCoordsOf(r.centerline))
    if (r.start_point) coords.push(r.start_point)
    if (r.end_point)   coords.push(r.end_point)
  }
  const valid = coords.filter(c => Number.isFinite(c[0]) && Number.isFinite(c[1])
    && (Math.abs(c[0]) > 0.001 || Math.abs(c[1]) > 0.001))
  if (valid.length === 0) return
  const lngs = valid.map(c => c[0])
  const lats = valid.map(c => c[1])
  map.fitBounds(
    [[Math.min(...lngs), Math.min(...lats)], [Math.max(...lngs), Math.max(...lats)]],
    { padding: 60, maxZoom: 13, duration: 600 },
  )
}

function updateSources() {
  if (!map) return
  ;(map.getSource('basin-reaches')      as maplibregl.GeoJSONSource | undefined)?.setData(buildReachFC())
  ;(map.getSource('basin-endpoints')    as maplibregl.GeoJSONSource | undefined)?.setData(buildEndpointsFC())
  ;(map.getSource('basin-tributaries')  as maplibregl.GeoJSONSource | undefined)?.setData(props.network?.tributaries ?? EMPTY_FC)
  ;(map.getSource('basin-mainstem')     as maplibregl.GeoJSONSource | undefined)?.setData(props.network?.mainstem    ?? EMPTY_FC)
}

// Public: called by the parent page when the D3 tree node is clicked
function flyToReach(slug: string) {
  if (!map) return
  const reach = props.reaches.find(r => r.slug === slug)
  if (!reach?.centerline) return
  const coords = allCoordsOf(reach.centerline)
    .filter(c => Number.isFinite(c[0]) && Number.isFinite(c[1]))
  if (coords.length === 0) return
  const lngs = coords.map(c => c[0])
  const lats = coords.map(c => c[1])
  map.fitBounds(
    [[Math.min(...lngs), Math.min(...lats)], [Math.max(...lngs), Math.max(...lats)]],
    { padding: 80, maxZoom: 14, duration: 800 },
  )
  emit('select', slug)
}

defineExpose({ flyToReach })

onMounted(() => {
  if (!container.value) return

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
          attribution: 'Tiles © Esri — Esri, DeLorme, NAVTEQ',
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
          attribution: 'Tiles © Esri — Esri, i-cubed, USDA, USGS, AEX, GeoEye',
          maxzoom: 18,
        },
      },
      layers: [
        { id: 'street-tiles', type: 'raster', source: 'street', layout: { visibility: 'visible' } },
        { id: 'topo-tiles',   type: 'raster', source: 'topo',   layout: { visibility: 'none'    } },
        { id: 'esri-tiles',   type: 'raster', source: 'esri',   layout: { visibility: 'none'    } },
      ],
    },
    bounds: [-109.1, 36.9, -102.0, 41.1],
    fitBoundsOptions: { padding: 20 },
    attributionControl: false,
    fadeDuration: 0,
  })

  map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-right')
  map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-right')

  map.on('load', () => {
    // Network sources — added before reach layers so they render underneath
    map!.addSource('basin-tributaries', { type: 'geojson', data: props.network?.tributaries ?? EMPTY_FC })
    map!.addSource('basin-mainstem',    { type: 'geojson', data: props.network?.mainstem    ?? EMPTY_FC })

    map!.addLayer({
      id: 'basin-tributaries', type: 'line', source: 'basin-tributaries',
      paint: {
        'line-color': '#60a5fa',
        'line-width': 2.5,
        'line-opacity': 0.8,
      },
    })
    map!.addLayer({
      id: 'basin-mainstem', type: 'line', source: 'basin-mainstem',
      paint: {
        'line-color': '#0d9488',
        'line-width': 3,
        'line-opacity': 0.9,
      },
    })

    map!.addSource('basin-reaches', {
      type: 'geojson',
      data: buildReachFC(),
    })

    map!.addSource('basin-endpoints', {
      type: 'geojson',
      data: buildEndpointsFC(),
    })

    // Glow layer
    map!.addLayer({
      id: 'basin-glow', type: 'line', source: 'basin-reaches',
      paint: {
        'line-color': classColorExpr(),
        'line-width': ['interpolate', ['linear'], ['zoom'], 6, 8, 14, 18],
        'line-opacity': 0.18,
        'line-blur': 4,
      },
    })

    // Main reach lines, colored by difficulty
    map!.addLayer({
      id: 'basin-lines', type: 'line', source: 'basin-reaches',
      paint: {
        'line-color': classColorExpr(),
        'line-width': ['interpolate', ['linear'], ['zoom'], 6, 2.5, 14, 5],
        'line-opacity': 1,
        'line-cap': 'round',
        'line-join': 'round',
      },
    })

    // Invisible wide hit area for hover/click
    map!.addLayer({
      id: 'basin-lines-hit', type: 'line', source: 'basin-reaches',
      paint: { 'line-color': 'transparent', 'line-width': 14, 'line-opacity': 0 },
    })

    // Start (green) / end (red) circles — matches admin reach map
    map!.addLayer({
      id: 'basin-endpoints', type: 'circle', source: 'basin-endpoints',
      paint: {
        'circle-radius': 6,
        'circle-color': ['match', ['get', 'kind'], 'start', '#16a34a', '#dc2626'],
        'circle-stroke-width': 1.5,
        'circle-stroke-color': '#ffffff',
        'circle-opacity': 1,
      },
    })

    map!.on('mouseenter', 'basin-lines-hit', (e) => {
      map!.getCanvas().style.cursor = 'pointer'
      const p = e.features?.[0]?.properties
      if (!p) return
      const display  = p.common_name ?? p.name
      const classMax = p.class_max as number | null
      const classStr = classMax != null ? ` · Class ${classMax}` : ''
      reachTooltip.setLngLat(e.lngLat).setHTML(
        `<strong>${display}</strong>${classStr}`
      ).addTo(map!)
    })
    map!.on('mousemove', 'basin-lines-hit', (e) => {
      reachTooltip.setLngLat(e.lngLat)
    })
    map!.on('mouseleave', 'basin-lines-hit', () => {
      map!.getCanvas().style.cursor = ''
      reachTooltip.remove()
    })
    map!.on('click', 'basin-lines-hit', (e) => {
      const slug = e.features?.[0]?.properties?.slug
      if (slug) flyToReach(slug)
    })

    mapReady.value = true
    updateSources()   // handles data that arrived while map was loading
    fitToReaches()
  })

  map.on('error', e => console.warn('[BasinMap]', e.error?.message ?? e))
})

onUnmounted(() => {
  reachTooltip.remove()
  map?.remove()
  map = null
})

watch(() => props.reaches, () => {
  updateSources()
  fitToReaches()
}, { deep: true })

watch(() => props.network, () => {
  updateSources()
})
</script>

<style>
.basin-map-tooltip .maplibregl-popup-content {
  padding: 5px 10px !important;
  font-size: 0.8rem;
  font-weight: 500;
  background: rgba(17, 24, 39, 0.92) !important;
  color: #f9fafb !important;
  border-radius: 6px !important;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3) !important;
}
.basin-map-tooltip .maplibregl-popup-tip {
  border-top-color: rgba(17, 24, 39, 0.92) !important;
}
</style>
