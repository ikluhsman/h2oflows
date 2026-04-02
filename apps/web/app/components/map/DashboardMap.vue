<template>
  <div class="relative w-full rounded-xl overflow-hidden bg-gray-100 dark:bg-gray-900" style="height: 420px;">
    <div ref="container" class="w-full h-full" />

    <div v-if="!mapReady" class="absolute inset-0 flex items-center justify-center text-gray-400 text-sm pointer-events-none">
      Loading map…
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

    <div v-if="mapReady && gauges.length === 0" class="absolute inset-0 flex items-center justify-center pointer-events-none">
      <p class="text-sm text-gray-400 bg-white/80 dark:bg-gray-900/80 rounded-lg px-4 py-2">
        Add gauges above to see them on the map.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import type { WatchedGauge } from '~/stores/watchlist'

const props = defineProps<{ gauges: WatchedGauge[] }>()
const emit  = defineEmits<{ (e: 'remove-gauge', id: string): void }>()

const { apiBase } = useRuntimeConfig().public
const container  = ref<HTMLDivElement>()
const mapReady   = ref(false)
const basemap = ref<'street' | 'topo' | 'satellite'>('topo')
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
let map: maplibregl.Map | null = null
const activeMarkers: maplibregl.Marker[] = []

// Colorado bbox — all current reaches are here
const CO_BBOX = '-109.1,36.9,-102.0,41.1'

// Track last gauge-id fingerprint to know when to re-fit bounds
let prevGaugeIds  = ''
let hasFitBounds  = false
let fetchSeq      = 0   // incremented on each call; lets an in-flight fetch self-cancel if superseded

// ── Color helpers ─────────────────────────────────────────────────────────────

function flowColor(status: string): string {
  return ({ runnable: '#16a34a', caution: '#d97706', low: '#9ca3af', flood: '#dc2626' } as Record<string, string>)[status] ?? '#9ca3af'
}

function difficultyColorExpr(): any {
  return ['step', ['coalesce', ['get', 'class_max'], 0],
    '#16a34a', 2.5, '#3b82f6', 3.5, '#111827', 5.0, '#111827']
}

// ── Geometry helpers ──────────────────────────────────────────────────────────

function midpoint(geom: any): [number, number] | null {
  if (!geom?.coordinates) return null
  const coords: [number, number][] = geom.type === 'LineString'
    ? geom.coordinates
    : (geom.coordinates as [number, number][][]).flat()
  if (coords.length < 2) return null
  return coords[Math.floor(coords.length / 2)]
}

function allCoordsOf(geom: any): [number, number][] {
  if (!geom?.coordinates) return []
  return geom.type === 'LineString'
    ? geom.coordinates
    : (geom.coordinates as [number, number][][]).flat()
}

// ── Marker element ────────────────────────────────────────────────────────────

function makeMarkerEl(gauge: WatchedGauge): HTMLElement {
  const accent = flowColor(gauge.flowStatus)
  const el = document.createElement('div')
  el.style.cssText = [
    'background:white',
    'border:1.5px solid #e5e7eb',
    `border-left:4px solid ${accent}`,
    'border-radius:8px',
    'padding:5px 8px 5px 10px',
    'display:flex',
    'align-items:center',
    'gap:6px',
    'box-shadow:0 2px 8px rgba(0,0,0,0.15)',
    'white-space:nowrap',
    'font-family:system-ui,-apple-system,sans-serif',
    'cursor:default',
    'user-select:none',
    'max-width:220px',
  ].join(';')

  const nameEl = document.createElement('span')
  nameEl.textContent = gauge.name ?? gauge.externalId
  nameEl.style.cssText = 'font-size:12px;font-weight:600;color:#111827;overflow:hidden;text-overflow:ellipsis;max-width:120px;display:block'

  const cfsEl = document.createElement('span')
  cfsEl.textContent = gauge.currentCfs != null
    ? `${Number(gauge.currentCfs).toLocaleString()} cfs`
    : '— cfs'
  cfsEl.style.cssText = `font-size:11px;color:${accent};font-weight:500`

  const btn = document.createElement('button')
  btn.textContent = '×'
  btn.title = 'Remove from dashboard'
  btn.style.cssText = 'background:none;border:none;cursor:pointer;color:#d1d5db;font-size:18px;line-height:1;padding:0 0 0 2px;display:flex;align-items:center'
  btn.addEventListener('mouseenter', () => { btn.style.color = '#ef4444' })
  btn.addEventListener('mouseleave', () => { btn.style.color = '#d1d5db' })
  btn.addEventListener('click', e => { e.stopPropagation(); emit('remove-gauge', gauge.id) })

  el.appendChild(nameEl)
  el.appendChild(cfsEl)
  el.appendChild(btn)
  return el
}

