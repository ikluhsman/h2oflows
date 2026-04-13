<template>
  <div class="relative w-full h-full bg-gray-100 dark:bg-gray-900">
    <div ref="container" class="w-full h-full" />

    <div v-if="!mapReady" class="absolute inset-0 flex items-center justify-center text-gray-400 text-sm pointer-events-none">
      Loading map…
    </div>

    <div v-if="mapReady" class="absolute bottom-8 left-2 z-10 flex items-center gap-1.5">
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
      <button
        class="flex items-center gap-1 px-2 py-1 rounded-md shadow border text-xs font-medium transition-colors"
        :class="locating
          ? 'bg-blue-50 dark:bg-blue-950 border-blue-300 dark:border-blue-700 text-blue-600 dark:text-blue-400'
          : locateError
            ? 'bg-red-50 dark:bg-red-950 border-red-300 dark:border-red-700 text-red-600 dark:text-red-400'
            : 'bg-white/90 dark:bg-gray-800/90 border-gray-200 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700'"
        :disabled="locating"
        :title="locateError || 'Zoom to my location'"
        @click="locateMe"
      >
        <svg class="w-3 h-3 shrink-0" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2"><circle cx="10" cy="10" r="3"/><circle cx="10" cy="10" r="7.5" stroke-dasharray="2 2"/><line x1="10" y1="1" x2="10" y2="3.5"/><line x1="10" y1="16.5" x2="10" y2="19"/><line x1="1" y1="10" x2="3.5" y2="10"/><line x1="16.5" y1="10" x2="19" y2="10"/></svg>
        <span>{{ locateError || (locating ? 'Locating…' : 'My location') }}</span>
      </button>
    </div>

    <!-- Difficulty legend -->
    <div v-if="mapReady" class="absolute bottom-8 right-2 z-10 bg-white/90 dark:bg-gray-900/90 backdrop-blur rounded-lg border border-gray-200 dark:border-gray-700 px-3 py-2 text-xs space-y-1.5 shadow">
      <p class="font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide text-[10px] mb-1">Difficulty</p>
      <div v-for="d in DIFFICULTY_LEGEND" :key="d.label" class="flex items-center gap-2">
        <span class="shrink-0" v-html="d.symbol" />
        <span class="text-gray-700 dark:text-gray-300">{{ d.label }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

export interface ReachListItem {
  slug:        string
  name:        string        // full display form: "put_in to take_out" or reach name
  common_name: string | null // short common name: "Foxton", "The Numbers", etc.
  class_max:   number | null
  flow_status: string
  current_cfs: number | null
}

const props = defineProps<{ hoveredSlug?: string | null }>()
const emit  = defineEmits<{
  (e: 'reaches-updated', reaches: ReachListItem[]): void
  (e: 'bounds-updated', bbox: string): void
  (e: 'zoom-updated', zoom: number): void
  (e: 'hover-changed', slug: string | null): void
  (e: 'reach-click', slug: string): void
  (e: 'gauge-add', gaugeId: string): void
}>()

// Local cache of loaded reach features so flyToSlug can look up geometry
let loadedFeatures: ReachFeature[] = []

function flyToSlug(slug: string) {
  if (!map) return
  const f = loadedFeatures.find(f => f.properties.slug === slug)
  if (!f?.geometry) return
  const coords: [number, number][] = f.geometry.type === 'LineString'
    ? f.geometry.coordinates
    : (f.geometry.coordinates as [number, number][][]).flat()
  if (coords.length < 2) return
  map.fitBounds(
    [[Math.min(...coords.map(c => c[0])), Math.min(...coords.map(c => c[1]))],
     [Math.max(...coords.map(c => c[0])), Math.max(...coords.map(c => c[1]))]],
    { padding: 80, maxZoom: 14, duration: 800 },
  )
}

defineExpose({ flyToSlug })

const { apiBase } = useRuntimeConfig().public
const container   = ref<HTMLDivElement>()
const mapReady    = ref(false)
const locating    = ref(false)
const locateError = ref('')
const basemap = ref<'street' | 'topo' | 'satellite'>('street')
const BASEMAP_OPTIONS = [
  { value: 'street',    label: 'Street'    },
  { value: 'topo',      label: 'Topo'      },
  { value: 'satellite', label: 'Satellite' },
] as const

function setBasemap(value: 'street' | 'topo' | 'satellite') {
  if (!map) return
  basemap.value = value
  map.setLayoutProperty('street-tiles', 'visibility', value === 'street'    ? 'visible' : 'none')
  map.setLayoutProperty('topo-tiles',   'visibility', value === 'topo'      ? 'visible' : 'none')
  map.setLayoutProperty('esri-tiles',   'visibility', value === 'satellite' ? 'visible' : 'none')
}

function locateMe() {
  if (!map || !navigator.geolocation) return
  locating.value = true
  locateError.value = ''
  navigator.geolocation.getCurrentPosition(
    pos => {
      locating.value = false
      map!.flyTo({ center: [pos.coords.longitude, pos.coords.latitude], zoom: 11, duration: 1000 })
    },
    () => {
      locating.value = false
      locateError.value = 'Location unavailable'
      setTimeout(() => { locateError.value = '' }, 3000)
    },
    { timeout: 10_000 },
  )
}

let map: maplibregl.Map | null = null

const reachTooltip = new maplibregl.Popup({
  closeButton: false, closeOnClick: false, offset: [0, -8],
  className: 'reach-map-tooltip',
})

// Initial viewport — western US (Colorado + surrounding states)
const INITIAL_BBOX = { west: -116.0, south: 35.5, east: -101.5, north: 45.5 }

// ── Difficulty config ─────────────────────────────────────────────────────────

// Icons: I-II circle, II+ blue square (previewing III), III blue square,
//        III+ black diamond (previewing IV), IV black diamond, V double diamond
// Lines: green <3.0, blue 3.0–3.9, black 4.0–4.9, red 5.0+
const DIFFICULTY = [
  { maxClass: 2.4, color: '#16a34a', imageId: 'diff-1-2', label: 'Class I–II'  },
  { maxClass: 2.9, color: '#16a34a', imageId: 'diff-3',   label: 'Class II+'   }, // green line, blue square
  { maxClass: 3.4, color: '#3b82f6', imageId: 'diff-3',   label: 'Class III'   },
  { maxClass: 3.9, color: '#3b82f6', imageId: 'diff-4',   label: 'Class III+'  }, // blue line, black diamond
  { maxClass: 4.9, color: '#1f2937', imageId: 'diff-4',   label: 'Class IV'    },
  { maxClass: 99,  color: '#1f2937', imageId: 'diff-5',   label: 'Class V'     },
]

const DIFFICULTY_LEGEND = [
  { label: 'Class I–II', symbol: circleSvg('#16a34a')     },
  { label: 'Class III',  symbol: squareSvg('#3b82f6')     },
  { label: 'Class IV',   symbol: diamondSvg('#1f2937')    },
  { label: 'Class V',    symbol: dblDiamondSvg('#1f2937') },
]

function difficultyFor(classMax: number | null) {
  const c = classMax ?? 0
  return DIFFICULTY.find(d => c <= d.maxClass) ?? DIFFICULTY[DIFFICULTY.length - 1]
}

// ── SVG helpers ───────────────────────────────────────────────────────────────

function circleSvg(color: string) {
  return `<svg width="16" height="16" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
    <circle cx="10" cy="10" r="8" fill="${color}" stroke="white" stroke-width="1.5"/></svg>`
}

function squareSvg(color: string) {
  return `<svg width="16" height="16" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
    <rect x="2" y="2" width="16" height="16" rx="1.5" fill="${color}" stroke="white" stroke-width="1.5"/></svg>`
}
function diamondSvg(color: string) {
  return `<svg width="16" height="16" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
    <path d="M10 1 L19 10 L10 19 L1 10 Z" fill="${color}" stroke="white" stroke-width="1.5"/></svg>`
}
function dblDiamondSvg(color: string) {
  return `<svg width="28" height="16" viewBox="0 0 36 20" xmlns="http://www.w3.org/2000/svg">
    <path d="M9 1 L17 10 L9 19 L1 10 Z" fill="${color}" stroke="white" stroke-width="1.5"/>
    <path d="M26 1 L34 10 L26 19 L18 10 Z" fill="${color}" stroke="white" stroke-width="1.5"/></svg>`
}

function loadSvgImage(m: maplibregl.Map, id: string, svg: string, w: number, h: number): Promise<void> {
  return new Promise((resolve, reject) => {
    const img = new Image(w, h)
    img.onload = () => { m.addImage(id, img); resolve() }
    img.onerror = reject
    img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg)
  })
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

onMounted(async () => {
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
          attribution: 'Tiles © Esri',
          maxzoom: 18,
        },
      },
      layers: [
        { id: 'street-tiles', type: 'raster', source: 'street', layout: { visibility: 'visible' } },
        { id: 'topo-tiles',   type: 'raster', source: 'topo',   layout: { visibility: 'none'    } },
        { id: 'esri-tiles',   type: 'raster', source: 'esri',   layout: { visibility: 'none'    } },
      ],
    },
    bounds: [INITIAL_BBOX.west, INITIAL_BBOX.south, INITIAL_BBOX.east, INITIAL_BBOX.north],
    fitBoundsOptions: { padding: 20 },
    attributionControl: false,
    fadeDuration: 0,
  })

  map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-right')
  map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-right')

  map.on('load', async () => {
    await Promise.all([
      loadSvgImage(map!, 'diff-1-2', circleSvg('#16a34a'),     20, 20),
      loadSvgImage(map!, 'diff-3',   squareSvg('#3b82f6'),     20, 20),
      loadSvgImage(map!, 'diff-4',   diamondSvg('#1f2937'),    20, 20),
      loadSvgImage(map!, 'diff-5',   dblDiamondSvg('#1f2937'), 36, 20),
    ])
    mapReady.value = true
    await loadAllReaches()   // one request — all features, cached server-side
  })

  map.on('moveend', () => {
    emit('zoom-updated', map!.getZoom())
    filterVisible()   // no network call — filter already-loaded features
  })
  map.on('error', e => console.warn('[ReachesMap]', e.error?.message ?? e))
})

