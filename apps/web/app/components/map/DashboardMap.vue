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
let map: maplibregl.Map | null = null
const activeMarkers: maplibregl.Marker[] = []

const gaugeTooltip = new maplibregl.Popup({
  closeButton: false, closeOnClick: false, offset: [0, -28],
  className: 'dash-map-tooltip',
})
let gaugeClickPopup: maplibregl.Popup | null = null

// Colorado bbox — all current reaches are here
const CO_BBOX = '-109.1,36.9,-102.0,41.1'

// Track last gauge-id fingerprint to know when to re-fit bounds
let prevGaugeIdSet = new Set<string>()
let hasFitBounds   = false
let fetchSeq       = 0   // incremented on each call; lets an in-flight fetch self-cancel if superseded

// ── Color helpers ─────────────────────────────────────────────────────────────


function difficultyColorExpr(): any {
  return ['step', ['coalesce', ['get', 'class_max'], 0],
    '#16a34a', 2.5, '#3b82f6', 4.0, '#111827', 5.0, '#111827']
}

// ── Geometry helpers ──────────────────────────────────────────────────────────

function midpoint(geom: any): [number, number] | null {
  if (!geom?.coordinates) return null
  const coords: [number, number][] = geom.type === 'LineString'
    ? geom.coordinates
    : (geom.coordinates as [number, number][][]).flat()
  if (coords.length < 2) return null
  return coords[Math.floor(coords.length / 2)] ?? null
}

function allCoordsOf(geom: any): [number, number][] {
  if (!geom?.coordinates) return []
  return geom.type === 'LineString'
    ? geom.coordinates
    : (geom.coordinates as [number, number][][]).flat()
}

// ── Marker element ────────────────────────────────────────────────────────────

function makeGaugePinEl(gauge: WatchedGauge, pos: [number, number]): HTMLElement {
  const color = '#6366f1'  // indigo-500
  const el = document.createElement('div')
  el.style.cssText = 'cursor:pointer;filter:drop-shadow(0 2px 4px rgba(0,0,0,0.4));transition:filter 0.12s'
  el.innerHTML = `<svg width="28" height="36" viewBox="0 0 28 36" xmlns="http://www.w3.org/2000/svg">
    <path d="M14 1C7.1 1 1.5 6.6 1.5 13.5c0 8.7 12.5 20.5 12.5 20.5S26.5 22.2 26.5 13.5C26.5 6.6 20.9 1 14 1z"
      fill="${color}" stroke="white" stroke-width="1.5"/>
    <path d="M9 14.5 A5 5 0 0 1 19 14.5" fill="none" stroke="white" stroke-width="1.3" stroke-linecap="round" opacity="0.9"/>
    <line x1="14" y1="14.5" x2="17.2" y2="10.8" stroke="white" stroke-width="1.3" stroke-linecap="round"/>
    <circle cx="14" cy="14.5" r="1.2" fill="white"/>
  </svg>`

  const name    = gauge.name ?? gauge.externalId
  const cfsText = gauge.currentCfs != null ? `${Number(gauge.currentCfs).toLocaleString()} cfs` : '— cfs'
  const label   = name.length > 26 ? name.slice(0, 24) + '…' : name

  el.addEventListener('mouseenter', () => {
    if (!map) return
    gaugeTooltip.setLngLat(pos).setHTML(`<span>${label} · ${cfsText}</span>`).addTo(map)
  })
  el.addEventListener('mouseleave', () => gaugeTooltip.remove())

  el.addEventListener('click', () => {
    gaugeTooltip.remove()
    gaugeClickPopup?.remove()
    const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    const popup = new maplibregl.Popup({ offset: [0, -34], className: 'dash-gauge-popup' })
      .setLngLat(pos)
      .setHTML(`<div class="dgp-inner">
        <p class="dgp-name">${esc(name)}</p>
        <p class="dgp-cfs">${esc(cfsText)}</p>
        <button class="dgp-remove">× Remove from dashboard</button>
      </div>`)
      .addTo(map!)
    gaugeClickPopup = popup
    popup.getElement().querySelector('.dgp-remove')
      ?.addEventListener('click', () => { popup.remove(); emit('remove-gauge', gauge.id) })
    el.style.filter = 'drop-shadow(0 0 4px rgba(255,255,255,0.9)) drop-shadow(0 2px 6px rgba(0,0,0,0.5))'
    popup.on('close', () => { el.style.filter = 'drop-shadow(0 2px 4px rgba(0,0,0,0.4))' })
  })

  return el
}

// ── Data refresh ──────────────────────────────────────────────────────────────

function clearMarkers() {
  gaugeTooltip.remove()
  gaugeClickPopup?.remove()
  gaugeClickPopup = null
  for (const m of activeMarkers) m.remove()
  activeMarkers.length = 0
}

