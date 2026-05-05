<template>
  <div>
    <!-- Step 1: anchor not yet set -->
    <template v-if="!authorAnchorSnap && !authorAnchorSnapping">
      <div class="mb-3 rounded-lg bg-blue-50 dark:bg-blue-950/60 border border-blue-200 dark:border-blue-800 px-3 py-2.5 text-xs text-blue-800 dark:text-blue-200">
        <span class="font-medium">Tap the river on the map</span> near the reach start point. We'll snap to the nearest NHD flowline.
      </div>
    </template>
    <template v-else-if="authorAnchorSnapping">
      <div class="mb-3 rounded-lg bg-blue-50 dark:bg-blue-950/60 border border-blue-200 dark:border-blue-800 px-3 py-2 text-xs text-blue-700 dark:text-blue-300 animate-pulse">
        Snapping to NHD…
      </div>
    </template>
    <!-- Anchor snapped — show controls -->
    <template v-else-if="authorAnchorSnap">
      <div class="mb-3 flex items-center gap-3 px-3 py-2 rounded-lg bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 text-xs">
        <span class="w-2.5 h-2.5 rounded-full bg-blue-600 shrink-0" />
        <span class="flex-1 font-medium text-blue-800 dark:text-blue-200">
          <template v-if="authorAnchorSnap.name">{{ authorAnchorSnap.name }}</template>
          <template v-else>ComID {{ authorAnchorSnap.comid }}</template>
        </span>
        <UButton size="xs" variant="ghost" color="neutral" @click="resetToPickMode">Pick another point</UButton>
        <UButton size="xs" variant="ghost" color="error" @click="reset">Clear</UButton>
      </div>
    </template>

    <!-- Step 2: anchor snapped, prompt for ComID selection -->
    <div v-if="authorAnchorSnap && !authorUpComID" class="mb-3 rounded-lg bg-gray-50 dark:bg-gray-800/60 border border-gray-200 dark:border-gray-700 px-3 py-2 text-xs text-gray-600 dark:text-gray-300">
      Click the upstream (put-in) flowline segment on the map. Try satellite view to find the boat ramp.
    </div>
    <div v-else-if="authorUpComID && !authorDownComID" class="mb-3 rounded-lg bg-gray-50 dark:bg-gray-800/60 border border-gray-200 dark:border-gray-700 px-3 py-2 text-xs text-gray-600 dark:text-gray-300">
      Now click the downstream (take-out) flowline segment.
    </div>

    <!-- ComID slot selector -->
    <div v-if="authorTributaries" class="flex items-center gap-3 mb-3 text-xs flex-wrap">
      <span class="text-gray-500 shrink-0">Click flowline for:</span>
      <button
        class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
        :class="authorComIDSlot === 'up' ? 'border-green-500 bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
        @click="authorComIDSlot = 'up'; authorGaugeSelectMode = false"
      >
        <span class="w-2 h-2 rounded-full bg-green-500 shrink-0" />
        Upstream<template v-if="authorUpComID"> · <span class="font-mono">{{ authorUpComID }}</span></template>
      </button>
      <button
        class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
        :class="authorComIDSlot === 'down' ? 'border-red-500 bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
        @click="authorComIDSlot = 'down'; authorGaugeSelectMode = false"
      >
        <span class="w-2 h-2 rounded-full bg-red-500 shrink-0" />
        Downstream<template v-if="authorDownComID"> · <span class="font-mono">{{ authorDownComID }}</span></template>
      </button>
      <button
        class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        :class="authorGaugeSelectMode ? 'border-amber-500 bg-amber-50 dark:bg-amber-950 text-amber-700 dark:text-amber-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
        :disabled="!authorGauges && !authorAnchorSnap"
        @click="authorGaugeSelectMode = !authorGaugeSelectMode; if (!authorGaugeSelectMode) authorPendingGauge = null"
      >
        <span class="w-2 h-2 rounded-full bg-amber-500 shrink-0" />
        <template v-if="authorGaugeSelectMode">Cancel</template>
        <template v-else-if="authorPendingGauge">Gauge · <span class="font-mono">{{ authorPendingGauge.externalId }}</span></template>
        <template v-else>Select gauge</template>
      </button>
    </div>

    <!-- Pending gauge card -->
    <div v-if="authorPendingGauge" class="flex items-center gap-2 px-3 py-2 mb-3 rounded-lg bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800 text-xs">
      <span class="w-2 h-2 rounded-full bg-amber-500 shrink-0" />
      <span class="flex-1 truncate font-medium text-amber-800 dark:text-amber-200">{{ authorPendingGauge.name || authorPendingGauge.externalId }}</span>
      <span class="font-mono text-amber-600 dark:text-amber-400 shrink-0">{{ authorPendingGauge.externalId }}</span>
      <button class="text-amber-400 hover:text-amber-600 dark:hover:text-amber-300 shrink-0 px-1" @click="authorPendingGauge = null">✕</button>
    </div>
    <p v-if="authorGaugeError" class="text-xs text-red-500 mb-3">{{ authorGaugeError }}</p>

    <NHDExplorerMap
      :upstream-flowlines="authorTributaries"
      :downstream-flowlines="authorDownstreamFlowlines"
      :upstream-gauges="authorGauges"
      :snap-lat="null"
      :snap-lng="null"
      :pick-mode="authorPickMode"
      :comid-select-mode="!!authorAnchorSnap && !authorPickMode && !authorGaugeSelectMode"
      :comid-select-slot="authorComIDSlot"
      :selected-up-comid="authorUpComID"
      :selected-down-comid="authorDownComID"
      :put-in-pin="authorPutInPin"
      :take-out-pin="authorTakeOutPin"
      :preview-centerline="authorPreviewCenterline"
      :gauge-select-mode="authorGaugeSelectMode"
      @pick="onAnchorPick"
      @comid-select="onComIDSelect"
      @gauge-select="onGaugeSelect"
    />
    <p v-if="authorDownstreamLoading" class="text-xs text-blue-500 dark:text-blue-400 mt-1 animate-pulse">Loading downstream mainstem…</p>
    <div v-if="authorUpComID && authorDownComID" class="flex items-center gap-2 mt-1 flex-wrap">
      <UButton size="xs" variant="outline" color="neutral" :loading="authorPreviewLoading" @click="previewCenterline">
        {{ authorPreviewCenterline ? 'Refresh preview' : 'Preview centerline' }}
      </UButton>
      <span v-if="authorPreviewCenterline" class="text-xs text-blue-600 dark:text-blue-400">Dashed line shows trimmed reach</span>
      <span v-if="authorRiverNameFetching" class="text-xs text-gray-400 animate-pulse">Looking up river…</span>
    </div>

    <!-- "Looks like [river name]" hint — appears once river name is auto-detected -->
    <div v-if="authorUpComID && authorDownComID && authorForm.riverName && !authorRiverNameFetching" class="mt-2 flex items-center gap-2 px-3 py-2 rounded-lg bg-emerald-50 dark:bg-emerald-950/50 border border-emerald-200 dark:border-emerald-800 text-xs text-emerald-800 dark:text-emerald-200">
      <svg class="w-3.5 h-3.5 shrink-0 text-emerald-500" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/></svg>
      <span>Looks like <strong>{{ authorForm.riverName }}</strong><template v-if="authorGnisId"> · GNIS {{ authorGnisId }}</template></span>
    </div>

    <!-- Reach form — shown once both ComIDs selected -->
    <div v-if="authorUpComID && authorDownComID" class="mt-4 space-y-3 rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900">
      <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-100">New reach details</h3>

      <!-- ComID summary -->
      <div class="grid grid-cols-2 gap-2 text-xs">
        <div class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800">
          <span class="w-2 h-2 rounded-full bg-green-600 shrink-0" />
          <div>
            <div class="font-medium text-green-800 dark:text-green-200">Upstream (put-in) ComID</div>
            <div class="text-green-600 dark:text-green-400 font-mono">{{ authorUpComID }}</div>
          </div>
        </div>
        <div class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800">
          <span class="w-2 h-2 rounded-full bg-red-600 shrink-0" />
          <div>
            <div class="font-medium text-red-800 dark:text-red-200">Downstream (take-out) ComID</div>
            <div class="text-red-600 dark:text-red-400 font-mono">{{ authorDownComID }}</div>
          </div>
        </div>
      </div>

      <!-- River name -->
      <div>
        <label class="block text-xs text-gray-500 mb-1">River name <span class="text-gray-300">(from NHD)</span></label>
        <div class="flex items-center gap-2">
          <input
            v-model="authorForm.riverName"
            :readonly="!authorRiverNameOverride"
            class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm"
            :class="authorRiverNameOverride ? '' : 'text-gray-400 dark:text-gray-500 cursor-default'"
            placeholder="Auto-filled from anchor snap"
          />
          <UButton
            size="xs"
            variant="outline"
            color="neutral"
            :loading="authorRiverNameFetching"
            :disabled="!authorAnchorSnap && !authorUpComID"
            @click="fetchRiverName"
          >Lookup from NLDI</UButton>
          <button class="text-xs text-blue-500 hover:text-blue-400 shrink-0" @click="authorRiverNameOverride = !authorRiverNameOverride">
            {{ authorRiverNameOverride ? 'Lock' : 'Override' }}
          </button>
        </div>
      </div>

      <div>
        <label class="block text-xs text-gray-500 mb-1">Reach name <span class="text-red-400">*</span></label>
        <input v-model="authorForm.name" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Lees Ferry to Diamond Creek" />
      </div>

      <div>
        <label class="block text-xs text-gray-500 mb-1">Slug</label>
        <div class="flex items-center gap-2">
          <input
            v-model="authorForm.slug"
            class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-xs font-mono"
            placeholder="auto-generated from river + reach name"
            @input="authorSlugManual = true"
          />
          <span v-if="authorSlugChecking" class="text-xs text-gray-400 shrink-0">checking…</span>
          <span v-else-if="authorForm.slug && authorSlugAvailable === true" class="text-xs text-green-600 dark:text-green-400 shrink-0">available</span>
          <span v-else-if="authorForm.slug && authorSlugAvailable === false" class="text-xs text-red-500 shrink-0">taken</span>
        </div>
      </div>

      <div>
        <label class="block text-xs text-gray-500 mb-1">Common name</label>
        <input v-model="authorForm.commonName" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. Grand Canyon" />
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="block text-xs text-gray-500 mb-1">Class min</label>
          <input v-model.number="authorForm.classMin" type="number" min="1" max="6" step="0.5" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="3" />
        </div>
        <div>
          <label class="block text-xs text-gray-500 mb-1">Class max</label>
          <input v-model.number="authorForm.classMax" type="number" min="1" max="6" step="0.5" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="5" />
        </div>
      </div>

      <div>
        <label class="block text-xs text-gray-500 mb-1">Description</label>
        <textarea
          v-model="authorForm.description"
          rows="4"
          class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm resize-y"
          placeholder="Overview of the reach — character, season, access notes…"
        />
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="block text-xs text-gray-500 mb-1">Multi-day (days)</label>
          <input
            v-model.number="authorForm.multiDay"
            type="number" min="1" max="30"
            class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm"
            placeholder="1"
          />
          <p class="text-xs text-gray-400 mt-0.5">1 = single day</p>
        </div>
        <div class="flex items-start pt-5">
          <label class="flex items-center gap-2 cursor-pointer select-none text-sm text-gray-700 dark:text-gray-300">
            <input type="checkbox" v-model="authorForm.permitRequired" class="rounded" />
            Permit required
          </label>
        </div>
      </div>

      <!-- Flow ranges -->
      <div class="rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50 p-3 space-y-2">
        <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Flow ranges (CFS)</p>
        <div class="grid grid-cols-2 gap-2 text-xs">
          <div v-for="band in authorFlowBands" :key="band.key" class="space-y-1">
            <div class="flex items-center gap-1.5">
              <span class="w-2 h-2 rounded-full shrink-0" :style="{ background: band.dot }" />
              <span class="font-medium text-gray-600 dark:text-gray-300">{{ band.label }}</span>
            </div>
            <div class="flex items-center gap-1">
              <input
                v-model.number="authorForm.flowRanges[band.key].min"
                type="number" min="0"
                class="w-full rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
                :placeholder="band.showMin ? 'min' : '—'"
                :disabled="!band.showMin"
              />
              <span class="text-gray-300 shrink-0">–</span>
              <input
                v-model.number="authorForm.flowRanges[band.key].max"
                type="number" min="0"
                class="w-full rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
                :placeholder="band.showMax ? 'max' : '—'"
                :disabled="!band.showMax"
              />
            </div>
          </div>
        </div>
      </div>

      <div v-if="authorError" class="text-xs text-red-500">{{ authorError }}</div>
      <div v-if="authorSuccess" class="text-xs text-green-600 dark:text-green-400">{{ authorSuccess }}</div>

      <!-- GNIS confirm step — inline before final save -->
      <div v-if="authorGnisConfirm" class="rounded-lg border border-amber-200 dark:border-amber-700 bg-amber-50 dark:bg-amber-950/40 p-3 space-y-2">
        <p class="text-xs font-semibold text-amber-800 dark:text-amber-200">Confirm river assignment</p>
        <div class="flex items-center gap-2">
          <label class="text-xs text-gray-600 dark:text-gray-300 shrink-0">River name</label>
          <input
            v-model="authorForm.riverName"
            class="flex-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1 text-xs"
            placeholder="River name"
          />
        </div>
        <p class="text-xs text-gray-400">{{ authorGnisId ? `GNIS ${authorGnisId} · ` : 'No GNIS match · ' }}River will be created if it doesn't exist.</p>
        <div class="flex gap-2 justify-end">
          <UButton size="xs" variant="ghost" color="neutral" @click="authorGnisConfirm = false">Back</UButton>
          <UButton size="xs" :loading="authorSaving" :disabled="!authorForm.name.trim()" @click="submit">Confirm &amp; save</UButton>
        </div>
      </div>

      <div v-else class="flex gap-2 justify-end pt-1">
        <UButton size="sm" variant="ghost" color="neutral" @click="onCancel">Cancel</UButton>
        <UButton size="sm" :disabled="!authorForm.name.trim()" @click="authorGnisConfirm = true">Save reach</UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'

