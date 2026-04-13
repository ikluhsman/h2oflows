<template>
  <!-- Map + feature list side-by-side on sm+, stacked on mobile -->
  <div
    :class="isFullscreen
      ? 'fixed inset-0 z-50 flex flex-col sm:flex-row bg-white dark:bg-gray-950'
      : 'rounded-xl overflow-hidden border border-gray-200 dark:border-gray-700 flex flex-col sm:flex-row sm:h-140'"
  >

    <!-- MapLibre container — ref is on THIS element so MapLibre reads its own clientHeight -->
    <div
      ref="container"
      class="relative flex-1 bg-gray-100 dark:bg-gray-800"
      :class="isFullscreen ? '' : 'min-h-100 sm:min-h-0'"
    >
      <div
        v-if="!mapReady && hasCoords"
        class="absolute inset-0 flex items-center justify-center text-gray-400 text-sm pointer-events-none z-10"
      >
        Loading map…
      </div>
      <div
        v-if="!hasCoords"
        class="absolute inset-0 flex items-center justify-center text-gray-400 text-sm z-10"
      >
        No GPS data for this reach yet
      </div>
      <!-- Bottom-left controls -->
      <div v-if="mapReady" class="absolute bottom-7 left-2 z-10 flex gap-1">
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
          v-if="allFeatures.length > 0 || centerline"
          class="text-xs bg-white/90 dark:bg-gray-800/90 rounded-md px-2 py-1 shadow border border-gray-200 dark:border-gray-600 font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
          @click="exportKml"
        >⬇ KML</button>
        <button
          class="text-xs bg-white/90 dark:bg-gray-800/90 rounded-md px-2 py-1 shadow border border-gray-200 dark:border-gray-600 font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
          @click="toggleFullscreen"
        >
          <svg v-if="!isFullscreen" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3H5a2 2 0 00-2 2v3m18 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M3 16v3a2 2 0 002 2h3"/></svg>
          <svg v-else class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M4 14h6v6m10-10h-6V4m0 6l7-7M3 21l7-7"/></svg>
        </button>
      </div>
    </div>

    <!-- Feature list sidebar -->
    <div
      v-if="allFeatures.length > 0"
      class="sm:w-52 border-t sm:border-t-0 sm:border-l border-gray-100 dark:border-gray-800 overflow-y-auto h-44 sm:h-full"
    >
      <!-- Access group -->
      <template v-if="accessFeatures.length > 0">
        <p class="px-3 pt-2.5 pb-1 text-[10px] font-bold uppercase tracking-wider text-gray-400">Access</p>
        <div
          v-for="a in accessFeatures"
          :key="a.id"
          class="flex items-center gap-1 pr-1.5"
          :class="selectedId === a.id ? 'bg-gray-100 dark:bg-gray-800' : 'hover:bg-gray-50 dark:hover:bg-gray-800/60'"
        >
          <button
            class="flex-1 flex items-center gap-2 px-3 py-1.5 text-left transition-colors text-xs min-w-0"
            @mousedown.prevent
            @click="selectFeature(a.id, a.lng, a.lat)"
          >
            <span class="shrink-0 w-5 h-5 rounded-full flex items-center justify-center p-0.75"
              :style="{ background: accessColor(a.type) }"
              v-html="accessFeatureIcon(a.type)"
            />
            <span class="truncate text-gray-700 dark:text-gray-300">{{ a.label }}</span>
          </button>
          <a
            v-if="a.type === 'put_in' || a.type === 'take_out'"
            :href="`https://www.google.com/maps/dir/?api=1&destination=${a.lat},${a.lng}`"
            target="_blank"
            rel="noopener"
            class="shrink-0 w-5 h-5 rounded flex items-center justify-center text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/40 transition-colors"
            title="Get directions"
            @mousedown.prevent
            @click.stop
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="3 11 22 2 13 21 11 13 3 11"/></svg>
          </a>
        </div>
      </template>

      <!-- Rapids group -->
      <template v-if="rapidFeatures.length > 0">
        <p class="px-3 pt-2.5 pb-1 text-[10px] font-bold uppercase tracking-wider text-gray-400">Rapids</p>
        <button
          v-for="r in rapidFeatures"
          :key="r.id"
          class="w-full flex items-center gap-2 px-3 py-1.5 text-left hover:bg-gray-50 dark:hover:bg-gray-800/60 transition-colors text-xs"
          :class="selectedId === r.id ? 'bg-gray-100 dark:bg-gray-800' : ''"
          @mousedown.prevent
          @click="selectFeature(r.id, r.lng, r.lat)"
        >
          <span class="shrink-0 w-5 h-5 rounded-full flex items-center justify-center p-0.75 bg-blue-500"
            v-html="rapidFeatureIcon(r.isSurf)"
          />
          <span class="truncate text-gray-700 dark:text-gray-300">{{ r.label }}</span>
          <span v-if="r.classLabel" class="shrink-0 text-[10px] text-gray-400 font-medium">{{ r.classLabel }}</span>
        </button>
      </template>

      <!-- Gauges group -->
      <template v-if="allGauges.length > 0">
        <p class="px-3 pt-2.5 pb-1 text-[10px] font-bold uppercase tracking-wider text-gray-400">Gauges</p>
        <div
          v-for="g in allGauges"
          :key="g.id"
          class="flex items-center gap-1 pr-1.5"
          :class="selectedId === g.id ? 'bg-gray-100 dark:bg-gray-800' : 'hover:bg-gray-50 dark:hover:bg-gray-800/60'"
        >
          <button
            class="flex-1 flex items-center gap-2 px-3 py-1.5 text-left transition-colors text-xs min-w-0"
            :class="g.lng == null || g.lat == null ? 'opacity-50 cursor-default' : ''"
            @mousedown.prevent
            @click="g.lng != null && g.lat != null && selectFeature(g.id, g.lng, g.lat)"
          >
            <span class="shrink-0 w-5 h-5 rounded-full flex items-center justify-center p-0.75 bg-cyan-600"
              v-html="gaugeFeatureIcon(g.reach_relationship)"
            />
            <span class="truncate text-gray-700 dark:text-gray-300">{{ g.name ?? g.external_id ?? gaugeRelLabel(g.reach_relationship) }}</span>
          </button>
          <!-- Add to dashboard -->
          <button
            v-if="!onDashboard(g.id)"
            class="shrink-0 w-5 h-5 rounded flex items-center justify-center text-cyan-600 hover:bg-cyan-50 dark:hover:bg-cyan-900/40 transition-colors"
            title="Add to dashboard"
            @mousedown.prevent
            @click.stop="emit('gauge-add', g.id)"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
          </button>
          <span
            v-else
            class="shrink-0 w-5 h-5 rounded flex items-center justify-center text-emerald-500"
            title="On dashboard"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
          </span>
        </div>
      </template>
    </div>
  </div>

  <!-- Selected feature detail card -->
  <div
    v-if="selectedFeature"
    class="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50 px-3 py-3"
  >
    <div class="flex gap-2.5 items-start">
      <span
        class="shrink-0 w-6 h-6 rounded-full flex items-center justify-center p-1 mt-0.5"
        :style="{ background: selectedFeature.pinColor }"
        v-html="selectedFeature.circleIcon"
      />
      <!-- Content -->
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <p class="text-sm font-semibold text-gray-800 dark:text-gray-100 leading-tight">{{ selectedFeature.title }}</p>
          <span v-if="selectedFeature.classLabel" class="text-xs font-mono text-gray-500 dark:text-gray-400">Class {{ selectedFeature.classLabel }}</span>
        </div>
        <p v-if="selectedFeature.subtitle" class="text-xs text-gray-400 mt-0.5">{{ selectedFeature.subtitle }}</p>
        <p v-if="selectedFeature.desc" class="text-xs text-gray-600 dark:text-gray-400 mt-1 leading-relaxed line-clamp-3">{{ selectedFeature.desc }}</p>
        <p v-else-if="!selectedFeature.gaugeId && !selectedFeature.directionsUrl" class="text-xs text-gray-400 mt-1 italic">No description available.</p>
        <!-- Access directions link -->
        <div v-if="selectedFeature.directionsUrl" class="flex items-center gap-3 mt-2">
          <a
            :href="selectedFeature.directionsUrl"
            target="_blank"
            rel="noopener"
            class="flex items-center gap-1 text-xs text-blue-500 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="3 11 22 2 13 21 11 13 3 11"/></svg>
            Get directions
          </a>
        </div>
        <!-- Gauge actions: add to dashboard + source link -->
        <div v-if="selectedFeature.gaugeId" class="flex items-center gap-3 mt-2">
          <button
            v-if="!onDashboard(selectedFeature.gaugeId)"
            class="flex items-center gap-1 text-xs text-cyan-600 dark:text-cyan-400 hover:text-cyan-700 dark:hover:text-cyan-300 font-medium transition-colors"
            @click="emit('gauge-add', selectedFeature.gaugeId)"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
            Add to dashboard
          </button>
          <span v-else class="flex items-center gap-1 text-xs text-emerald-500 font-medium">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
            On dashboard
          </span>
          <a
            v-if="selectedFeature.sourceUrl"
            :href="selectedFeature.sourceUrl"
            target="_blank"
            rel="noopener"
            class="flex items-center gap-1 text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
            View source
          </a>
        </div>
      </div>
      <button
        class="shrink-0 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 text-xl leading-none"
        @click="selectedId = null"
      >×</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import { useWatchlistStore } from '~/stores/watchlist'