// ── Data refresh ──────────────────────────────────────────────────────────────

function clearMarkers() {
  for (const m of activeMarkers) m.remove()
  activeMarkers.length = 0
}

async function refreshData() {
  const m = map
  if (!m || !mapReady.value) return

  const seq = ++fetchSeq                   // capture sequence number for this call
  const currentIds = props.gauges.map(g => g.id).sort().join(',')
  const idsChanged = currentIds !== prevGaugeIds
  if (idsChanged) hasFitBounds = false
  prevGaugeIds = currentIds

  const slugSet = new Set(props.gauges.map(g => g.reachSlug).filter((s): s is string => !!s))

  let bySlug = new Map<string, any>()
  try {
    if (slugSet.size > 0) {
      const res = await fetch(`${apiBase}/api/v1/reaches/map?bbox=${CO_BBOX}`)
      if (!res.ok) return
      const fc = await res.json()
      if (seq !== fetchSeq) return          // a newer call superseded us — discard
      const matched: any[] = (fc.features ?? []).filter((f: any) => slugSet.has(f.properties.slug))
      bySlug = new Map(matched.map((f: any) => [f.properties.slug, f]))
      ;(m.getSource('dash-reaches') as maplibregl.GeoJSONSource | undefined)
        ?.setData({ type: 'FeatureCollection', features: matched })
    } else {
      ;(m.getSource('dash-reaches') as maplibregl.GeoJSONSource | undefined)
        ?.setData({ type: 'FeatureCollection', features: [] })
    }

    clearMarkers()

    // Place a marker for each gauge; fall back to gauge lat/lng when no centerline exists
    const gaugePoints: [number, number][] = []
    for (const gauge of props.gauges) {
      let pos: [number, number] | null = null
      if (gauge.reachSlug) {
        const feature = bySlug.get(gauge.reachSlug)
        if (feature) pos = midpoint(feature.geometry)
      }
      if (!pos && Number.isFinite(gauge.lng) && Number.isFinite(gauge.lat)) {
        pos = [gauge.lng as number, gauge.lat as number]
      }
      if (!pos) continue

      gaugePoints.push(pos)
      const marker = new maplibregl.Marker({ element: makeMarkerEl(gauge), anchor: 'left', offset: [6, 0] })
        .setLngLat(pos)
        .addTo(m)
      activeMarkers.push(marker)
    }

    const validPoints = gaugePoints.filter(p => Number.isFinite(p[0]) && Number.isFinite(p[1]))
    if ((idsChanged || !hasFitBounds) && validPoints.length > 0) {
      const lngs = validPoints.map(c => c[0])
      const lats  = validPoints.map(c => c[1])
      m.fitBounds(
        [[Math.min(...lngs), Math.min(...lats)], [Math.max(...lngs), Math.max(...lats)]],
        { padding: 80, maxZoom: 13, duration: 800 },
      )
      hasFitBounds = true
    }
  } catch (e) {
    console.warn('[DashboardMap] fetch:', e)
  }
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

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
          attribution: 'Tiles © Esri — Esri, i-cubed, USDA, USGS, AEX, GeoEye, Getmapping, Aerogrid, IGN, IGP, UPR-EGP',
          maxzoom: 18,
        },
      },
      layers: [
        { id: 'street-tiles', type: 'raster', source: 'street', layout: { visibility: 'none'    } },
        { id: 'topo-tiles',   type: 'raster', source: 'topo',   layout: { visibility: 'visible' } },
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
    map!.addSource('dash-reaches', {
      type: 'geojson',
      data: { type: 'FeatureCollection', features: [] },
    })

    // Glow layer
    map!.addLayer({
      id: 'dash-glow', type: 'line', source: 'dash-reaches',
      paint: {
        'line-color': difficultyColorExpr(),
        'line-width': ['interpolate', ['linear'], ['zoom'], 6, 6, 12, 14],
        'line-opacity': 0.12,
        'line-blur': 4,
      },
    })

    // Main line layer
    map!.addLayer({
      id: 'dash-lines', type: 'line', source: 'dash-reaches',
      paint: {
        'line-color': difficultyColorExpr(),
        'line-width': ['interpolate', ['linear'], ['zoom'], 6, 2.5, 12, 5],
        'line-opacity': 0.9,
      },
    })

    mapReady.value = true
    refreshData()
  })

  map.on('error', e => console.warn('[DashboardMap]', e.error?.message ?? e))
})

onUnmounted(() => {
  clearMarkers()
  map?.remove()
  map = null
})

// Re-render whenever gauge data changes (CFS updates re-draw labels; ID changes re-fit bounds)
watch(() => props.gauges, refreshData, { deep: true })
</script>
