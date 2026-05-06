<template>
  <div class="pb-20">
    <!-- Step 1: pick anchor -->
    <template v-if="!anchorSnap && !anchorSnapping">
      <div class="mb-3 rounded-lg bg-blue-50 dark:bg-blue-950/60 border border-blue-200 dark:border-blue-800 px-3 py-2.5 text-xs text-blue-800 dark:text-blue-200">
        <span class="font-medium">Tap the river on the map</span> near the reach start point. We'll snap to the nearest NHD flowline.
      </div>
    </template>
    <template v-else-if="anchorSnapping">
      <div class="mb-3 rounded-lg bg-blue-50 dark:bg-blue-950/60 border border-blue-200 dark:border-blue-800 px-3 py-2 text-xs text-blue-700 dark:text-blue-300 animate-pulse">
        Snapping to NHD…
      </div>
    </template>
    <template v-else-if="anchorSnap">
      <div class="mb-3 flex items-center gap-3 px-3 py-2 rounded-lg bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 text-xs">
        <span class="w-2.5 h-2.5 rounded-full bg-blue-600 shrink-0" />
        <span class="flex-1 font-medium text-blue-800 dark:text-blue-200">
          <template v-if="anchorSnap.name">{{ anchorSnap.name }}</template>
          <template v-else>ComID {{ anchorSnap.comid }}</template>
        </span>
        <UButton size="xs" variant="ghost" color="neutral" @click="resetToPickMode">Pick another point</UButton>
        <UButton size="xs" variant="ghost" color="error" @click="reset">Clear</UButton>
      </div>
    </template>

    <!-- Step 2: guide text for ComID selection -->
    <div v-if="anchorSnap && !upComID && !gaugeSelectMode" class="mb-3 rounded-lg bg-gray-50 dark:bg-gray-800/60 border border-gray-200 dark:border-gray-700 px-3 py-2 text-xs text-gray-600 dark:text-gray-300">
      Click the <strong>put-in</strong> flowline segment on the map.
    </div>
    <div v-else-if="upComID && !downComID && !gaugeSelectMode" class="mb-3 rounded-lg bg-gray-50 dark:bg-gray-800/60 border border-gray-200 dark:border-gray-700 px-3 py-2 text-xs text-gray-600 dark:text-gray-300">
      Now click the <strong>take-out</strong> flowline segment.
    </div>

    <!-- ComID + gauge slot selector -->
    <div v-if="tributaries" class="flex items-center gap-2 mb-3 text-xs flex-wrap">
      <span class="text-gray-500 shrink-0">Click for:</span>
      <button
        class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
        :class="comIDSlot === 'up' && !gaugeSelectMode ? 'border-green-500 bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
        @click="comIDSlot = 'up'; gaugeSelectMode = false"
      >
        <span class="w-2 h-2 rounded-full bg-green-500 shrink-0" />
        Put-in<template v-if="upComID"> · <span class="font-mono">{{ upComID }}</span></template>
      </button>
      <button
        class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
        :class="comIDSlot === 'down' && !gaugeSelectMode ? 'border-red-500 bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
        @click="comIDSlot = 'down'; gaugeSelectMode = false"
      >
        <span class="w-2 h-2 rounded-full bg-red-500 shrink-0" />
        Take-out<template v-if="downComID"> · <span class="font-mono">{{ downComID }}</span></template>
      </button>
      <button
        class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
        :class="gaugeSelectMode ? 'border-amber-500 bg-amber-50 dark:bg-amber-950 text-amber-700 dark:text-amber-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
        @click="gaugeSelectMode = !gaugeSelectMode; if (!gaugeSelectMode) pendingGauge = null"
      >
        <span class="w-2 h-2 rounded-full bg-amber-500 shrink-0" />
        <template v-if="gaugeSelectMode">Cancel gauge</template>
        <template v-else-if="pendingGauge">Gauge · <span class="font-mono">{{ pendingGauge.externalId }}</span></template>
        <template v-else>Gauge <span class="text-gray-400 font-normal">(optional)</span></template>
      </button>
    </div>

    <!-- Pending gauge card -->
    <div v-if="pendingGauge" class="flex items-center gap-2 px-3 py-2 mb-3 rounded-lg bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800 text-xs">
      <span class="w-2 h-2 rounded-full bg-amber-500 shrink-0" />
      <span class="flex-1 truncate font-medium text-amber-800 dark:text-amber-200">{{ pendingGauge.name || pendingGauge.externalId }}</span>
      <span class="font-mono text-amber-600 dark:text-amber-400 shrink-0 uppercase">{{ pendingGauge.source }} {{ pendingGauge.externalId }}</span>
      <button class="text-amber-400 hover:text-amber-600 dark:hover:text-amber-300 shrink-0 px-1" @click="pendingGauge = null">✕</button>
    </div>

    <!-- Map -->
    <ClientOnly>
      <NHDExplorerMap
        :upstream-flowlines="tributaries"
        :downstream-flowlines="downstreamFlowlines"
        :upstream-gauges="nearbyGauges"
        :pick-mode="pickMode"
        :comid-select-mode="!!anchorSnap && !pickMode && !gaugeSelectMode"
        :comid-select-slot="comIDSlot"
        :selected-up-comid="upComID"
        :selected-down-comid="downComID"
        :put-in-pin="putInPin"
        :take-out-pin="takeOutPin"
        :preview-centerline="previewCenterline"
        :gauge-select-mode="gaugeSelectMode"
        :disable-auto-fit="disableAutoFit"
        @pick="onAnchorPick"
        @comid-select="onComIDSelect"
        @gauge-select="onGaugeSelect"
      />
    </ClientOnly>
    <p v-if="downstreamLoading" class="text-xs text-blue-500 dark:text-blue-400 mt-1 animate-pulse">Loading downstream mainstem…</p>

    <!-- Preview centerline controls -->
    <div v-if="upComID && downComID" class="flex items-center gap-2 mt-1 flex-wrap">
      <UButton size="xs" variant="outline" color="neutral" :loading="previewLoading" @click="fetchPreviewCenterline">
        {{ previewCenterline ? 'Refresh preview' : 'Preview centerline' }}
      </UButton>
      <span v-if="previewCenterline" class="text-xs text-blue-600 dark:text-blue-400">Dashed line shows trimmed reach</span>
      <span v-if="riverNameFetching" class="text-xs text-gray-400 animate-pulse">Looking up river…</span>
    </div>

    <!-- River name auto-detected -->
    <div v-if="upComID && downComID && form.riverName && !riverNameFetching" class="mt-2 flex items-center gap-2 px-3 py-2 rounded-lg bg-emerald-50 dark:bg-emerald-950/50 border border-emerald-200 dark:border-emerald-800 text-xs text-emerald-800 dark:text-emerald-200">
      <svg class="w-3.5 h-3.5 shrink-0 text-emerald-500" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/></svg>
      <span>Looks like <strong>{{ form.riverName }}</strong><template v-if="gnisId"> · GNIS {{ gnisId }}</template></span>
    </div>

    <!-- Reach form — shown once both ComIDs selected -->
    <div v-if="upComID && downComID" class="mt-4 space-y-3 rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900">
      <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-100">Reach details</h3>

      <!-- River name -->
      <div>
        <label class="block text-xs text-gray-500 mb-1">River name <span class="text-gray-300">(from NHD)</span></label>
        <div class="flex items-center gap-2">
          <input
            v-model="form.riverName"
            :readonly="!riverNameOverride"
            class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm"
            :class="riverNameOverride ? '' : 'text-gray-400 dark:text-gray-500 cursor-default'"
            placeholder="Auto-filled from NHD"
          />
          <button class="text-xs text-blue-500 hover:text-blue-400 shrink-0" @click="riverNameOverride = !riverNameOverride">
            {{ riverNameOverride ? 'Lock' : 'Override' }}
          </button>
        </div>
      </div>

      <!-- Reach name -->
      <div>
        <label class="block text-xs text-gray-500 mb-1">Reach name <span class="text-red-400">*</span></label>
        <input v-model="form.name" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Upper Canyon" />
      </div>

      <!-- Notes -->
      <div>
        <label class="block text-xs text-gray-500 mb-1">Notes <span class="text-gray-300">(optional)</span></label>
        <textarea
          v-model="form.note"
          rows="2"
          class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm resize-y"
          placeholder="Beta, access notes, hazards…"
        />
      </div>

      <!-- Flow bands -->
      <div class="rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50 p-3 space-y-2">
        <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Flow bands (CFS) <span class="font-normal text-gray-400 normal-case">optional</span></p>
        <div class="space-y-2">
          <div v-for="band in flowBands" :key="band.key" class="flex items-center gap-3">
            <span class="w-16 text-xs font-medium" :class="band.labelClass">{{ band.label }}</span>
            <input
              v-model.number="form.flowRanges[band.key].min"
              type="number" min="0"
              class="w-24 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
              :placeholder="band.showMin ? 'min' : '—'"
              :disabled="!band.showMin"
            />
            <span class="text-gray-400 text-xs">–</span>
            <input
              v-model.number="form.flowRanges[band.key].max"
              type="number" min="0"
              class="w-24 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
              :placeholder="band.showMax ? 'max' : '—'"
              :disabled="!band.showMax"
            />
            <span class="text-xs text-gray-400">cfs</span>
          </div>
        </div>
      </div>

      <div v-if="submitError" class="text-xs text-red-500">{{ submitError }}</div>

      <div class="flex gap-2 justify-end pt-1">
        <UButton size="sm" variant="ghost" color="neutral" @click="emit('cancel')">Cancel</UButton>
        <UButton size="sm" :disabled="!form.name.trim()" :loading="saving" @click="submit">Save reach</UButton>
      </div>
    </div>

    <!-- Error from anchor snap -->
    <p v-if="snapError" class="text-xs text-red-500 mt-2">{{ snapError }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'

const emit = defineEmits<{
  created: [slug: string]
  cancel:  []
}>()

const { getToken } = useAuth()
const { apiBase }  = useRuntimeConfig().public

const pickMode            = ref(false)
const anchorSnapping      = ref(false)
const anchorSnap          = ref<{ comid: string; name: string } | null>(null)
const tributaries         = ref<object | null>(null)
const downstreamFlowlines = ref<object | null>(null)
const downstreamLoading   = ref(false)
const comIDSlot           = ref<'up' | 'down'>('up')
const upComID             = ref<string | null>(null)
const downComID           = ref<string | null>(null)
const startLat            = ref<number | null>(null)
const startLng            = ref<number | null>(null)
const endLat              = ref<number | null>(null)
const endLng              = ref<number | null>(null)
const riverNameOverride   = ref(false)
const riverNameFetching   = ref(false)
const gnisId              = ref('')
const snapError           = ref('')
const submitError         = ref('')
const saving              = ref(false)
const previewCenterline   = ref<object | null>(null)
const previewLoading      = ref(false)
const previewGeoJSON      = ref<object | null>(null)

// Gauge selection
const gaugeSelectMode = ref(false)
const nearbyGauges    = ref<object | null>(null)
const pendingGauge    = ref<{ externalId: string; source: string; name: string; lat: number; lng: number } | null>(null)

// Prevent zoom-out when tributaries load after anchor snap.
// Only auto-fit when both ComIDs are set (reach extent known).
const disableAutoFit = computed(() => !!anchorSnap.value && !(upComID.value && downComID.value))

const form = ref({
  name:      '',
  riverName: '',
  note:      '',
  flowRanges: {
    low:     { min: null as number | null, max: null as number | null },
    running: { min: null as number | null, max: null as number | null },
    high:    { min: null as number | null, max: null as number | null },
  },
})

const flowBands = [
  { key: 'low',     label: 'Too Low',  labelClass: 'text-gray-400',    showMin: false, showMax: true  },
  { key: 'running', label: 'Runnable', labelClass: 'text-emerald-500', showMin: true,  showMax: true  },
  { key: 'high',    label: 'High',     labelClass: 'text-sky-400',     showMin: true,  showMax: false },
] as const

const putInPin = computed(() =>
  startLat.value != null && startLng.value != null
    ? { lat: startLat.value, lng: startLng.value, label: 'Put-in' }
    : null
)
const takeOutPin = computed(() =>
  endLat.value != null && endLng.value != null
    ? { lat: endLat.value, lng: endLng.value, label: 'Take-out' }
    : null
)

onMounted(() => { pickMode.value = true })

function resetToPickMode() {
  pickMode.value         = true
  anchorSnap.value       = null
  tributaries.value      = null
  downstreamFlowlines.value = null
  nearbyGauges.value     = null
  gaugeSelectMode.value  = false
  pendingGauge.value     = null
  upComID.value = null; downComID.value = null
  startLat.value = null; startLng.value = null
  endLat.value = null;   endLng.value   = null
  previewCenterline.value = null
  previewGeoJSON.value    = null
  comIDSlot.value = 'up'
}

function reset() {
  resetToPickMode()
  anchorSnapping.value = false
  downstreamLoading.value = false
  riverNameOverride.value = false
  riverNameFetching.value = false
  gnisId.value      = ''
  snapError.value   = ''
  submitError.value = ''
  saving.value      = false
  previewLoading.value = false
  form.value = {
    name: '', riverName: '', note: '',
    flowRanges: {
      low:     { min: null, max: null },
      running: { min: null, max: null },
      high:    { min: null, max: null },
    },
  }
}

watch(upComID, async (comid) => {
  downstreamFlowlines.value = null
  if (!comid) return
  downstreamLoading.value = true
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/nldi/downstream?comid=${comid}&distance=800`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (res.ok) downstreamFlowlines.value = (await res.json()).downstream_flowlines
  } catch { /* non-fatal */ }
  finally { downstreamLoading.value = false }
})

watch([upComID, downComID], async ([up, down]) => {
  if (!up || !down) return
  fetchPreviewCenterline()
  fetchRiverName()
})

async function fetchNearbyGauges(lat: number, lng: number, comid?: string) {
  try {
    const token = await getToken()
    const comidParam = comid ? `&comid=${comid}` : ''
    const res = await fetch(
      `${apiBase}/api/v1/nldi/nearby-gauges?lat=${lat}&lng=${lng}&distance=100${comidParam}`,
      { headers: token ? { Authorization: `Bearer ${token}` } : {} },
    )
    if (res.ok) nearbyGauges.value = await res.json()
  } catch { /* non-fatal */ }
}

async function onAnchorPick(lat: number, lng: number) {
  pickMode.value       = false
  anchorSnapping.value = true
  anchorSnap.value     = null
  tributaries.value    = null
  snapError.value      = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/nldi/upstream-tributaries?lat=${lat}&lng=${lng}&distance=50`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) { snapError.value = `Snap failed: HTTP ${res.status}`; return }
    const data = await res.json()
    anchorSnap.value  = { comid: data.snap.comid, name: data.snap.name ?? '' }
    tributaries.value = data.tributaries
    if (!riverNameOverride.value && data.snap.name) {
      form.value.riverName = data.snap.name
    }
    // Fetch nearby gauges immediately so they're ready on the map.
    fetchNearbyGauges(lat, lng, data.snap.comid)
  } catch (e: any) {
    snapError.value = e.message ?? 'Snap failed'
  } finally {
    anchorSnapping.value = false
  }
}

