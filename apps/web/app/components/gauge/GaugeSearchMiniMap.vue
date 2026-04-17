<template>
  <div ref="container" class="w-full h-48 bg-gray-100 dark:bg-gray-800" />
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import type { WatchedGauge } from '~/stores/watchlist'

type SearchGauge = Omit<WatchedGauge, 'watchState' | 'activeSince'>

const props = defineProps<{
  gauges: SearchGauge[]
  highlightId: string | null
}>()

const container = ref<HTMLElement | null>(null)
let map: maplibregl.Map | null = null
const markers: maplibregl.Marker[] = []

function clearMarkers() {
  markers.forEach(m => m.remove())
  markers.length = 0
}

function syncMarkers() {
  if (!map) return
  clearMarkers()

  const bounds = new maplibregl.LngLatBounds()
  let hasPoints = false

  for (const g of props.gauges) {
    if (g.lng == null || g.lat == null) continue
    hasPoints = true
    bounds.extend([g.lng, g.lat])

    const isHighlighted = props.highlightId === g.id
    const el = document.createElement('div')
    el.style.cssText = `
      width: ${isHighlighted ? '12px' : '8px'};
      height: ${isHighlighted ? '12px' : '8px'};
      border-radius: 50%;
      background: ${isHighlighted ? '#3b82f6' : '#6b7280'};
      border: 2px solid white;
      box-shadow: 0 1px 3px rgba(0,0,0,0.3);
      transition: all 150ms ease;
    `

    const marker = new maplibregl.Marker({ element: el })
      .setLngLat([g.lng, g.lat])
      .addTo(map)
    markers.push(marker)
  }

  if (hasPoints) {
    map.fitBounds(bounds, { padding: 30, maxZoom: 10, duration: 300 })
  }
}

onMounted(() => {
  if (!container.value) return

  map = new maplibregl.Map({
    container: container.value,
    style: {
      version: 8,
      sources: {
        topo: {
          type: 'raster',
          tiles: ['https://server.arcgisonline.com/ArcGIS/rest/services/World_Topo_Map/MapServer/tile/{z}/{y}/{x}'],
          tileSize: 256,
          maxzoom: 15,
        },
      },
      layers: [
        { id: 'topo-tiles', type: 'raster', source: 'topo' },
      ],
    },
    bounds: [-109.1, 36.9, -102.0, 41.1] as [number, number, number, number],
    fitBoundsOptions: { padding: 20 },
    attributionControl: false,
    interactive: false,
  })

  map.on('load', syncMarkers)
})

onUnmounted(() => {
  clearMarkers()
  map?.remove()
  map = null
})

watch(() => props.gauges, syncMarkers, { deep: true })
watch(() => props.highlightId, syncMarkers)
</script>