interface NHDFC { type: string; features: any[] }

const emit = defineEmits<{
  created: [slug: string]
  cancel:  []
}>()

const { getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public

const authorPickMode            = ref(false)
const authorAnchorSnapping      = ref(false)
const authorAnchorSnap          = ref<{ comid: string; name: string } | null>(null)
const authorTributaries         = ref<NHDFC | null>(null)
const authorDownstreamFlowlines = ref<NHDFC | null>(null)
const authorDownstreamLoading   = ref(false)
const authorComIDSlot           = ref<'up' | 'down'>('up')
const authorUpComID             = ref<string | null>(null)
const authorDownComID           = ref<string | null>(null)
const authorStartLat            = ref<number | null>(null)
const authorStartLng            = ref<number | null>(null)
const authorEndLat              = ref<number | null>(null)
const authorEndLng              = ref<number | null>(null)
const authorRiverNameOverride   = ref(false)
const authorError               = ref('')
const authorSuccess             = ref('')
const authorSaving              = ref(false)
const authorPreviewCenterline   = ref<object | null>(null)
const authorPreviewLoading      = ref(false)
const authorSlugManual          = ref(false)
const authorSlugAvailable       = ref<boolean | null>(null)
const authorSlugChecking        = ref(false)
let   authorSlugTimer: ReturnType<typeof setTimeout> | null = null

const authorGaugeSelectMode     = ref(false)
const authorGauges              = ref<NHDFC | null>(null)
const authorPendingGauge        = ref<{ externalId: string; source: string; name: string; lat: number; lng: number } | null>(null)
const authorGaugeSaving         = ref(false)
const authorGaugeError          = ref('')
const authorRiverNameFetching   = ref(false)
const authorGnisId              = ref('')
const authorGnisConfirm         = ref(false)

const authorForm = ref({
  name: '', commonName: '', riverName: '', slug: '',
  classMin: null as number | null, classMax: null as number | null,
  description: '',
  permitRequired: false,
  multiDay: 1,
  flowRanges: {
    low:     { min: null as number | null, max: null as number | null },
    running: { min: null as number | null, max: null as number | null },
    high:    { min: null as number | null, max: null as number | null },
  },
})

const authorFlowBands = [
  { key: 'low',     label: 'Too Low',  dot: '#ef4444', showMin: false, showMax: true  },
  { key: 'running', label: 'Runnable', dot: '#34d399', showMin: true,  showMax: true  },
  { key: 'high',    label: 'High',     dot: '#38bdf8', showMin: true,  showMax: false },
] as const

const authorPutInPin = computed(() =>
  authorStartLat.value != null && authorStartLng.value != null
    ? { lat: authorStartLat.value, lng: authorStartLng.value, label: 'Start' }
    : null
)
const authorTakeOutPin = computed(() =>
  authorEndLat.value != null && authorEndLng.value != null
    ? { lat: authorEndLat.value, lng: authorEndLng.value, label: 'End' }
    : null
)

const authorComputedSlug = computed(() => {
  const river = authorForm.value.riverName.trim()
  const name  = authorForm.value.name.trim()
  if (!river || !name) return ''
  const slugify = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
  return `${slugify(river)}-${slugify(name)}`
})

watch(authorComputedSlug, (val) => {
  if (!authorSlugManual.value) authorForm.value.slug = val
})

watch(() => authorForm.value.slug, (val) => {
  authorSlugAvailable.value = null
  if (authorSlugTimer) clearTimeout(authorSlugTimer)
  if (!val) return
  authorSlugChecking.value = true
  authorSlugTimer = setTimeout(async () => {
    try {
      const d = await $fetch<{ available: boolean }>(`${apiBase}/api/v1/admin/slug-check?slug=${encodeURIComponent(val)}`)
      authorSlugAvailable.value = d.available
    } catch { authorSlugAvailable.value = null }
    finally { authorSlugChecking.value = false }
  }, 400)
})

watch(authorUpComID, async (comid) => {
  authorDownstreamFlowlines.value = null
  if (!comid) return
  authorDownstreamLoading.value = true
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/downstream?comid=${comid}&distance=800`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (res.ok) {
      const data = await res.json()
      authorDownstreamFlowlines.value = data.downstream_flowlines
    }
  } catch { /* non-fatal */ }
  finally { authorDownstreamLoading.value = false }
})

// Both ComIDs set → auto-preview centerline and look up river name.
watch([authorUpComID, authorDownComID], async ([up, down]) => {
  if (!up || !down) return
  previewCenterline()
  fetchRiverName()
})

onMounted(() => {
  authorPickMode.value = true
})

function resetToPickMode() {
  authorPickMode.value = true
  authorAnchorSnap.value = null
  authorTributaries.value = null
  authorDownstreamFlowlines.value = null
  authorUpComID.value = null; authorDownComID.value = null
  authorStartLat.value = null; authorStartLng.value = null
  authorEndLat.value = null; authorEndLng.value = null
  authorPreviewCenterline.value = null
  authorComIDSlot.value = 'up'
  authorGnisConfirm.value = false
}

function reset() {
  authorPickMode.value = true  // auto-restart pick mode after clear
  authorAnchorSnapping.value = false
  authorAnchorSnap.value = null
  authorTributaries.value = null
  authorDownstreamFlowlines.value = null
  authorDownstreamLoading.value = false
  authorComIDSlot.value = 'up'
  authorUpComID.value = null
  authorDownComID.value = null
  authorStartLat.value = null; authorStartLng.value = null
  authorEndLat.value = null;   authorEndLng.value = null
  authorRiverNameOverride.value = false
  authorError.value = ''; authorSuccess.value = ''
  authorSaving.value = false
  authorPreviewCenterline.value = null
  authorForm.value = {
    name: '', commonName: '', riverName: '', slug: '',
    classMin: null, classMax: null,
    description: '',
    permitRequired: false,
    multiDay: 1,
    flowRanges: {
      low:     { min: null, max: null },
      running: { min: null, max: null },
      high:    { min: null, max: null },
    },
  }
  authorSlugManual.value = false
  authorSlugAvailable.value = null
  authorGaugeSelectMode.value = false
  authorGauges.value = null
  authorPendingGauge.value = null
  authorGaugeSaving.value = false
  authorGaugeError.value = ''
  authorRiverNameFetching.value = false
  authorGnisId.value = ''
  authorGnisConfirm.value = false
}

async function fetchNearbyGauges(lat: number, lng: number, comid?: string | null) {
  try {
    const token = await getToken()
    const comidParam = comid ? `&comid=${comid}` : ''
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/nearby-gauges?lat=${lat}&lng=${lng}&distance=100${comidParam}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (res.ok) authorGauges.value = await res.json()
  } catch { /* non-fatal */ }
}

async function fetchRiverName() {
  const comid = authorUpComID.value ?? authorAnchorSnap.value?.comid
  if (!comid) return
  authorRiverNameFetching.value = true
  try {
    const token = await getToken()
    if (!token) return
    const lat = authorStartLat.value
    const lng = authorStartLng.value
    const coordParams = lat != null && lng != null ? `&lat=${lat}&lng=${lng}` : ''
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/river-name?comid=${comid}${coordParams}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok) return
    const data = await res.json()
    if (data.river_name && !authorRiverNameOverride.value) {
      authorForm.value.riverName = data.river_name
    }
    if (data.gnis_id) authorGnisId.value = data.gnis_id
  } catch { /* non-fatal */ }
  finally { authorRiverNameFetching.value = false }
}

function onGaugeSelect(externalId: string, source: string, name: string, lat: number, lng: number) {
  authorGaugeSelectMode.value = false
  authorPendingGauge.value = { externalId, source, name, lat, lng }
}

function onCancel() {
  reset()
  emit('cancel')
}

async function onAnchorPick(lat: number, lng: number) {
  authorPickMode.value = false
  authorAnchorSnapping.value = true
  authorAnchorSnap.value = null
  authorTributaries.value = null
  authorError.value = ''
  try {
    const token = await getToken()
    if (!token) return
    const url = `${apiBase}/api/v1/admin/nldi/upstream-tributaries?lat=${lat}&lng=${lng}&distance=50`
    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    if (!res.ok) { authorError.value = `Snap failed: HTTP ${res.status}`; return }
    const data = await res.json()
    authorAnchorSnap.value = { comid: data.snap.comid, name: data.snap.name ?? '' }
    authorTributaries.value = data.tributaries
    if (!authorRiverNameOverride.value && data.snap.name) {
      authorForm.value.riverName = data.snap.name
    }
    await fetchNearbyGauges(lat, lng, data.snap.comid)
  } catch (e: any) {
    authorError.value = e.message ?? 'Snap failed'
  } finally {
    authorAnchorSnapping.value = false
  }
}

function onComIDSelect(comid: string, lat: number, lng: number) {
  if (authorComIDSlot.value === 'up') {
    authorUpComID.value = comid
    authorStartLat.value = lat; authorStartLng.value = lng
    authorComIDSlot.value = 'down'
  } else {
    authorDownComID.value = comid
    authorEndLat.value = lat; authorEndLng.value = lng
  }
}

async function previewCenterline() {
  if (!authorUpComID.value || !authorDownComID.value) return
  authorPreviewLoading.value = true
  authorPreviewCenterline.value = null
  try {
    const token = await getToken()
    let url = `${apiBase}/api/v1/admin/nldi/preview-centerline?up_comid=${authorUpComID.value}&down_comid=${authorDownComID.value}`
    if (authorStartLat.value != null && authorStartLng.value != null && authorEndLat.value != null && authorEndLng.value != null)
      url += `&start_lat=${authorStartLat.value}&start_lng=${authorStartLng.value}&end_lat=${authorEndLat.value}&end_lng=${authorEndLng.value}`
    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    if (res.ok) {
      const data = await res.json()
      authorPreviewCenterline.value = data.geojson
    }
  } catch { /* non-fatal */ }
  finally { authorPreviewLoading.value = false }
}

async function submit() {
  if (!authorForm.value.name.trim() || !authorUpComID.value || !authorDownComID.value) return
  authorSaving.value = true
  authorError.value = ''
  authorSuccess.value = ''
  const token = await getToken()
  if (!token) { authorSaving.value = false; return }
  try {
    const f = authorForm.value
    const days = (f.multiDay ?? 1) < 1 ? 1 : (f.multiDay ?? 1)
    const body = {
      slug:            f.slug.trim() || undefined,
      name:            f.name.trim(),
      common_name:     f.commonName.trim(),
      river_name:      f.riverName.trim(),
      up_comid:        authorUpComID.value,
      down_comid:      authorDownComID.value,
      start_lat:       authorStartLat.value,
      start_lng:       authorStartLng.value,
      end_lat:         authorEndLat.value,
      end_lng:         authorEndLng.value,
      class_min:       f.classMin,
      class_max:       f.classMax,
      description:     f.description.trim() || undefined,
      permit_required: f.permitRequired,
      multi_day_days:  days,
    }
    const res = await fetch(`${apiBase}/api/v1/admin/reaches`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    })
    const data = await res.json()
    if (!res.ok) { authorError.value = data.error ?? `HTTP ${res.status}`; return }

    const slug: string = data.slug

    // Submit flow ranges if any band has at least one value set
    const ranges = f.flowRanges
    const hasRanges = Object.values(ranges).some(b => b.min != null || b.max != null)
    if (hasRanges) {
      await fetch(`${apiBase}/api/v1/reaches/${slug}/flow-ranges`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          low:     { min_value: null,              max_value: ranges.low.max     },
          running: { min_value: ranges.running.min, max_value: ranges.running.max },
          high:    { min_value: ranges.high.min,    max_value: null               },
        }),
      })
    }

    if (f.riverName.trim()) {
      await fetch(`${apiBase}/api/v1/admin/reaches/${slug}/auto-river`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ river_name: f.riverName.trim(), gnis_id: authorGnisId.value }),
      }).catch(() => { /* non-fatal */ })
    }

    await fetch(`${apiBase}/api/v1/admin/reaches/${slug}/nldi-centerline-by-comid`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        up_comid:   authorUpComID.value,
        down_comid: authorDownComID.value,
        start_lat:  authorStartLat.value,
        start_lng:  authorStartLng.value,
        end_lat:    authorEndLat.value,
        end_lng:    authorEndLng.value,
      }),
    }).catch(() => { /* non-fatal */ })

    if (authorPendingGauge.value) {
      const { externalId, source, name: gaugeName, lat: gLat, lng: gLng } = authorPendingGauge.value
      try {
        await fetch(`${apiBase}/api/v1/admin/reaches/${slug}/primary-gauge`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
          body: JSON.stringify({ external_id: externalId, source, name: gaugeName, lat: gLat, lng: gLng }),
        })
      } catch (e: any) {
        console.error('Failed to save gauge:', e)
      }
    }

    emit('created', slug)
  } catch (e: any) {
    authorError.value = e.message ?? 'Unknown error'
  } finally {
    authorSaving.value = false
  }
}
</script>
