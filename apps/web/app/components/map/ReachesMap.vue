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

const props = defineProps<{ hoveredSlug?: string | null }>()
const emit  = defineEmits<{
  (e: 'reaches-updated', reaches: { slug: string; name: string; class_max: number | null }[]): void
  (e: 'bounds-updated', bbox: string): void
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
  const lngs = coords.map(c => c[0])
  const lats  = coords.map(c => c[1])
  map.fitBounds(
    [[Math.min(...lngs), Math.min(...lats)], [Math.max(...lngs), Math.max(...lats)]],
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

// Initial viewport — lower 48 states so all reaches are visible on load
const INITIAL_BBOX = { west: -124.8, south: 24.4, east: -66.9, north: 49.4 }

// ── Difficulty config ─────────────────────────────────────────────────────────

const DIFFICULTY = [
  { maxClass: 2.4, color: '#16a34a', imageId: 'diff-1-2', label: 'Class I–II' },
  { maxClass: 3.9, color: '#3b82f6', imageId: 'diff-3',   label: 'Class III'  },
  { maxClass: 4.9, color: '#111827', imageId: 'diff-4',   label: 'Class IV'   },
  { maxClass: 99,  color: '#111827', imageId: 'diff-5',   label: 'Class V'    },
]

const DIFFICULTY_LEGEND = [
  { label: 'Class I–II', symbol: circleSvg('#16a34a')     },
  { label: 'Class III',  symbol: squareSvg('#3b82f6')     },
  { label: 'Class IV',   symbol: diamondSvg('#111827')    },
  { label: 'Class V',    symbol: dblDiamondSvg('#111827') },
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

// Load an SVG string into the map sprite as a named image
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
          attribution: 'Tiles © Esri — Esri, i-cubed, USDA, USGS, AEX, GeoEye, Getmapping, Aerogrid, IGN, IGP, UPR-EGP',
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
    // Register difficulty shape images in the sprite
    await Promise.all([
      loadSvgImage(map!, 'diff-1-2', circleSvg('#16a34a'),     20, 20),
      loadSvgImage(map!, 'diff-3',   squareSvg('#3b82f6'),     20, 20),
      loadSvgImage(map!, 'diff-4',   diamondSvg('#111827'),    20, 20),
      loadSvgImage(map!, 'diff-5',   dblDiamondSvg('#111827'), 36, 20),
    ])
    mapReady.value = true
    await loadReaches()
  })

  map.on('moveend', loadReaches)
  map.on('error', e => console.warn('[ReachesMap]', e.error?.message ?? e))
})

onUnmounted(() => {
  map?.remove()
  map = null
})

// ── Data loading ──────────────────────────────────────────────────────────────

interface ReachFeature {
  type: 'Feature'
  geometry: { type: string; coordinates: any }
  properties: {
    id: string; name: string; slug: string
    class_max: number | null; flow_status: string
  }
}

