<template>
  <div ref="container" class="w-full h-72 rounded-xl overflow-hidden bg-gray-100 dark:bg-gray-800" />
</template>

<script setup lang="ts">
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

interface RapidFeature {
  id: string
  name: string
  class_rating: number | null
  description: string | null
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
  centerline?: any       // GeoJSON geometry (LineString) or null
  rapids: RapidFeature[]
  access: AccessFeature[]
}>()

const container = ref<HTMLDivElement>()
let map: maplibregl.Map | null = null

// Placemarks that have valid coords
const rapidPoints = computed(() =>
  props.rapids.filter(r => r.lng != null && r.lat != null)
)
const accessPoints = computed(() =>
  props.access.filter(a => a.water_lng != null && a.water_lat != null)
)

const hasFeatures = computed(() =>
  props.centerline || rapidPoints.value.length > 0 || accessPoints.value.length > 0
)

onMounted(() => {
  if (!container.value || !hasFeatures.value) return

  map = new maplibregl.Map({
    container: container.value,
    style: 'https://tiles.openfreemap.org/styles/liberty',
    zoom: 12,
    center: [-105.5, 39.0],
    attributionControl: false,
  })

  map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-right')
  map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-right')

  map.on('load', () => {
    if (!map) return

    // ── Centerline ──────────────────────────────────────────────────────────
    if (props.centerline) {
      map.addSource('centerline', {
        type: 'geojson',
        data: { type: 'Feature', geometry: props.centerline, properties: {} },
      })
      map.addLayer({
        id: 'centerline',
        type: 'line',
        source: 'centerline',
        paint: { 'line-color': '#3b82f6', 'line-width': 3, 'line-opacity': 0.85 },
      })
    }

    // ── Access points ────────────────────────────────────────────────────────
    map.addSource('access', {
      type: 'geojson',
      data: {
        type: 'FeatureCollection',
        features: accessPoints.value.map(a => ({
          type: 'Feature',
          geometry: { type: 'Point', coordinates: [a.water_lng!, a.water_lat!] },
          properties: { id: a.id, name: a.name ?? '', type: a.access_type, notes: a.notes ?? '' },
        })),
      },
    })
    map.addLayer({
      id: 'access-circles',
      type: 'circle',
      source: 'access',
      paint: {
        'circle-radius': 9,
        'circle-stroke-width': 2,
        'circle-stroke-color': '#fff',
        'circle-color': [
          'match', ['get', 'type'],
          'put_in',       '#22c55e',
          'take_out',     '#ef4444',
          'shuttle_drop', '#a855f7',
          '#94a3b8',
        ],
      },
    })

    // ── Rapids ───────────────────────────────────────────────────────────────
    map.addSource('rapids', {
      type: 'geojson',
      data: {
        type: 'FeatureCollection',
        features: rapidPoints.value.map(r => ({
          type: 'Feature',
          geometry: { type: 'Point', coordinates: [r.lng!, r.lat!] },
          properties: {
            id: r.id,
            name: r.name,
            class: r.class_rating != null ? formatClass(r.class_rating) : null,
            description: r.description ?? '',
          },
        })),
      },
    })
    map.addLayer({
      id: 'rapid-circles',
      type: 'circle',
      source: 'rapids',
      paint: {
        'circle-radius': 7,
        'circle-color': '#f97316',
        'circle-stroke-width': 2,
        'circle-stroke-color': '#fff',
      },
    })

    // ── Hover tooltip ────────────────────────────────────────────────────────
    const tooltip = new maplibregl.Popup({
      closeButton: false,
      closeOnClick: false,
      offset: 12,
      className: 'reach-map-tooltip',
    })

    for (const layer of ['rapid-circles', 'access-circles']) {
      map.on('mouseenter', layer, e => {
        if (!map || !e.features?.length) return
        map.getCanvas().style.cursor = 'pointer'
        const p = e.features[0].properties
        const label = layer === 'rapid-circles'
          ? `<strong>${p.name}</strong>${p.class ? ` · Class ${p.class}` : ''}`
          : `<strong>${accessTypeLabel(p.type)}</strong>${p.name ? ` · ${p.name}` : ''}`
        tooltip.setLngLat(e.lngLat).setHTML(label).addTo(map)
      })
      map.on('mouseleave', layer, () => {
        map?.getCanvas().setAttribute('style', '')
        tooltip.remove()
      })
    }

    // ── Click popup (full detail) ─────────────────────────────────────────────
    map.on('click', 'rapid-circles', e => {
      if (!map || !e.features?.length) return
      const p = e.features[0].properties
      const html = [
        `<div class="map-popup">`,
        `<p class="map-popup-title">${p.name}${p.class ? ` <span class="map-popup-class">Class ${p.class}</span>` : ''}</p>`,
        p.description ? `<p class="map-popup-desc">${p.description}</p>` : '',
        `</div>`,
      ].join('')
      new maplibregl.Popup({ offset: 12 })
        .setLngLat(e.lngLat)
        .setHTML(html)
        .addTo(map)
    })

    map.on('click', 'access-circles', e => {
      if (!map || !e.features?.length) return
      const p = e.features[0].properties
      const html = [
        `<div class="map-popup">`,
        `<p class="map-popup-title">${accessTypeLabel(p.type)}${p.name ? ` · ${p.name}` : ''}</p>`,
        p.notes ? `<p class="map-popup-desc">${p.notes}</p>` : '',
        `</div>`,
      ].join('')
      new maplibregl.Popup({ offset: 12 })
        .setLngLat(e.lngLat)
        .setHTML(html)
        .addTo(map)
    })

    // ── Fit bounds ───────────────────────────────────────────────────────────
    fitBounds()
  })
})

onUnmounted(() => { map?.remove() })

function fitBounds() {
  if (!map) return
  const coords: [number, number][] = []
  rapidPoints.value.forEach(r => coords.push([r.lng!, r.lat!]))
  accessPoints.value.forEach(a => coords.push([a.water_lng!, a.water_lat!]))
  if (props.centerline?.coordinates) {
    const line = props.centerline.coordinates as [number, number][]
    coords.push(line[0], line[line.length - 1])
  }
  if (coords.length === 0) return
  if (coords.length === 1) { map.setCenter(coords[0]); return }
  const lngs = coords.map(c => c[0])
  const lats = coords.map(c => c[1])
  map.fitBounds(
    [[Math.min(...lngs), Math.min(...lats)], [Math.max(...lngs), Math.max(...lats)]],
    { padding: 48, maxZoom: 14, duration: 0 },
  )
}

function formatClass(v: number) {
  return Number.isInteger(v) ? String(v) : v.toFixed(1)
}

function accessTypeLabel(type: string) {
  const labels: Record<string, string> = {
    put_in: 'Put-in', take_out: 'Take-out',
    shuttle_drop: 'Shuttle', intermediate: 'Access', camp: 'Camp',
  }
  return labels[type] ?? type
}
</script>

<style>
/* Popup styling — plain CSS since MapLibre renders outside Vue scope */
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
}
</style>