onUnmounted(() => {
  reachTooltip.remove()
  map?.remove()
  map = null
})

// ── Data loading ──────────────────────────────────────────────────────────────

interface ReachFeature {
  type: 'Feature'
  geometry: { type: string; coordinates: any }
  properties: {
    id: string; name: string; slug: string
    class_max: number | null; flow_status: string; current_cfs: number | null
    put_in_name: string | null; take_out_name: string | null; common_name: string | null
    river_name: string | null; gauge_id: string | null
  }
}

// All features from the server — loaded once at startup.
let allServerFeatures: ReachFeature[] = []

/** One-time load of the full cached dataset from the server. */
async function loadAllReaches() {
  if (!map) return
  try {
    const res = await fetch(`${apiBase}/api/v1/reaches/map/all`)
    if (!res.ok) return
    const fc = await res.json()
    allServerFeatures = fc.features ?? []
    loadedFeatures = allServerFeatures
    filterVisible()
  } catch (e) {
    console.warn('[ReachesMap] fetch:', e)
  }
}

/** Filter already-loaded features to the current viewport and update layers. */
function filterVisible() {
  if (!map || allServerFeatures.length === 0) return
  const b = map.getBounds()
  emit('bounds-updated', `${b.getWest()},${b.getSouth()},${b.getEast()},${b.getNorth()}`)
  emit('zoom-updated', map.getZoom())

  // Simple bbox test — avoids the PostGIS round-trip entirely.
  const visible = allServerFeatures.filter(f => {
    if (!f.geometry?.coordinates) return false
    const coords: [number, number][] = f.geometry.type === 'LineString'
      ? f.geometry.coordinates
      : (f.geometry.coordinates as [number, number][][]).flat()
    return coords.some(([lng, lat]) =>
      lng >= b.getWest() && lng <= b.getEast() &&
      lat >= b.getSouth() && lat <= b.getNorth()
    )
  })

  updateLayers(visible)
  emit('reaches-updated', visible.map(f => ({
    slug:        f.properties.slug,
    name:        displayName(f.properties),
    common_name: f.properties.common_name ?? null,
    class_max:   f.properties.class_max,
    flow_status: f.properties.flow_status ?? 'unknown',
    current_cfs: f.properties.current_cfs ?? null,
  })))
}