import { useRouter } from '#app'
import { useRuntimeConfig } from '#imports'
import { accessFeatureIcon, rapidFeatureIcon, gaugeFeatureIcon } from '~/utils/featureIcons'

// ── Types ─────────────────────────────────────────────────────────────────────

interface RapidFeature {
  id: string
  name: string
  class_rating: number | null
  description: string | null
  is_surf_wave: boolean
  lng: number | null
  lat: number | null
}

interface AccessFeature {
  id: string
  access_type: string
  name: string | null
  notes: string | null
  water_lng: number | null
  water_lat: number | null
}

interface GaugeProp {
  id: string
  name?: string | null
  external_id?: string | null
  source?: string | null
  reach_relationship?: string | null
  lng?: number | null
  lat?: number | null
}

const props = defineProps<{
  name?: string
  classMax?: number | null
  centerline?: any
  rapids: RapidFeature[]
  access: AccessFeature[]
  // Current reach slug — used to exclude self from nearby reach layer
  slug?: string
  // River name — loads all reaches on the same river instead of viewport bbox
  riverName?: string
  // Legacy single-gauge props — kept for backwards compat
  gaugeLng?: number | null
  gaugeLat?: number | null
  // Preferred: pass all gauges as an array so each gets a pin
  gauges?: GaugeProp[]
}>()