async function refreshData() {
  const m = map
  if (!m || !mapReady.value) return

  const seq = ++fetchSeq                   // capture sequence number for this call
  const currentIdSet = new Set(props.gauges.map(g => g.id))
  const addedIds = [...currentIdSet].filter(id => !prevGaugeIdSet.has(id))
  const idsChanged = addedIds.length > 0 || [...prevGaugeIdSet].some(id => !currentIdSet.has(id))
  if (idsChanged) hasFitBounds = false
  prevGaugeIdSet = currentIdSet

  // Collect all reach slugs across all gauges (primary + all associated)
  const slugSet = new Set(props.gauges.flatMap(g =>
    [g.reachSlug, ...(g.reachSlugs ?? [])].filter((s): s is string => !!s)
  ))

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

    // Place a marker for each gauge.
    // Prefer the gauge's own GPS location; fall back to midpoint of an associated reach centerline.
    const gaugePoints: [number, number][] = []
    for (const gauge of props.gauges) {
      let pos: [number, number] | null = null
      if (gauge.lng != null && gauge.lat != null) {
        pos = [gauge.lng, gauge.lat]
      }
      if (!pos) {
        const slugsToTry = [gauge.reachSlug, ...(gauge.reachSlugs ?? [])].filter((s): s is string => !!s)
        for (const slug of slugsToTry) {
          const feature = bySlug.get(slug)
          if (feature) { pos = midpoint(feature.geometry); break }
        }
      }
      if (!pos) continue

      gaugePoints.push(pos)
      const marker = new maplibregl.Marker({ element: makeGaugePinEl(gauge, pos), anchor: 'bottom' })
        .setLngLat(pos)
        .addTo(m)
      activeMarkers.push(marker)
    }

    // When the user adds a new gauge, zoom to that gauge + its reach centerline
    // so they immediately see the run they just bookmarked. On initial mount or
    // after a removal, fall back to fitting the full set of saved gauges.
    const validPoints = gaugePoints.filter(p => Number.isFinite(p[0]) && Number.isFinite(p[1]))
    let fitPoints: [number, number][] | null = null
    if (addedIds.length > 0) {
      const focusPts: [number, number][] = []
      for (const gauge of props.gauges) {
        if (!addedIds.includes(gauge.id)) continue
        if (Number.isFinite(gauge.lng) && Number.isFinite(gauge.lat)) {
          focusPts.push([gauge.lng as number, gauge.lat as number])
        }
        const slugsToTry = [gauge.reachSlug, ...(gauge.reachSlugs ?? [])].filter((s): s is string => !!s)
        for (const slug of slugsToTry) {
          const feature = bySlug.get(slug)
          if (feature) {
            // Guard against null/zero coordinates from OSM data that would
            // send fitBounds to null island or an extreme zoom-out.
            const validCoords = allCoordsOf(feature.geometry)
              .filter(c => Number.isFinite(c[0]) && Number.isFinite(c[1])
                        && (Math.abs(c[0]) > 0.001 || Math.abs(c[1]) > 0.001))
            focusPts.push(...validCoords)
          }
        }
      }
      if (focusPts.length > 0) fitPoints = focusPts
    }
    if (!fitPoints && (idsChanged || !hasFitBounds) && validPoints.length > 0) {
      fitPoints = validPoints
    }
    if (fitPoints) {
      const lngs = fitPoints.map(c => c[0])
      const lats = fitPoints.map(c => c[1])
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

<style>
.dash-map-tooltip .maplibregl-popup-content {
  padding: 5px 10px !important;
  font-size: 0.8rem;
  font-weight: 500;
  background: rgba(17, 24, 39, 0.92) !important;
  color: #f9fafb !important;
  border-radius: 6px !important;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3) !important;
}
.dash-map-tooltip .maplibregl-popup-tip {
  border-top-color: rgba(17, 24, 39, 0.92) !important;
}

.dash-gauge-popup .maplibregl-popup-content {
  border-radius: 10px !important;
  padding: 0 !important;
  box-shadow: 0 4px 16px rgba(0,0,0,0.18) !important;
  min-width: 170px;
}
.dgp-inner {
  padding: 10px 14px 8px;
  font-family: system-ui, sans-serif;
}
.dgp-name {
  font-weight: 600;
  font-size: 0.85rem;
  color: #111827;
  margin: 0 0 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 180px;
}
.dgp-cfs {
  font-size: 0.78rem;
  color: #6366f1;
  font-weight: 500;
  margin: 0 0 8px;
}
.dgp-remove {
  display: block;
  width: 100%;
  background: none;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 4px 8px;
  font-size: 0.75rem;
  color: #6b7280;
  cursor: pointer;
  text-align: center;
  transition: background 0.1s, color 0.1s;
}
.dgp-remove:hover {
  background: #fee2e2;
  border-color: #fca5a5;
  color: #dc2626;
}
</style>