function displayName(p: ReachFeature['properties']): string {
  if (p.put_in_name && p.take_out_name) {
    return p.common_name
      ? `${p.put_in_name}–${p.take_out_name} (${p.common_name})`
      : `${p.put_in_name}–${p.take_out_name}`
  }
  return p.common_name ?? p.name
}

function midpoint(f: ReachFeature): [number, number] | null {
  if (!f.geometry?.coordinates) return null
  const coords: [number, number][] = f.geometry.type === 'LineString'
    ? f.geometry.coordinates
    : (f.geometry.coordinates as [number, number][][]).flat()
  if (coords.length < 2) return null
  return coords[Math.floor(coords.length / 2)]
}

function updateLayers(features: ReachFeature[]) {
  if (!map) return

  const lineFC = { type: 'FeatureCollection' as const, features: features as any[] }

  const markerFeatures = features.flatMap(f => {
    const mid = midpoint(f)
    if (!mid) return []
    const diff = difficultyFor(f.properties.class_max)
    return [{
      type: 'Feature' as const,
      geometry: { type: 'Point' as const, coordinates: mid },
      properties: { slug: f.properties.slug, name: f.properties.name, icon: diff.imageId },
    }]
  })
  const markerFC = { type: 'FeatureCollection' as const, features: markerFeatures }

  // ── Update or create reach line layers ──────────────────────────────────────
  if (map.getSource('reaches')) {
    ;(map.getSource('reaches') as maplibregl.GeoJSONSource).setData(lineFC)
  } else {
    map.addSource('reaches', { type: 'geojson', data: lineFC })

    // Glow — class V gets a red glow, others a softer color-matched glow
    map.addLayer({
      id: 'reach-glow', type: 'line', source: 'reaches',
      paint: {
        'line-color': difficultyColorExpr(),
        'line-width': ['interpolate', ['linear'], ['zoom'], 6, 6, 12, 14],
        'line-opacity': 0.15, 'line-blur': 4,
      },
    })
    map.addLayer({
      id: 'reach-lines', type: 'line', source: 'reaches',
      paint: {
        'line-color': difficultyColorExpr(),
        'line-width': ['interpolate', ['linear'], ['zoom'], 6, 2.5, 12, 5],
        'line-opacity': 0.9,
      },
    })

    // Hover / selected highlight layer
    map.addLayer({
      id: 'reach-highlight', type: 'line', source: 'reaches',
      filter: ['==', ['get', 'slug'], ''],
      paint: {
        // V-class reaches highlight red; others highlight yellow
        'line-color': ['case',
          ['>=', ['coalesce', ['get', 'class_max'], 0], 5.0], '#ef4444',
          '#facc15',
        ],
        'line-width': ['interpolate', ['linear'], ['zoom'], 6, 5, 12, 10],
        'line-opacity': 0.95,
        'line-blur': ['case',
          ['>=', ['coalesce', ['get', 'class_max'], 0], 5.0], 3,
          0,
        ],
      },
    })

    // Click → navigate (no popup)
    map.on('click', 'reach-lines', e => {
      if (!map || !e.features?.length) return
      const slug = (e.features[0].properties as any).slug as string
      if (slug) emit('reach-click', slug)
    })

    // Hover → sync sidebar + tooltip
    map.on('mouseenter', 'reach-lines', e => {
      if (!map || !e.features?.length) return
      map.getCanvas().style.cursor = 'pointer'
      const p = e.features[0].properties as any
      emit('hover-changed', p.slug)
      const commonName = p.common_name ?? p.name
      const subtitle = p.common_name && p.common_name !== p.name
        ? `<br><span style="opacity:0.6;font-size:0.7rem;font-weight:400">${p.name}</span>`
        : ''
      reachTooltip.setLngLat(e.lngLat).setHTML(
        `<strong>${commonName}</strong>${subtitle}`
      ).addTo(map!)
    })
    map.on('mousemove', 'reach-lines', e => {
      reachTooltip.setLngLat(e.lngLat)
    })
    map.on('mouseleave', 'reach-lines', () => {
      if (map) map.getCanvas().style.cursor = ''
      reachTooltip.remove()
      emit('hover-changed', null)
    })
  }

  // ── Update or create clustered difficulty marker layers ─────────────────────
  if (map.getSource('diff-markers')) {
    ;(map.getSource('diff-markers') as maplibregl.GeoJSONSource).setData(markerFC)
    return
  }

  map.addSource('diff-markers', {
    type: 'geojson',
    data: markerFC,
  })

  map.addLayer({
    id: 'diff-points', type: 'symbol', source: 'diff-markers',
    minzoom: 9,
    layout: {
      'icon-image': ['get', 'icon'],
      'icon-size': 0.9,
      'icon-allow-overlap': true,
      'icon-ignore-placement': true,
    },
  })

  map.on('mouseenter', 'diff-points',   () => { if (map) map.getCanvas().style.cursor = 'pointer' })
  map.on('mouseleave', 'diff-points',   () => { if (map) map.getCanvas().style.cursor = '' })
}

