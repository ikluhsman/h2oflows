<template>
  <!-- Map + feature list side-by-side on sm+, stacked on mobile -->
  <div class="rounded-xl overflow-hidden border border-gray-200 dark:border-gray-700 flex flex-col sm:flex-row sm:h-120">

    <!-- MapLibre container — ref is on THIS element so MapLibre reads its own clientHeight -->
    <div
      ref="container"
      class="relative flex-1 min-h-56 sm:min-h-0 bg-gray-100 dark:bg-gray-800"
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
        <button
          v-for="a in accessFeatures"
          :key="a.id"
          class="w-full flex items-center gap-2 px-3 py-1.5 text-left hover:bg-gray-50 dark:hover:bg-gray-800/60 transition-colors text-xs"
          :class="selectedId === a.id ? 'bg-gray-100 dark:bg-gray-800' : ''"
          @click="selectFeature(a.id, a.lng, a.lat)"
        >
          <span class="shrink-0 w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold text-white"
            :style="{ background: accessColor(a.type) }">
            {{ accessIcon(a.type) }}
          </span>
          <span class="truncate text-gray-700 dark:text-gray-300">{{ a.label }}</span>
        </button>
      </template>

      <!-- Rapids group -->
      <template v-if="rapidFeatures.length > 0">
        <p class="px-3 pt-2.5 pb-1 text-[10px] font-bold uppercase tracking-wider text-gray-400">Rapids</p>
        <button
          v-for="r in rapidFeatures"
          :key="r.id"
          class="w-full flex items-center gap-2 px-3 py-1.5 text-left hover:bg-gray-50 dark:hover:bg-gray-800/60 transition-colors text-xs"
          :class="selectedId === r.id ? 'bg-gray-100 dark:bg-gray-800' : ''"
          @click="selectFeature(r.id, r.lng, r.lat)"
        >
          <span class="shrink-0 w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold text-white"
            :style="{ background: r.isSurf ? '#3b82f6' : '#f97316' }">
            {{ r.isSurf ? '🌊' : '~' }}
          </span>
          <span class="truncate text-gray-700 dark:text-gray-300">{{ r.label }}</span>
          <span v-if="r.classLabel" class="shrink-0 text-[10px] text-gray-400 font-medium">{{ r.classLabel }}</span>
        </button>
      </template>
    </div>
  </div>

  <!-- Selected feature detail panel -->
  <div
    v-if="selectedFeature"
    class="border-t border-gray-200 dark:border-gray-700 px-4 py-3 flex items-start gap-3"
  >
    <div class="flex-1 min-w-0">
      <p class="text-sm font-semibold text-gray-800 dark:text-gray-100">{{ selectedFeature.title }}</p>
      <p v-if="selectedFeature.subtitle" class="text-xs text-gray-400 mt-0.5">{{ selectedFeature.subtitle }}</p>
      <p v-if="selectedFeature.desc" class="text-sm text-gray-600 dark:text-gray-400 mt-1 leading-relaxed">{{ selectedFeature.desc }}</p>
      <p v-else class="text-xs text-gray-400 mt-1 italic">No description available.</p>
    </div>
    <button
      class="shrink-0 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 text-lg leading-none mt-0.5"
      @click="selectedId = null"
    >×</button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

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

const props = defineProps<{
  name?: string
  classMax?: number | null
  centerline?: any
  rapids: RapidFeature[]
  access: AccessFeature[]
  gaugeLng?: number | null
  gaugeLat?: number | null
}>()

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
    title:    rapid.label + (rapid.classLabel ? ` · Class ${rapid.classLabel}` : ''),
    subtitle: rapid.isSurf ? 'Surf wave' : null,
    desc:     rapid.desc || null,
  }
  const access = accessFeatures.value.find(a => a.id === selectedId.value)
  if (access) return {
    title:    accessTypeLabel(access.type),
    subtitle: access.label !== accessTypeLabel(access.type) ? access.label : null,
    desc:     access.notes || null,
  }
  return null
})

const hasCoords = computed(() =>
  props.centerline || allFeatures.value.length > 0 || props.gaugeLng != null
)

// ── Map state ─────────────────────────────────────────────────────────────────