const emit = defineEmits<{
  (e: 'gauge-add', gaugeId: string): void
}>()

const watchlist = useWatchlistStore()
const onDashboard = (id: string) => watchlist.gauges.some(g => g.id === id)

const router = useRouter()
const config = useRuntimeConfig()

function gaugeSourceUrl(g: GaugeProp): string | null {
  if (g.source === 'usgs' && g.external_id)
    return `https://waterdata.usgs.gov/monitoring-location/${g.external_id}/`
  if (g.source === 'dwr' && g.external_id)
    return `https://dwr.state.co.us/Tools/Stations/${g.external_id}`
  return null
}

// ── Derived lists ─────────────────────────────────────────────────────────────

// Normalised list items for the sidebar
const accessFeatures = computed(() =>
  props.access
    .filter(a => a.water_lng != null && a.water_lat != null)
    .map(a => ({
      id:    a.id,
      type:  a.access_type,
      label: a.name ?? accessTypeLabel(a.access_type),
      notes: a.notes ?? '',
      lng:   a.water_lng!,
      lat:   a.water_lat!,
    }))
)

const rapidFeatures = computed(() =>
  props.rapids
    .filter(r => r.lng != null && r.lat != null)
    .map(r => ({
      id:         r.id,
      label:      r.name,
      desc:       r.description ?? '',
      isSurf:     r.is_surf_wave,
      classLabel: r.class_rating != null ? `${formatClass(r.class_rating)}` : null,
      lng:        r.lng!,
      lat:        r.lat!,
    }))
)

const allFeatures = computed(() => [...accessFeatures.value, ...rapidFeatures.value])

const selectedFeature = computed(() => {
  if (!selectedId.value) return null
  const rapid = rapidFeatures.value.find(r => r.id === selectedId.value)
  if (rapid) return {
    title:       rapid.label,
    classLabel:  rapid.classLabel,
    subtitle:    rapid.isSurf ? 'Surf wave' : null,
    desc:        rapid.desc || null,
    pinColor:    '#3b82f6',
    circleIcon:  rapidFeatureIcon(rapid.isSurf),
  }
  const access = accessFeatures.value.find(a => a.id === selectedId.value)
  if (access) return {
    title:         access.label !== accessTypeLabel(access.type) ? access.label : accessTypeLabel(access.type),
    classLabel:    null as string | null,
    subtitle:      access.label !== accessTypeLabel(access.type) ? accessTypeLabel(access.type) : null,
    desc:          access.notes || null,
    pinColor:      accessColor(access.type),
    circleIcon:    accessFeatureIcon(access.type),
    directionsUrl: (access.type === 'put_in' || access.type === 'take_out')
      ? `https://www.google.com/maps/dir/?api=1&destination=${access.lat},${access.lng}`
      : null,
  }
  const gauge = gaugeFeatures.value.find(g => g.id === selectedId.value)
  if (gauge) return {
    title:       gauge.name ?? gauge.external_id ?? 'Flow gauge',
    classLabel:  null as string | null,
    subtitle:    gaugeRelLabel(gauge.reach_relationship),
    desc:        null as string | null,
    pinColor:    '#0891b2',
    circleIcon:  gaugeFeatureIcon(gauge.reach_relationship),
    gaugeId:     gauge.id,
    sourceUrl:   gaugeSourceUrl(gauge),
  }
  return null
})


// All gauges — shown in the sidebar regardless of coordinates
const allGauges = computed<GaugeProp[]>(() => {
  if (props.gauges && props.gauges.length > 0) return props.gauges
  if (props.gaugeLng != null && props.gaugeLat != null) {
    return [{ id: 'gauge', lng: props.gaugeLng, lat: props.gaugeLat }]
  }
  return []
})

// Gauges with valid coordinates — used for map markers and bounds
const gaugeFeatures = computed<GaugeProp[]>(() =>
  allGauges.value.filter(g => g.lng != null && g.lat != null)
)

const hasCoords = computed(() =>
  props.centerline || allFeatures.value.length > 0 || gaugeFeatures.value.length > 0
)

// ── Map state ─────────────────────────────────────────────────────────────────

const container    = ref<HTMLDivElement>()
const mapReady     = ref(false)
const selectedId   = ref<string | null>(null)
const isFullscreen = ref(false)
const basemap = ref<'street' | 'topo' | 'satellite'>('street')
const BASEMAP_OPTIONS = [
  { value: 'street',    label: 'Street'    },
  { value: 'topo',      label: 'Topo'      },
  { value: 'satellite', label: 'Satellite' },
] as const
let map: maplibregl.Map | null = null
let clickPopup: maplibregl.Popup | null = null
let allMarkers: maplibregl.Marker[] = []