function relativeTime(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime()
  const m = Math.floor(ms / 60_000)
  if (m < 1)  return 'just now'
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ${m % 60}m ago`
}

// Sync hover highlight from parent (sidebar hovering a row)
watch(() => props.hoveredSlug, slug => {
  if (!map || !map.getLayer('reach-highlight')) return
  map.setFilter('reach-highlight', ['==', ['get', 'slug'], slug ?? ''])
})

// Line colors: I-II green, III blue, IV near-black, V red (expert warning)
// Icon colors: all black for IV+V — only the line changes for V
function difficultyColorExpr(): maplibregl.ExpressionSpecification {
  return ['step', ['coalesce', ['get', 'class_max'], 0],
    '#16a34a',       // 0–2.9  I–II+  green
    3.0, '#3b82f6',  // 3.0–3.9 III   blue
    4.0, '#1f2937',  // 4.0–4.9 IV    near-black
    5.0, '#dc2626',  // 5.0+    V     red
  ] as any
}
</script>

<style>
.reach-map-tooltip .maplibregl-popup-content {
  padding: 5px 10px !important;
  font-size: 0.8rem;
  font-weight: 500;
  background: rgba(17, 24, 39, 0.92) !important;
  color: #f9fafb !important;
  border-radius: 6px !important;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3) !important;
}
.reach-map-tooltip .maplibregl-popup-tip {
  border-top-color: rgba(17, 24, 39, 0.92) !important;
}
</style>