const container  = ref<HTMLDivElement>()
const mapReady   = ref(false)
const selectedId = ref<string | null>(null)
const basemap = ref<'street' | 'topo' | 'satellite'>('street')
const BASEMAP_OPTIONS = [
  { value: 'street',    label: 'Street'    },
  { value: 'topo',      label: 'Topo'      },
  { value: 'satellite', label: 'Satellite' },
] as const
let map: maplibregl.Map | null = null
let clickPopup: maplibregl.Popup | null = null
let allMarkers: maplibregl.Marker[] = []
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
    center:   [props.gaugeLng ?? -105.5, props.gaugeLat ?? 39.2],
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
  })

  map.on('error', (e) => { console.warn('[ReachMap]', e.error?.message ?? e) })
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

  // Rapid markers
  for (const r of rapidFeatures.value) {
    const color = r.isSurf ? '#3b82f6' : '#f97316'
    const el = makePinEl(color, null, r.classLabel ?? '~', r.id)
    el.title = `${r.label}${r.classLabel ? ' · Class ' + r.classLabel : ''}`
    el.addEventListener('mouseenter', () => showTooltip(el, el.title, [r.lng, r.lat]))
    el.addEventListener('mouseleave', () => tooltip.remove())
    el.addEventListener('click', () => {
      clickPopup?.remove()
      const title = `${r.label}${r.classLabel ? ` <span class="map-popup-class">Class ${r.classLabel}</span>` : ''}`
      clickPopup = new maplibregl.Popup({ offset: [0, -32] })
        .setLngLat([r.lng, r.lat])
        .setHTML(`<div class="map-popup"><p class="map-popup-title">${title}</p>${r.desc ? `<p class="map-popup-desc">${r.desc}</p>` : ''}</div>`)
        .addTo(map!)
      setSelectedMarker(r.id)
    })
    const marker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
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

function makePinEl(color: string, imgUrl: string | null, label: string, id: string): HTMLElement {
  const el = document.createElement('div')
  el.dataset.markerId = id
  el.dataset.pinColor = color
  el.style.cssText = 'cursor:pointer;filter:drop-shadow(0 2px 4px rgba(0,0,0,0.4));transition:filter 0.12s'
  const inner = imgUrl
    ? `<image href="${imgUrl}" x="4" y="4" width="20" height="20" clip-path="circle(10px at 10px 10px)"/>`
    : `<text x="14" y="17" text-anchor="middle" dominant-baseline="middle"
        font-size="10" font-weight="700" font-family="system-ui,sans-serif" fill="white">${label}</text>`
  el.innerHTML = `<svg width="28" height="36" viewBox="0 0 28 36" xmlns="http://www.w3.org/2000/svg">
    <path d="M14 1C7.1 1 1.5 6.6 1.5 13.5c0 8.7 12.5 20.5 12.5 20.5S26.5 22.2 26.5 13.5C26.5 6.6 20.9 1 14 1z"
      fill="${color}" stroke="white" stroke-width="1.5"/>
    ${inner}
  </svg>`
  return el
}

function accessIconUrl(type: string): string | null {
  // Map access types to the KMZ icon images (served from /icons/)
  const map: Record<string, string> = {
    put_in:       '/icons/kmz-icon-1.png',   // blue kayaker
    take_out:     '/icons/kmz-icon-5.png',   // red kayaker
    intermediate: '/icons/kmz-icon-4.png',   // green kayaker
    shuttle_drop: '/icons/kmz-icon-2.png',   // red P (parking)
  }
  return map[type] ?? null
}

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
    coords.push(line[0], line[line.length - 1])
  }
  if (coords.length === 0) {
    if (props.gaugeLng != null && props.gaugeLat != null) {
      map.setCenter([props.gaugeLng, props.gaugeLat]); map.setZoom(12)
    }
    return
  }
  if (coords.length === 1) { map.setCenter(coords[0]); map.setZoom(14); return }
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
  if (maxRating == null) return '#6b7280'  // gray — no class data
  if (maxRating < 2.5)   return '#16a34a'  // green  — Class I–II
  if (maxRating < 4.0)   return '#3b82f6'  // blue   — Class III/III+
  if (maxRating < 5.0)   return '#111827'  // black  — Class IV (incl. IV+)
  return '#111827'                          // black  — Class V
}

function formatClass(v: number): string {
  const map: Record<number, string> = {
    1: 'I', 1.5: 'I+', 2: 'II', 2.5: 'II+',
    3: 'III', 3.5: 'III+', 4: 'IV', 4.5: 'IV+', 5: 'V',
  }
  return map[v] ?? String(v)
}

function accessTypeLabel(type: string): string {
  return { put_in: 'Put-in', take_out: 'Take-out', shuttle_drop: 'Shuttle', intermediate: 'Access', camp: 'Camp' }[type] ?? type
}

function accessColor(type: string): string {
  return { put_in: '#22c55e', take_out: '#ef4444', shuttle_drop: '#a855f7', intermediate: '#94a3b8', camp: '#f59e0b' }[type] ?? '#94a3b8'
}

function accessIcon(type: string): string {
  return { put_in: '↓', take_out: '↑', shuttle_drop: 'S', intermediate: '◆', camp: '⛺' }[type] ?? '·'
}

function rebuildLayers() {
  if (!map || !mapReady.value) return
  for (const id of ['centerline-glow', 'centerline']) {
    if (map.getLayer(id)) map.removeLayer(id)
    if (map.getSource(id)) map.removeSource(id)
  }
  for (const m of allMarkers) m.remove()
  allMarkers = []
  markerEls.clear()
  addLayers()
  fitBounds()
}

// Re-add layers when data changes (e.g. after KMZ import refreshes the page, or after OSM centerline fetch)
watch(allFeatures, rebuildLayers)
watch(() => props.centerline, rebuildLayers)
</script>

<style>
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