// ── Same-river reaches ────────────────────────────────────────────────────────
// Fetched once on load using river_name. Shown as dimmed lines beneath the
// current reach; clicking navigates to that reach's detail page.

async function fetchNearbyReaches() {
  if (!map || !props.riverName) return
  const url = `${config.public.apiBase}/api/v1/reaches/map?river_name=${encodeURIComponent(props.riverName)}`
  try {
    const fc = await $fetch<{ features: any[] }>(url)
    if (!map) return
    const source = map.getSource('other-reaches') as maplibregl.GeoJSONSource | undefined
    if (!source) return
    const filtered = {
      type: 'FeatureCollection' as const,
      features: (fc.features ?? []).filter(f => f.properties?.slug !== props.slug),
    }
    source.setData(filtered)
  } catch {
    // Silently ignore — other-river-reaches are a nice-to-have overlay.
  }
}
const markerEls = new Map<string, HTMLElement>()

// ── Lifecycle ─────────────────────────────────────────────────────────────────

onMounted(async () => {
  await nextTick()
  // Wait for the browser's layout pass so clientHeight is non-zero
  await new Promise<void>(r => requestAnimationFrame(() => r()))
  if (!container.value || !hasCoords.value) return

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
    center:   [gaugeFeatures.value[0]?.lng ?? -105.5, gaugeFeatures.value[0]?.lat ?? 39.2],
    zoom:     11,
    attributionControl: false,
    fadeDuration: 0,
  })

  map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-right')
  map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-right')

  map.on('load', () => {
    if (!map) return
    mapReady.value = true
    addLayers()
    fitBounds()
    fetchNearbyReaches()
  })


  map.on('error', (e) => { console.warn('[ReachMap]', e.error?.message ?? e) })

  // Scale rapid text labels with zoom level (larger when zoomed in on a rapid)
  map.on('zoom', () => {
    if (!map) return
    const z = map.getZoom()
    const size = z >= 15 ? '14px' : z >= 13 ? '12px' : '10px'
    document.querySelectorAll<HTMLElement>('.reach-map-rapid-label')
      .forEach(el => { el.style.fontSize = size })
  })
})

onUnmounted(() => {
  for (const m of allMarkers) m.remove()
  allMarkers = []
  markerEls.clear()
  map?.remove()
  map = null
})

// ── Layers ────────────────────────────────────────────────────────────────────