async function loadReaches() {
  if (!map) return
  const b = map.getBounds()
  const bbox = `${b.getWest()},${b.getSouth()},${b.getEast()},${b.getNorth()}`
  emit('bounds-updated', bbox)
  try {
    const res = await fetch(`${apiBase}/api/v1/reaches/map?bbox=${bbox}`)
    if (!res.ok) return
    const fc = await res.json()
    const features: ReachFeature[] = fc.features ?? []
    loadedFeatures = features
    updateLayers(features)
    emit('reaches-updated', features.map(f => ({
      slug:      f.properties.slug,
      name:      f.properties.name,
      class_max: f.properties.class_max,
    })))
  } catch (e) {
    console.warn('[ReachesMap] fetch:', e)
  }
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

  // Build a point feature at the midpoint of each reach for the cluster source
  const markerFeatures = features.flatMap(f => {
    const mid = midpoint(f)
    if (!mid) return []
    const diff = difficultyFor(f.properties.class_max)
    return [{
      type: 'Feature' as const,
      geometry: { type: 'Point' as const, coordinates: mid },
      properties: {
        slug: f.properties.slug,
        name: f.properties.name,
        icon: diff.imageId,
        label: diff.label,
      },
    }]
  })
  const markerFC = { type: 'FeatureCollection' as const, features: markerFeatures }

  // ── Update or create reach line layers ──────────────────────────────────────
  if (map.getSource('reaches')) {
    ;(map.getSource('reaches') as maplibregl.GeoJSONSource).setData(lineFC)
  } else {
    map.addSource('reaches', { type: 'geojson', data: lineFC })

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

    // Hover highlight layer — initially shows nothing
    map.addLayer({
      id: 'reach-highlight', type: 'line', source: 'reaches',
      filter: ['==', ['get', 'slug'], ''],
      paint: {
        'line-color': '#facc15',
        'line-width': ['interpolate', ['linear'], ['zoom'], 6, 4, 12, 8],
        'line-opacity': 1,
      },
    })

    hoverPopup = new maplibregl.Popup({
      closeButton: false, closeOnClick: false, offset: 8,
      className: 'reach-map-tooltip',
    })

    map.on('mouseenter', 'reach-lines', e => {
      if (!map || !e.features?.length) return
      map.getCanvas().style.cursor = 'pointer'
      if (map.getZoom() < TOOLTIP_MIN_ZOOM) return
      if (clickPopup?.isOpen()) return
      const p = e.features[0].properties as any
      const cfs = p.current_cfs != null ? `${Number(p.current_cfs).toLocaleString()} cfs` : null
      const age = p.last_reading_at ? relativeTime(p.last_reading_at) : null
      const body = cfs ? `${cfs}${age ? ` · ${age}` : ''}` : 'No recent reading'
      const statusColors: Record<string, string> = {
        runnable: '#22c55e',   // fun/optimal — green
        caution:  '#eab308',   // minimum/pushy — yellow
        low:      '#ef4444',   // too_low — red
        flood:    '#3b82f6',   // flood — blue
        unknown:  'rgba(255,255,255,0.5)',
      }
      const bodyColor = statusColors[String(p.flow_status ?? 'unknown')] ?? 'rgba(255,255,255,0.5)'
      const title = (p.put_in_name && p.take_out_name)
        ? `${p.put_in_name} to ${p.take_out_name}${p.common_name ? ` (${p.common_name})` : ''}`
        : (p.common_name ?? p.name)
      hoverPopup!
        .setLngLat(e.lngLat)
        .setHTML(`<strong>${title}</strong><br/><span style="color:${bodyColor};font-size:0.85em">${body}</span>`)
        .addTo(map)
    })
    map.on('mousemove', 'reach-lines', e => { hoverPopup!.setLngLat(e.lngLat) })
    map.on('mouseleave', 'reach-lines', () => {
      if (map) map.getCanvas().style.cursor = ''
      hoverPopup!.remove()
    })
    map.on('click', 'reach-lines', e => {
      if (!map || !e.features?.length) return
      hoverPopup!.remove()
      showReachPopup(e.features[0].properties as any, e.lngLat)
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
    cluster: true,
    clusterMaxZoom: 11,
    clusterRadius: 48,
  })

  // Cluster bubble
  map.addLayer({
    id: 'diff-clusters', type: 'circle', source: 'diff-markers',
    filter: ['has', 'point_count'],
    paint: {
      'circle-color': '#1d4ed8',
      'circle-radius': ['step', ['get', 'point_count'], 16, 5, 20, 15, 24],
      'circle-stroke-width': 2,
      'circle-stroke-color': '#fff',
      'circle-opacity': 0.88,
    },
  })

  // Cluster count label
  map.addLayer({
    id: 'diff-cluster-count', type: 'symbol', source: 'diff-markers',
    filter: ['has', 'point_count'],
    layout: {
      'text-field': '{point_count_abbreviated}',
      'text-font': ['Noto Sans Bold'],
      'text-size': 12,
      'text-allow-overlap': true,
    },
    paint: { 'text-color': '#fff' },
  })

  // Individual difficulty icon
  map.addLayer({
    id: 'diff-points', type: 'symbol', source: 'diff-markers',
    filter: ['!', ['has', 'point_count']],
    layout: {
      'icon-image': ['get', 'icon'],
      'icon-size': 1.1,
      'icon-allow-overlap': true,
      'icon-ignore-placement': true,
    },
  })

  // Zoom into cluster on click
  map.on('click', 'diff-clusters', async e => {
    if (!map || !e.features?.length) return
    const clusterId = e.features[0].properties?.cluster_id as number
    const src = map.getSource('diff-markers') as maplibregl.GeoJSONSource
    const zoom = await src.getClusterExpansionZoom(clusterId)
    const coords = (e.features[0].geometry as GeoJSON.Point).coordinates as [number, number]
    map.flyTo({ center: coords, zoom })
  })

  // diff-points click reserved for future use

  map.on('mouseenter', 'diff-clusters', () => { if (map) map.getCanvas().style.cursor = 'pointer' })
  map.on('mouseleave', 'diff-clusters', () => { if (map) map.getCanvas().style.cursor = '' })
  map.on('mouseenter', 'diff-points',   () => { if (map) map.getCanvas().style.cursor = 'pointer' })
  map.on('mouseleave', 'diff-points',   () => { if (map) map.getCanvas().style.cursor = '' })
}

// ── Click popup ───────────────────────────────────────────────────────────────

let hoverPopup: maplibregl.Popup | null = null
let clickPopup: maplibregl.Popup | null = null

function showReachPopup(p: any, lngLat: maplibregl.LngLat) {
  if (!map) return
  clickPopup?.remove()

  const esc = (s: string) => String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  const cfs  = p.current_cfs != null ? `${Number(p.current_cfs).toLocaleString()} cfs` : null
  const age  = p.last_reading_at ? relativeTime(p.last_reading_at) : null
  const flow = cfs ? `${cfs}${age ? ` · ${age}` : ''}` : 'No recent reading'

  const displayName = (p.put_in_name && p.take_out_name)
    ? `${p.put_in_name} to ${p.take_out_name}${p.common_name ? ` (${p.common_name})` : ''}`
    : (p.common_name ?? p.name)
  const riverLine = p.river_name ? `<p class="rcp-river">${esc(p.river_name)}</p>` : ''

  const popup = new maplibregl.Popup({ offset: [0, -4], className: 'reach-click-popup' })
    .setLngLat(lngLat)
    .setHTML(`<div class="rcp-inner">
      ${riverLine}<p class="rcp-name">${esc(displayName)}</p>
      <p class="rcp-flow">${esc(flow)}</p>
      <div class="rcp-actions">
        <a class="rcp-btn rcp-btn-primary" href="/reaches/${esc(p.slug)}">View reach</a>
        ${p.gauge_id ? `<button class="rcp-btn rcp-btn-ghost" data-gauge-id="${esc(p.gauge_id)}">+ Dashboard</button>` : ''}
      </div>
    </div>`)
    .addTo(map)

  clickPopup = popup

  popup.getElement().querySelector('[data-gauge-id]')
    ?.addEventListener('click', () => {
      emit('gauge-add', p.gauge_id)
      popup.remove()
    })
}

// Only show hover tooltip when zoomed in enough that individual reaches are distinct
const TOOLTIP_MIN_ZOOM = 10

function relativeTime(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime()
  const m = Math.floor(ms / 60_000)
  if (m < 1)  return 'just now'
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ${m % 60}m ago`
}

watch(() => props.hoveredSlug, slug => {
  if (!map || !map.getLayer('reach-highlight')) return
  map.setFilter('reach-highlight', ['==', ['get', 'slug'], slug ?? ''])
})

function difficultyColorExpr(): maplibregl.ExpressionSpecification {
  return ['step', ['coalesce', ['get', 'class_max'], 0],
    '#16a34a',       // I–II  green
    2.5, '#3b82f6',  // III   blue
    4.0, '#111827',  // IV    black  (4.0, 4.5, 4.9 all stay here)
    5.0, '#111827',  // V     black
  ] as any
}
</script>

<style>
.reach-map-tooltip .maplibregl-popup-content {
  background: #1f2937;
  color: #f9fafb;
  border-radius: 6px !important;
  padding: 6px 10px !important;
  font-family: system-ui, sans-serif;
  font-size: 0.8rem;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3) !important;
}
.reach-map-tooltip .maplibregl-popup-tip {
  border-bottom-color: #1f2937 !important;
}
.reach-click-popup .maplibregl-popup-content {
  border-radius: 10px !important;
  padding: 0 !important;
  box-shadow: 0 4px 16px rgba(0,0,0,0.18) !important;
  min-width: 180px;
}
.rcp-inner {
  padding: 10px 14px 10px;
  font-family: system-ui, sans-serif;
}
.rcp-river {
  font-size: 0.7rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: #3b82f6;
  margin: 0 0 2px;
}
.rcp-name {
  font-weight: 600;
  font-size: 0.875rem;
  color: #111827;
  margin: 0 0 2px;
}
.rcp-flow {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0 0 10px;
}
.rcp-actions {
  display: flex;
  gap: 6px;
}
.rcp-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  padding: 5px 10px;
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
  border: none;
  transition: background 0.1s, color 0.1s;
}
.rcp-btn-primary {
  background: #2563eb;
  color: #fff;
}
.rcp-btn-primary:hover { background: #1d4ed8; }
.rcp-btn-ghost {
  background: #f3f4f6;
  color: #374151;
}
.rcp-btn-ghost:hover { background: #e5e7eb; }
</style>