function onComIDSelect(comid: string, lat: number, lng: number) {
  if (comIDSlot.value === 'up') {
    upComID.value = comid
    startLat.value = lat; startLng.value = lng
    comIDSlot.value = 'down'
  } else {
    downComID.value = comid
    endLat.value = lat; endLng.value = lng
  }
}

function onGaugeSelect(externalId: string, source: string, name: string, lat: number, lng: number) {
  gaugeSelectMode.value = false
  pendingGauge.value    = { externalId, source, name, lat, lng }
}

async function fetchRiverName() {
  const comid = upComID.value ?? anchorSnap.value?.comid
  if (!comid) return
  riverNameFetching.value = true
  try {
    const token = await getToken()
    const lat = startLat.value
    const lng = startLng.value
    const coordParams = lat != null && lng != null ? `&lat=${lat}&lng=${lng}` : ''
    const res = await fetch(`${apiBase}/api/v1/nldi/river-name?comid=${comid}${coordParams}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) return
    const data = await res.json()
    if (data.river_name && !riverNameOverride.value) form.value.riverName = data.river_name
    if (data.gnis_id) gnisId.value = data.gnis_id
  } catch { /* non-fatal */ }
  finally { riverNameFetching.value = false }
}

async function fetchPreviewCenterline() {
  if (!upComID.value || !downComID.value) return
  previewLoading.value    = true
  previewCenterline.value = null
  previewGeoJSON.value    = null
  try {
    const token = await getToken()
    let url = `${apiBase}/api/v1/nldi/preview-centerline?up_comid=${upComID.value}&down_comid=${downComID.value}`
    if (startLat.value != null && startLng.value != null && endLat.value != null && endLng.value != null)
      url += `&start_lat=${startLat.value}&start_lng=${startLng.value}&end_lat=${endLat.value}&end_lng=${endLng.value}`
    const res = await fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
    if (res.ok) {
      const data = await res.json()
      previewCenterline.value = data.geojson
      previewGeoJSON.value    = data.geojson
    }
  } catch { /* non-fatal */ }
  finally { previewLoading.value = false }
}

async function submit() {
  if (!form.value.name.trim() || !upComID.value || !downComID.value) return
  saving.value      = true
  submitError.value = ''
  try {
    const token = await getToken()
    const headers = { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) }

    const ranges = form.value.flowRanges
    const hasRanges = Object.values(ranges).some(b => b.min != null || b.max != null)

    const body: Record<string, any> = {
      name:       form.value.name.trim(),
      up_comid:   upComID.value,
      down_comid: downComID.value,
      put_in:    { lat: startLat.value!, lng: startLng.value! },
      take_out:  { lat: endLat.value!,   lng: endLng.value!   },
    }
    if (form.value.riverName.trim()) body.river_name = form.value.riverName.trim()
    if (gnisId.value)                body.gnis_id    = gnisId.value
    if (form.value.note.trim())      body.note       = form.value.note.trim()
    if (hasRanges) {
      body.flow_ranges = {
        low:     { min_value: null,               max_value: ranges.low.max      },
        running: { min_value: ranges.running.min, max_value: ranges.running.max  },
        high:    { min_value: ranges.high.min,    max_value: null                },
      }
    }

    const res = await fetch(`${apiBase}/api/v1/me/reaches`, {
      method: 'POST', headers, body: JSON.stringify(body),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error ?? `HTTP ${res.status}`)
    const slug: string = data.slug

    // Store centerline.
    if (previewGeoJSON.value) {
      await fetch(`${apiBase}/api/v1/me/reaches/${slug}/centerline`, {
        method: 'POST', headers,
        body: JSON.stringify({ geojson: previewGeoJSON.value }),
      }).catch(() => { /* non-fatal */ })
    }

    // Attach gauge if selected — resolve external_id+source → UUID via search.
    if (pendingGauge.value) {
      const { externalId, source } = pendingGauge.value
      const searchRes = await fetch(`${apiBase}/api/v1/gauges/search?q=${encodeURIComponent(externalId)}&limit=10`)
      const searchData = await searchRes.json()
      const feature = (searchData.features ?? []).find(
        (f: any) => f.properties.external_id === externalId && f.properties.source === source
      )
      if (feature) {
        await fetch(`${apiBase}/api/v1/me/reaches/${slug}/gauge`, {
          method: 'PUT', headers,
          body: JSON.stringify({ gauge_id: feature.properties.id }),
        }).catch(() => { /* non-fatal */ })
      }
    }

    emit('created', slug)
  } catch (e: any) {
    submitError.value = e.message ?? 'Failed to create reach.'
  } finally {
    saving.value = false
  }
}
</script>