function addLayers() {
  if (!map) return

  // Other reaches in the viewport — rendered beneath the current reach.
  // Populated/updated by fetchNearbyReaches on load and moveend.
  map.addSource('other-reaches', {
    type: 'geojson',
    data: { type: 'FeatureCollection', features: [] },
  })
  // Difficulty color expression — mirrors reachLineColor() and DashboardMap
  const difficultyExpr = ['step', ['coalesce', ['get', 'class_max'], 0],
    '#6b7280', 2.5, '#16a34a', 4.0, '#3b82f6', 5.0, '#475569', 6.0, '#dc2626']
  map.addLayer({
    id: 'other-reaches-glow',
    type: 'line',
    source: 'other-reaches',
    paint: { 'line-color': difficultyExpr, 'line-width': 8, 'line-opacity': 0.15, 'line-blur': 4 },
  })
  map.addLayer({
    id: 'other-reaches-line',
    type: 'line',
    source: 'other-reaches',
    paint: { 'line-color': difficultyExpr, 'line-width': 2.5, 'line-opacity': 0.55 },
  })
  // Hover highlight — filtered to only the hovered feature
  map.addLayer({
    id: 'other-reaches-hover',
    type: 'line',
    source: 'other-reaches',
    filter: ['==', ['get', 'slug'], ''],
    paint: { 'line-color': difficultyExpr, 'line-width': 5, 'line-opacity': 0.7 },
  })
  // Wider invisible hit area for click detection on thin lines
  map.addLayer({
    id: 'other-reaches-hit',
    type: 'line',
    source: 'other-reaches',
    paint: { 'line-color': 'transparent', 'line-width': 14, 'line-opacity': 0 },
  })
  map.on('click', 'other-reaches-hit', (e) => {
    const slug = e.features?.[0]?.properties?.slug
    if (slug) router.push(`/reaches/${slug}`)
  })
  map.on('mouseenter', 'other-reaches-hit', (e) => {
    if (!map) return
    map.getCanvas().style.cursor = 'pointer'
    const slug = e.features?.[0]?.properties?.slug
    if (slug) map.setFilter('other-reaches-hover', ['==', ['get', 'slug'], slug])
  })
  map.on('mouseleave', 'other-reaches-hit', () => {
    if (!map) return
    map.getCanvas().style.cursor = ''
    map.setFilter('other-reaches-hover', ['==', ['get', 'slug'], ''])
  })

  // Centerline — colored by max rapid difficulty
  if (props.centerline) {
    const ratings = props.rapids.map(r => r.class_rating ?? 0).filter(n => n > 0)
    const maxRating = ratings.length ? Math.max(...ratings) : (props.classMax ?? null)
    const lineColor = reachLineColor(maxRating)

    map.addSource('centerline', {
      type: 'geojson',
      data: { type: 'Feature', geometry: props.centerline, properties: {} },
    })

    // Soft glow behind the line
    map.addLayer({
      id: 'centerline-glow',
      type: 'line',
      source: 'centerline',
      paint: { 'line-color': lineColor, 'line-width': 12, 'line-opacity': 0.18, 'line-blur': 5 },
    })

    // Main reach line
    map.addLayer({
      id: 'centerline',
      type: 'line',
      source: 'centerline',
      paint: { 'line-color': lineColor, 'line-width': 3.5, 'line-opacity': 0.92 },
    })
  }

  // Access point markers
  for (const a of accessFeatures.value) {
    const el = makePinEl(accessColor(a.type), accessIconUrl(a.type), accessIcon(a.type), a.id)
    el.title = `${accessTypeLabel(a.type)}${a.label !== accessTypeLabel(a.type) ? ' · ' + a.label : ''}`
    el.addEventListener('mouseenter', () => showTooltip(el, el.title, [a.lng, a.lat]))
    el.addEventListener('mouseleave', () => tooltip.remove())
    el.addEventListener('click', () => {
      clickPopup?.remove()
      const title = `${accessTypeLabel(a.type)} <span class="map-popup-class">${a.label}</span>`
      clickPopup = new maplibregl.Popup({ offset: [0, -32] })
        .setLngLat([a.lng, a.lat])
        .setHTML(`<div class="map-popup"><p class="map-popup-title">${title}</p>${a.notes ? `<p class="map-popup-desc">${a.notes}</p>` : ''}</div>`)
        .addTo(map!)
      setSelectedMarker(a.id)
    })
    const marker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
      .setLngLat([a.lng, a.lat])
      .addTo(map!)
    allMarkers.push(marker)
    markerEls.set(a.id, el)
  }

  // Gauge pins — cyan/teal color, "G" label with relationship subtitle
  for (const g of gaugeFeatures.value) {
    const relLabel = gaugeRelLabel(g.reach_relationship)
    const label = g.reach_relationship === 'upstream_indicator' ? '▲' :
                  g.reach_relationship === 'downstream_indicator' ? '▼' : '~'
    const el = makePinEl('#0891b2', null, label, g.id)
    el.title = `${relLabel}${g.name ? ': ' + g.name : g.external_id ? ': ' + g.external_id : ''}`
    el.addEventListener('mouseenter', () => showTooltip(el, el.title, [g.lng!, g.lat!]))
    el.addEventListener('mouseleave', () => tooltip.remove())
    el.addEventListener('click', () => {
      clickPopup?.remove()
      map!.flyTo({ center: [g.lng!, g.lat!], zoom: Math.max(map!.getZoom(), 14), duration: 600 })
      setSelectedMarker(g.id)
    })
    const marker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
      .setLngLat([g.lng!, g.lat!])
      .addTo(map!)
    allMarkers.push(marker)
    markerEls.set(g.id, el)
  }

  // Rapid labels — text only with a blue glow
  for (const r of rapidFeatures.value) {
    const el = document.createElement('div')
    el.dataset.markerId = r.id
    el.className = 'reach-map-rapid-label'
    el.tabIndex = -1
    el.addEventListener('mousedown', e => e.preventDefault())
    el.textContent = r.label
    el.title = r.label
    el.addEventListener('click', (e) => {
      e.stopPropagation()
      clickPopup?.remove()
      clickPopup = new maplibregl.Popup({ offset: [0, -8] })
        .setLngLat([r.lng, r.lat])
        .setHTML(`<div class="map-popup"><p class="map-popup-title">${r.label}</p>${r.desc ? `<p class="map-popup-desc">${r.desc}</p>` : ''}</div>`)
        .addTo(map!)
      setSelectedMarker(r.id)
    })
    const marker = new maplibregl.Marker({ element: el, anchor: 'center' })
      .setLngLat([r.lng, r.lat])
      .addTo(map!)
    allMarkers.push(marker)
    markerEls.set(r.id, el)
  }
}

const tooltip = new maplibregl.Popup({
  closeButton: false, closeOnClick: false, offset: [0, -28],
  className: 'reach-map-tooltip',
})

function showTooltip(_el: HTMLElement, text: string, lngLat: [number, number]) {
  if (!map) return
  tooltip.setLngLat(lngLat).setHTML(`<span>${text}</span>`).addTo(map)
}

/**
 * Build an SVG DOM element for an access point marker.
 * put_in    → green teardrop with down-arrow
 * take_out  → red teardrop with up-arrow
 * parking   → red teardrop with bold "P"
 * others    → gray teardrop with icon letter
 */
function makePinEl(color: string, _imgUrl: string | null, label: string, id: string): HTMLElement {
  const el = document.createElement('div')
  el.dataset.markerId = id
  el.dataset.pinColor = color
  // tabindex="-1" keeps the element programmatically focusable but not in the tab order.
  // mousedown.preventDefault() stops the browser from assigning focus on click, which
  // would cause window.scroll to track the element as MapLibre repositions it via transform.
  el.tabIndex = -1
  el.addEventListener('mousedown', e => e.preventDefault())
  el.style.cssText = 'cursor:pointer;filter:drop-shadow(0 2px 4px rgba(0,0,0,0.35));transition:filter 0.12s'
  el.innerHTML = makePinSvg(color, label)
  return el
}

function makePinSvg(color: string, label: string): string {
  const arrow = label === '↓'
    // Down-arrow (put-in)
    ? `<path d="M14 8 L14 19 M9 15 L14 20 L19 15" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>`
    : label === '↑'
    // Up-arrow (take-out)
    ? `<path d="M14 20 L14 9 M9 13 L14 8 L19 13" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>`
    // Text label (P, S, ◆, etc.)
    : `<text x="14" y="18" text-anchor="middle" dominant-baseline="middle"
        font-size="12" font-weight="800" font-family="system-ui,sans-serif" fill="white">${label}</text>`
  return `<svg width="28" height="36" viewBox="0 0 28 36" xmlns="http://www.w3.org/2000/svg">
    <path d="M14 2C7.9 2 3 6.9 3 13c0 7.5 11 21 11 21S25 20.5 25 13C25 6.9 20.1 2 14 2z"
      fill="${color}" stroke="white" stroke-width="1.5"/>
    ${arrow}
  </svg>`
}

// Kept for API compatibility — no longer used (inline SVG replaces KMZ images)
function accessIconUrl(_type: string): string | null { return null }

function setSelectedMarker(id: string) {
  selectedId.value = id
  for (const [mid, el] of markerEls) {
    const sel = mid === id
    if (sel) {
      const color = el.dataset.pinColor ?? '#6b7280'
      el.style.filter = `drop-shadow(0 0 5px ${color}) drop-shadow(0 2px 6px rgba(0,0,0,0.5))`
    } else {
      el.style.filter = 'drop-shadow(0 2px 4px rgba(0,0,0,0.4))'
    }
    el.style.zIndex = sel ? '10' : '1'
  }
}

// ── Basemap toggle ────────────────────────────────────────────────────────────

function setBasemap(value: 'street' | 'topo' | 'satellite') {
  if (!map) return
  basemap.value = value
  map.setLayoutProperty('street-tiles', 'visibility', value === 'street'    ? 'visible' : 'none')
  map.setLayoutProperty('topo-tiles',   'visibility', value === 'topo'      ? 'visible' : 'none')
  map.setLayoutProperty('esri-tiles',   'visibility', value === 'satellite' ? 'visible' : 'none')
}

// ── Fullscreen ───────────────────────────────────────────────────────────────

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
  if (isFullscreen.value) {
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
  nextTick(() => map?.resize())
}

function onEscKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && isFullscreen.value) toggleFullscreen()
}
onMounted(() => document.addEventListener('keydown', onEscKey))
onUnmounted(() => { document.removeEventListener('keydown', onEscKey); document.body.style.overflow = '' })

// ── KML export ────────────────────────────────────────────────────────────────

function exportKml() {
  const docName = props.name ?? 'Reach'
  const esc = (s: string) => s
    .replace(/&/g, '&amp;').replace(/</g, '&lt;')
    .replace(/>/g, '&gt;').replace(/"/g, '&quot;')

  // Determine line color from max rapid class (AABBGGRR format for KML)
  const ratings = props.rapids.map(r => r.class_rating ?? 0).filter(n => n > 0)
  const maxRating = ratings.length ? Math.max(...ratings) : null
  const lineColorKml = maxRating == null ? 'ffF64169'  // blue
    : maxRating <= 2 ? 'ff5EC522'                       // green
    : maxRating <= 3 ? 'ff08B8EA'                       // yellow
    : maxRating <= 4 ? 'ff1673F9'                       // orange
    : 'ff4444EF'                                         // red

  const styles = `
  <Style id="put_in">
    <IconStyle><scale>1.0</scale><Icon><href>http://maps.google.com/mapfiles/kml/paddle/grn-circle.png</href></Icon></IconStyle>
    <LabelStyle><scale>0.8</scale></LabelStyle>
  </Style>
  <Style id="take_out">
    <IconStyle><scale>1.0</scale><Icon><href>http://maps.google.com/mapfiles/kml/paddle/red-circle.png</href></Icon></IconStyle>
    <LabelStyle><scale>0.8</scale></LabelStyle>
  </Style>
  <Style id="shuttle_drop">
    <IconStyle><scale>1.0</scale><Icon><href>http://maps.google.com/mapfiles/kml/paddle/purple-circle.png</href></Icon></IconStyle>
    <LabelStyle><scale>0.8</scale></LabelStyle>
  </Style>
  <Style id="access">
    <IconStyle><scale>0.8</scale><Icon><href>http://maps.google.com/mapfiles/kml/paddle/ltblu-circle.png</href></Icon></IconStyle>
    <LabelStyle><scale>0.8</scale></LabelStyle>
  </Style>
  <Style id="rapid">
    <IconStyle><scale>0.9</scale><Icon><href>http://maps.google.com/mapfiles/kml/paddle/orange-circle.png</href></Icon></IconStyle>
    <LabelStyle><scale>0.8</scale></LabelStyle>
  </Style>
  <Style id="surf_wave">
    <IconStyle><scale>0.9</scale><Icon><href>http://maps.google.com/mapfiles/kml/paddle/blu-circle.png</href></Icon></IconStyle>
    <LabelStyle><scale>0.8</scale></LabelStyle>
  </Style>
  <Style id="river">
    <LineStyle><color>${lineColorKml}</color><width>4</width></LineStyle>
    <PolyStyle><fill>0</fill></PolyStyle>
  </Style>`

  // Access Points folder
  let accessFolder = ''
  if (accessFeatures.value.length > 0) {
    const marks = accessFeatures.value.map(a => `
    <Placemark>
      <name>${esc(a.label)}</name>
      ${a.notes ? `<description>${esc(a.notes)}</description>` : ''}
      <styleUrl>#${['put_in','take_out','shuttle_drop'].includes(a.type) ? a.type : 'access'}</styleUrl>
      <Point><coordinates>${a.lng},${a.lat},0</coordinates></Point>
    </Placemark>`).join('')
    accessFolder = `\n  <Folder><name>Access Points</name>${marks}\n  </Folder>`
  }

  // Rivers folder
  let riverFolder = ''
  if (props.centerline?.coordinates) {
    const raw = props.centerline.type === 'LineString'
      ? props.centerline.coordinates as [number, number][]
      : (props.centerline.coordinates as [number, number][][]).flat()
    const coordStr = raw.map((c: [number, number]) => `${c[0]},${c[1]},0`).join(' ')
    riverFolder = `\n  <Folder><name>Rivers</name>
    <Placemark>
      <name>${esc(docName)}</name>
      <styleUrl>#river</styleUrl>
      <LineString><tessellate>1</tessellate><coordinates>${coordStr}</coordinates></LineString>
    </Placemark>
  </Folder>`
  }

  // Rapids folder
  let rapidsFolder = ''
  if (rapidFeatures.value.length > 0) {
    const marks = rapidFeatures.value.map(r => `
    <Placemark>
      <name>${esc(r.label)}${r.classLabel ? ` (Class ${r.classLabel})` : ''}</name>
      ${r.desc ? `<description>${esc(r.desc)}</description>` : ''}
      <styleUrl>#${r.isSurf ? 'surf_wave' : 'rapid'}</styleUrl>
      <Point><coordinates>${r.lng},${r.lat},0</coordinates></Point>
    </Placemark>`).join('')
    rapidsFolder = `\n  <Folder><name>Rapids</name>${marks}\n  </Folder>`
  }

  const kml = `<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
  <Document>
    <name>${esc(docName)}</name>
    ${styles}
    ${accessFolder}
    ${riverFolder}
    ${rapidsFolder}
  </Document>
</kml>`

  const blob = new Blob([kml], { type: 'application/vnd.google-earth.kml+xml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${docName.toLowerCase().replace(/[^a-z0-9]+/g, '-')}.kml`
  a.click()
  URL.revokeObjectURL(url)
}

// ── Selection (list → map) ─────────────────────────────────────────────────────

function selectFeature(id: string, lng: number, lat: number) {
  if (!map) return
  map.flyTo({ center: [lng, lat], zoom: Math.max(map.getZoom(), 14), duration: 600 })
  clickPopup?.remove()
  // Gauges use the detail card only — no popup
  const rapid = rapidFeatures.value.find(r => r.id === id)
  const access = accessFeatures.value.find(a => a.id === id)
  if (rapid) {
    const title = `${rapid.label}${rapid.classLabel ? ` <span class="map-popup-class">Class ${rapid.classLabel}</span>` : ''}`
    clickPopup = new maplibregl.Popup({ offset: [0, -32] })
      .setLngLat([lng, lat])
      .setHTML(`<div class="map-popup"><p class="map-popup-title">${title}</p>${rapid.desc ? `<p class="map-popup-desc">${rapid.desc}</p>` : ''}</div>`)
      .addTo(map)
  } else if (access) {
    const title = `${accessTypeLabel(access.type)} <span class="map-popup-class">${access.label}</span>`
    clickPopup = new maplibregl.Popup({ offset: [0, -32] })
      .setLngLat([lng, lat])
      .setHTML(`<div class="map-popup"><p class="map-popup-title">${title}</p>${access.notes ? `<p class="map-popup-desc">${access.notes}</p>` : ''}</div>`)
      .addTo(map)
  }
  setSelectedMarker(id)
}

// ── Fit bounds ────────────────────────────────────────────────────────────────

function fitBounds() {
  if (!map) return
  const coords: [number, number][] = []
  rapidFeatures.value.forEach(r => coords.push([r.lng, r.lat]))
  accessFeatures.value.forEach(a => coords.push([a.lng, a.lat]))
  if (props.centerline?.coordinates) {
    const line = props.centerline.coordinates as [number, number][]
    if (line.length > 0) coords.push(line[0] as [number, number], line[line.length - 1] as [number, number])
  }
  gaugeFeatures.value.forEach(g => coords.push([g.lng!, g.lat!]))
  if (coords.length === 0) return
  if (coords.length === 1) { map.setCenter(coords[0] as [number, number]); map.setZoom(14); return }
  const lngs = coords.map(c => c[0])
  const lats = coords.map(c => c[1])
  map.fitBounds(
    [[Math.min(...lngs), Math.min(...lats)], [Math.max(...lngs), Math.max(...lats)]],
    { padding: 40, maxZoom: 15, duration: 0 },
  )
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// Color the reach centerline by difficulty — matches the home map scale.
function reachLineColor(maxRating: number | null): string {
  if (maxRating == null) return '#6b7280'  // gray   — no class data
  if (maxRating < 2.5)   return '#16a34a'  // green  — Class I–II
  if (maxRating < 4.0)   return '#3b82f6'  // blue   — Class III/III+
  if (maxRating < 5.0)   return '#1f2937'  // black  — Class IV (incl. IV+)
  return '#dc2626'                          // red    — Class V (expert warning)
}

function formatClass(v: number): string {
  const map: Record<number, string> = {
    1: 'I', 1.5: 'I+', 2: 'II', 2.5: 'II+',
    3: 'III', 3.5: 'III+', 4: 'IV', 4.5: 'IV+', 5: 'V',
  }
  return map[v] ?? String(v)
}

function gaugeRelLabel(rel: string | null | undefined): string {
  switch (rel) {
    case 'upstream_indicator':   return 'Upstream gauge'
    case 'downstream_indicator': return 'Downstream gauge'
    case 'tributary':            return 'Tributary gauge'
    default:                     return 'Flow gauge'
  }
}

function accessTypeLabel(type: string): string {
  return { put_in: 'Put-in', take_out: 'Take-out', shuttle_drop: 'Shuttle', intermediate: 'Access', parking: 'Parking', camp: 'Camp' }[type] ?? type
}

function accessColor(type: string): string {
  return { put_in: '#22c55e', take_out: '#ef4444', shuttle_drop: '#a855f7', intermediate: '#94a3b8', parking: '#dc2626', camp: '#f59e0b' }[type] ?? '#94a3b8'
}

function accessIcon(type: string): string {
  return { put_in: '↓', take_out: '↑', shuttle_drop: 'S', intermediate: '◆', parking: 'P', camp: '⛺' }[type] ?? '·'
}

function rebuildLayers() {
  if (!map || !mapReady.value) return
  for (const id of [
    'centerline-glow', 'centerline',
    'other-reaches-hit', 'other-reaches-hover', 'other-reaches-line', 'other-reaches-glow', 'other-reaches',
  ]) {
    if (map.getLayer(id)) map.removeLayer(id)
    if (map.getSource(id)) map.removeSource(id)
  }
  for (const m of allMarkers) m.remove()
  allMarkers = []
  markerEls.clear()
  addLayers()
  fitBounds()
  fetchNearbyReaches()
}

// Re-add layers when data changes (e.g. after KMZ import refreshes the page, or after OSM centerline fetch)
watch(allFeatures, rebuildLayers, { deep: true })
watch(() => props.centerline, rebuildLayers, { deep: true })
watch(() => props.gauges, rebuildLayers, { deep: true })
</script>

<style>
.maplibregl-popup {
  z-index: 10;
}
.maplibregl-popup-content {
  border-radius: 8px !important;
  padding: 0 !important;
  box-shadow: 0 4px 16px rgba(0,0,0,0.15) !important;
}
.map-popup {
  padding: 10px 14px;
  font-family: inherit;
  max-width: 220px;
}
.map-popup-title {
  font-weight: 600;
  font-size: 0.875rem;
  color: #111827;
  margin: 0 0 4px;
}
.map-popup-class {
  font-weight: 400;
  color: #6b7280;
  font-size: 0.8rem;
}
.map-popup-desc {
  font-size: 0.8rem;
  color: #4b5563;
  margin: 0;
  line-height: 1.4;
}
.reach-map-rapid-label {
  cursor: pointer;
  font-family: system-ui, sans-serif;
  font-size: 11px;
  font-weight: 700;
  /* Slightly off-white fill — softer than pure white */
  color: #e5e7eb;
  white-space: nowrap;
  user-select: none;
  letter-spacing: 0.02em;
  /* Dark gray stroke (was pure black) + subtle blue glow */
  text-shadow:
    -1px -1px 0 #374151,  1px -1px 0 #374151,
    -1px  1px 0 #374151,  1px  1px 0 #374151,
     0   -1px 0 #374151,  0    1px 0 #374151,
    -1px  0   0 #374151,  1px  0   0 #374151;
  transition: filter 0.15s, font-size 0.15s;
}
.reach-map-rapid-label:hover {
  filter: drop-shadow(0 0 5px #60a5fa) drop-shadow(0 0 10px rgba(59,130,246,0.65));
}
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
  border-bottom-color: rgba(17, 24, 39, 0.92) !important;
}
</style>
