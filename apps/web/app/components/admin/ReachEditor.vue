<template>
  <div>
    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
    </div>

    <!-- Error -->
    <div v-else-if="loadError" class="rounded-xl border border-red-200 dark:border-red-800 p-6 text-center text-sm text-red-500">
      {{ loadError }}
    </div>

    <!-- Editor -->
    <div v-else-if="repinReach" class="space-y-4">
      <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-100">
        {{ repinReach.name }}<span v-if="repinReach.river_name" class="text-gray-400 font-normal"> · {{ repinReach.river_name }}</span>
      </h3>

      <!-- Metadata form -->
      <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900 space-y-3">
        <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Metadata</p>
        <div>
          <label class="block text-xs text-gray-500 mb-1">Reach name <span class="text-red-400">*</span></label>
          <input v-model="repinForm.name" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
        </div>
        <div>
          <label class="block text-xs text-gray-500 mb-1">Slug</label>
          <div class="flex items-center gap-2">
            <input v-model="repinForm.slug" class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-xs font-mono" />
            <span v-if="repinSlugChecking" class="text-xs text-gray-400 shrink-0">checking…</span>
            <span v-else-if="repinForm.slug && repinSlugAvailable === true" class="text-xs text-green-600 dark:text-green-400 shrink-0">available</span>
            <span v-else-if="repinForm.slug && repinSlugAvailable === false" class="text-xs text-red-500 shrink-0">taken</span>
          </div>
          <p class="text-xs text-gray-400 mt-0.5">Changing the slug will break existing links.</p>
        </div>
        <div>
          <label class="block text-xs text-gray-500 mb-1">Common name</label>
          <input v-model="repinForm.commonName" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
        </div>
        <div>
          <label class="block text-xs text-gray-500 mb-1">River</label>
          <div class="space-y-2">
            <div class="flex items-center gap-2 flex-wrap">
              <USelectMenu
                v-model="repinRiverSelectItem"
                :items="riverSelectItems"
                searchable
                searchable-placeholder="Search rivers…"
                placeholder="— Unassigned —"
                class="flex-1"
              />
              <UButton size="xs" variant="outline" color="neutral" :loading="repinRiverNameFetching" :disabled="!repinUpComID && !repinAnchorSnap && !repinAnchorComID" @click="fetchRiverName">Auto-assign from NLDI</UButton>
            </div>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-xs text-gray-500 mb-1">Class min</label>
            <input v-model.number="repinForm.classMin" type="number" min="1" max="6" step="0.5" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
          </div>
          <div>
            <label class="block text-xs text-gray-500 mb-1">Class max</label>
            <input v-model.number="repinForm.classMax" type="number" min="1" max="6" step="0.5" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-xs text-gray-500 mb-1">Multi-day (days)</label>
            <input v-model.number="repinForm.multiDay" type="number" min="1" max="30" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
            <p class="text-xs text-gray-400 mt-0.5">1 = single day</p>
          </div>
          <div class="flex items-start pt-5">
            <label class="flex items-center gap-2 cursor-pointer select-none text-sm text-gray-700 dark:text-gray-300">
              <input type="checkbox" v-model="repinForm.permitRequired" class="rounded" />
              Permit required
            </label>
          </div>
        </div>
        <div class="flex items-center gap-3 pt-1">
          <span v-if="repinMetaMsg" class="text-xs" :class="repinMetaMsg === 'Saved' ? 'text-green-600 dark:text-green-400' : 'text-red-500'">{{ repinMetaMsg }}</span>
          <div class="flex-1" />
          <UButton size="sm" :loading="repinMetaSaving" :disabled="!repinForm.name.trim()" @click="saveMeta">Save metadata</UButton>
        </div>
      </div>

      <!-- Description editor -->
      <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900 space-y-2">
        <div class="flex items-center justify-between">
          <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Description</p>
          <UButton size="xs" variant="outline" color="neutral" :loading="repinDescGenerating" @click="generateDescription">Generate with AI</UButton>
        </div>
        <textarea
          v-model="repinDescEdit"
          rows="5"
          class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm resize-y"
          placeholder="No description yet — click Generate to create one with AI, or type directly."
        />
        <div class="flex items-center gap-3">
          <span v-if="repinDescMsg" class="text-xs" :class="repinDescMsg === 'Description saved' ? 'text-green-600 dark:text-green-400' : 'text-red-500'">{{ repinDescMsg }}</span>
          <div class="flex-1" />
          <UButton size="xs" :loading="repinDescSaving" @click="saveDescription">Save description</UButton>
        </div>
      </div>

      <!-- Flow bands editor -->
      <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900 space-y-3">
        <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Flow bands</p>
        <div class="space-y-2">
          <div v-for="band in repinFlowBandsDef" :key="band.key" class="flex items-center gap-2 text-xs">
            <span class="w-2.5 h-2.5 rounded-full shrink-0" :style="`background:${band.dot}`" />
            <span class="w-20 shrink-0 text-gray-600 dark:text-gray-400">{{ band.label }}</span>
            <template v-if="band.showMin">
              <input
                v-model.number="repinFlowBands[band.key].min"
                type="number" min="0" placeholder="min cfs"
                class="w-24 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
              />
            </template>
            <span v-else class="w-24" />
            <span class="text-gray-400">–</span>
            <template v-if="band.showMax">
              <input
                v-model.number="repinFlowBands[band.key].max"
                type="number" min="0" placeholder="max cfs"
                class="w-24 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
              />
            </template>
            <span v-else class="w-24 text-gray-400 italic">no limit</span>
          </div>
        </div>
        <div class="flex items-center gap-3 pt-1">
          <span v-if="repinFlowBandsMsg" class="text-xs" :class="repinFlowBandsMsg === 'Saved' ? 'text-green-600 dark:text-green-400' : 'text-red-500'">{{ repinFlowBandsMsg }}</span>
          <div class="flex-1" />
          <UButton size="xs" :loading="repinFlowBandsSaving" @click="saveFlowBands">Save flow bands</UButton>
        </div>
      </div>

      <!-- Flow lines & gauge -->
      <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900 space-y-3">
        <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Flow Lines &amp; Gauge</p>

        <div class="flex flex-wrap items-center gap-2">
          <UButton
            size="xs"
            :color="repinPickMode ? 'primary' : 'neutral'"
            :variant="repinPickMode ? 'solid' : 'outline'"
            @click="repinPickMode = !repinPickMode"
          >{{ repinPickMode ? 'Cancel' : 'Re-anchor' }}</UButton>
          <span v-if="repinAnchorSnapping" class="text-xs text-blue-600 dark:text-blue-400 animate-pulse">Snapping to NHD…</span>
          <span v-if="repinAnchorError" class="text-xs text-red-500">{{ repinAnchorError }}</span>
        </div>

        <div v-if="repinAnchorSnap" class="flex items-center gap-3 px-3 py-2 rounded-lg bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 text-xs">
          <span class="w-2.5 h-2.5 rounded-full bg-blue-600 shrink-0" />
          <span class="font-medium text-blue-800 dark:text-blue-200">Anchor ComID {{ repinAnchorSnap.comid }}</span>
          <span v-if="repinAnchorSnap.name" class="text-blue-600 dark:text-blue-300">{{ repinAnchorSnap.name }}</span>
        </div>

        <div v-if="repinAnchorSnap || repinDownstream" class="flex items-center gap-2 text-xs flex-wrap">
          <span class="text-gray-500 shrink-0">Click map for:</span>
          <button
            class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
            :class="repinComIDEditMode === 'up' && !repinGaugeSelectMode ? 'border-green-500 bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
            @click="repinGaugeSelectMode = false; repinComIDEditMode = 'up'"
          >
            <span class="w-2 h-2 rounded-full bg-green-500 shrink-0" />
            Upstream<template v-if="repinUpComID"> · <span class="font-mono">{{ repinUpComID }}</span></template>
          </button>
          <button
            class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
            :class="repinComIDEditMode === 'down' && !repinGaugeSelectMode ? 'border-red-500 bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
            @click="repinGaugeSelectMode = false; repinComIDEditMode = 'down'"
          >
            <span class="w-2 h-2 rounded-full bg-red-500 shrink-0" />
            Downstream<template v-if="repinDownComID"> · <span class="font-mono">{{ repinDownComID }}</span></template>
          </button>
          <button
            class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            :class="repinGaugeSelectMode ? 'border-amber-500 bg-amber-50 dark:bg-amber-950 text-amber-700 dark:text-amber-300 font-medium' : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
            :disabled="!repinGauges && !repinUpComID"
            @click="repinGaugeSelectMode = !repinGaugeSelectMode"
          >
            <span class="w-2 h-2 rounded-full bg-amber-500 shrink-0" />
            <template v-if="repinGaugeSaving">Saving…</template>
            <template v-else-if="repinGaugeSelectMode">Cancel gauge</template>
            <template v-else-if="repinPrimaryGauge">Gauge · <span class="font-mono">{{ repinPrimaryGauge.external_id }}</span></template>
            <template v-else>Select gauge</template>
          </button>
        </div>

        <p v-if="repinGaugeError" class="text-xs text-red-500">{{ repinGaugeError }}</p>
        <p v-if="repinGaugeSelectMode" class="text-xs text-amber-600 dark:text-amber-400">Click an amber dot on the map to assign a primary gauge.</p>

        <p v-if="repinFlowlinesLoading" class="text-xs text-blue-500 animate-pulse">Loading downstream mainstem…</p>

        <NHDExplorerMap
          :upstream-flowlines="repinTributaries"
          :downstream-flowlines="repinDownstream"
          :upstream-gauges="repinGauges"
          :snap-lat="null"
          :snap-lng="null"
          :pick-mode="repinPickMode"
          :put-in-pin="repinPutInPin"
          :take-out-pin="repinTakeOutPin"
          :comid-select-mode="!!repinAnchorSnap && !repinPickMode && !repinGaugeSelectMode"
          :comid-select-slot="repinComIDEditMode"
          :selected-up-comid="repinUpComID"
          :selected-down-comid="repinDownComID"
          :preview-centerline="repinPreviewCenterline"
          :gauge-select-mode="repinGaugeSelectMode"
          @pick="onAnchorPick"
          @comid-select="onComIDSelect"
          @gauge-select="onGaugeSelect"
        />

        <div class="flex items-center gap-3 pt-1">
          <span v-if="repinError" class="text-xs text-red-500">{{ repinError }}</span>
          <span v-if="repinSuccess" class="text-xs text-green-600 dark:text-green-400">{{ repinSuccess }}</span>
          <div class="flex-1" />
          <UButton v-if="repinUpComID && repinDownComID" size="xs" variant="outline" color="neutral" :loading="repinPreviewLoading" @click="previewCenterline">
            {{ repinPreviewCenterline ? 'Refresh preview' : 'Preview centerline' }}
          </UButton>
          <UButton v-if="repinFlowlinesDirty" size="sm" variant="outline" color="neutral" @click="resetComIDs">Revert</UButton>
          <UButton size="sm" :loading="repinSaving" :disabled="!repinFlowlinesDirty || !repinUpComID || !repinDownComID" @click="saveFlowLines">Save flow lines</UButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'

interface NHDFC { type: string; features: any[] }
interface RepinReach {
  slug: string; name: string; river_name: string | null; common_name: string | null
  description: string | null
  class_min: number | null; class_max: number | null
  permit_required: boolean; multi_day_days: number
  put_in:   { lat: number; lng: number } | null
  take_out: { lat: number; lng: number } | null
  start_comid: string | null; end_comid: string | null
}
interface River { id: string; slug: string; name: string; state_abbr: string | null }

const props = defineProps<{
  slug:   string
  rivers: River[]
}>()

const emit = defineEmits<{
  'slug-changed':    [newSlug: string]
  'rivers-updated':  []
}>()

const { getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public

// ── State ──────────────────────────────────────────────────────────────────────
const loading    = ref(false)
const loadError  = ref('')
const repinReach = ref<RepinReach | null>(null)

const repinDownstream       = ref<NHDFC | null>(null)
const repinFlowlinesLoading = ref(false)
const repinGauges           = ref<NHDFC | null>(null)
const repinGaugeSelectMode  = ref(false)
const repinGaugeSaving      = ref(false)
const repinGaugeError       = ref('')
const repinPrimaryGauge     = ref<{ id: string; external_id: string; name: string } | null>(null)
const repinError            = ref('')
const repinSuccess          = ref('')
const repinSaving           = ref(false)
const repinDescEdit         = ref('')
const repinDescGenerating   = ref(false)
const repinDescSaving       = ref(false)
const repinDescMsg          = ref('')

const repinForm = ref({
  name: '', commonName: '', riverName: '', slug: '',
  classMin: null as number | null, classMax: null as number | null,
  permitRequired: false, multiDay: 1,
})
const repinMetaSaving        = ref(false)
const repinMetaMsg           = ref('')
const repinRiverNameFetching = ref(false)
const repinRiverId           = ref('')

const repinSlugAvailable = ref<boolean | null>(null)
const repinSlugChecking  = ref(false)
let   repinSlugTimer: ReturnType<typeof setTimeout> | null = null

const repinFlowBands = ref({
  too_low:   { min: null as number | null, max: null as number | null },
  running:   { min: null as number | null, max: null as number | null },
  high:      { min: null as number | null, max: null as number | null },
  very_high: { min: null as number | null, max: null as number | null },
})
const repinFlowBandsDef = [
  { key: 'too_low',   label: 'Too Low',   dot: '#64748b', showMin: false, showMax: true  },
  { key: 'running',   label: 'Runnable',  dot: '#22c55e', showMin: true,  showMax: true  },
  { key: 'high',      label: 'High',      dot: '#f97316', showMin: true,  showMax: true  },
  { key: 'very_high', label: 'Very High', dot: '#ef4444', showMin: true,  showMax: false },
] as const
const repinFlowBandsSaving = ref(false)
const repinFlowBandsMsg    = ref('')

const repinUpComID        = ref<string | null>(null)
const repinDownComID      = ref<string | null>(null)
const repinOrigUpComID    = ref<string | null>(null)
const repinOrigDownComID  = ref<string | null>(null)
const repinAnchorComID    = ref<string | null>(null)
const repinComIDEditMode  = ref<'up' | 'down' | null>(null)
const repinPickMode       = ref(false)
const repinAnchorSnap     = ref<{ comid: string; name: string } | null>(null)
const repinAnchorSnapping = ref(false)
const repinTributaries    = ref<NHDFC | null>(null)
const repinAnchorError    = ref('')
const repinPreviewCenterline = ref<object | null>(null)
const repinPreviewLoading    = ref(false)
const repinStartLat       = ref<number | null>(null)
const repinStartLng       = ref<number | null>(null)
const repinEndLat         = ref<number | null>(null)
const repinEndLng         = ref<number | null>(null)
const repinFlowlinesDirty = ref(false)

// ── Computed ───────────────────────────────────────────────────────────────────
const repinRiverSelectItem = computed({
  get() {
    if (!repinRiverId.value) return { label: '— Unassigned —', value: '' }
    const rv = props.rivers.find(r => r.id === repinRiverId.value)
    return rv ? { label: rv.name, value: rv.id } : { label: '— Unassigned —', value: '' }
  },
  set(item: { label: string; value: string } | null) {
    repinRiverId.value = item?.value ?? ''
  },
})

const riverSelectItems = computed(() => {
  const groups = new Map<string, River[]>()
  const noState: River[] = []
  for (const rv of props.rivers) {
    const s = rv.state_abbr
    if (!s) { noState.push(rv); continue }
    if (!groups.has(s)) groups.set(s, [])
    groups.get(s)!.push(rv)
  }
  const result: any[] = [{ label: '— Unassigned —', value: '' }]
  for (const [state, rvs] of [...groups.entries()].sort(([a], [b]) => a.localeCompare(b))) {
    result.push({
      label: state,
      items: [...rvs].sort((a, b) => a.name.localeCompare(b.name)).map(rv => ({ label: rv.name, value: rv.id })),
    })
  }
  if (noState.length) {
    result.push({
      label: 'No state',
      items: [...noState].sort((a, b) => a.name.localeCompare(b.name)).map(rv => ({ label: rv.name, value: rv.id })),
    })
  }
  return result
})

const repinPutInPin = computed(() => {
  if (repinStartLat.value != null && repinStartLng.value != null)
    return { lat: repinStartLat.value, lng: repinStartLng.value, label: 'Put-in' }
  return repinReach.value?.put_in
    ? { lat: repinReach.value.put_in.lat, lng: repinReach.value.put_in.lng, label: 'Put-in' }
    : null
})
const repinTakeOutPin = computed(() => {
  if (repinEndLat.value != null && repinEndLng.value != null)
    return { lat: repinEndLat.value, lng: repinEndLng.value, label: 'Take-out' }
  return repinReach.value?.take_out
    ? { lat: repinReach.value.take_out.lat, lng: repinReach.value.take_out.lng, label: 'Take-out' }
    : null
})

// ── Watchers ──────────────────────────────────────────────────────────────────
watch(() => repinForm.value.slug, (val) => {
  repinSlugAvailable.value = null
  if (repinSlugTimer) clearTimeout(repinSlugTimer)
  if (!val) return
  if (val === repinReach.value?.slug) { repinSlugAvailable.value = true; return }
  repinSlugChecking.value = true
  repinSlugTimer = setTimeout(async () => {
    try {
      const exclude = encodeURIComponent(repinReach.value?.slug ?? '')
      const d = await $fetch<{ available: boolean }>(`${apiBase}/api/v1/admin/slug-check?slug=${encodeURIComponent(val)}&exclude=${exclude}`)
      repinSlugAvailable.value = d.available
    } catch { repinSlugAvailable.value = null }
    finally { repinSlugChecking.value = false }
  }, 400)
})

// ── Lifecycle ─────────────────────────────────────────────────────────────────
onMounted(() => {
  if (props.slug) loadReach()
})

// ── Internal helpers ──────────────────────────────────────────────────────────
function resetState() {
  repinReach.value = null
  repinDownstream.value = null
  loadError.value = ''
  repinError.value = ''; repinSuccess.value = ''
  repinSaving.value = false
  repinDescMsg.value = ''
  repinMetaMsg.value = ''
  repinUpComID.value = null; repinDownComID.value = null
  repinOrigUpComID.value = null; repinOrigDownComID.value = null
  repinAnchorComID.value = null
  repinComIDEditMode.value = null
  repinStartLat.value = null; repinStartLng.value = null
  repinEndLat.value = null;   repinEndLng.value = null
  repinFlowlinesDirty.value = false
  repinForm.value = { name: '', commonName: '', riverName: '', slug: '', classMin: null, classMax: null, permitRequired: false, multiDay: 1 }
  repinFlowBands.value = { too_low: { min: null, max: null }, running: { min: null, max: null }, high: { min: null, max: null }, very_high: { min: null, max: null } }
  repinFlowBandsSaving.value = false; repinFlowBandsMsg.value = ''
  repinRiverId.value = ''
  repinSlugAvailable.value = null
  repinPickMode.value = false
  repinAnchorSnap.value = null
  repinTributaries.value = null
  repinAnchorError.value = ''
  repinPreviewCenterline.value = null
  repinGauges.value = null; repinGaugeSelectMode.value = false; repinGaugeError.value = ''
}

function resetComIDs() {
  repinUpComID.value = repinOrigUpComID.value
  repinDownComID.value = repinOrigDownComID.value
  repinComIDEditMode.value = null
  repinStartLat.value = null; repinStartLng.value = null
  repinEndLat.value = null;   repinEndLng.value = null
  repinFlowlinesDirty.value = false
  repinError.value = ''; repinSuccess.value = ''
}

async function loadReach() {
  const slug = props.slug.trim()
  if (!slug) return
  loading.value = true
  resetState()
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${slug}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) {
      loadError.value = res.status === 404 ? `Reach "${slug}" not found` : `HTTP ${res.status}`
      return
    }
    const data = await res.json()
    repinReach.value = {
      slug,
      name:           data.name ?? slug,
      common_name:    data.common_name ?? null,
      river_name:     data.river_name ?? null,
      description:    data.description ?? null,
      class_min:      data.class_min ?? null,
      class_max:      data.class_max ?? null,
      permit_required: !!data.permit_required,
      multi_day_days:  data.multi_day_days ?? 1,
      put_in:   data.put_in ?? null,
      take_out: data.take_out ?? null,
      start_comid: data.start_comid ?? null,
      end_comid:   data.end_comid   ?? null,
    }
    repinRiverId.value = data.river_id ?? ''
    repinPrimaryGauge.value = data.primary_gauge_id
      ? { id: data.primary_gauge_id, external_id: data.primary_gauge_external_id ?? '', name: data.primary_gauge_name ?? data.primary_gauge_external_id ?? '' }
      : null
    repinForm.value = {
      name:           data.name ?? '',
      commonName:     data.common_name ?? '',
      riverName:      data.river_name ?? '',
      slug,
      classMin:       data.class_min ?? null,
      classMax:       data.class_max ?? null,
      permitRequired: !!data.permit_required,
      multiDay:       data.multi_day_days ?? 1,
    }
    repinDescEdit.value = data.description ?? ''
    repinUpComID.value       = data.start_comid ?? null
    repinDownComID.value     = data.end_comid   ?? null
    repinOrigUpComID.value   = data.start_comid ?? null
    repinOrigDownComID.value = data.end_comid   ?? null
    repinAnchorComID.value   = data.anchor_comid ?? null

    // Load existing flow bands
    try {
      const token2 = await getToken()
      const frRes = await fetch(`${apiBase}/api/v1/reaches/${slug}/flow-ranges`, {
        headers: token2 ? { Authorization: `Bearer ${token2}` } : {},
      })
      if (frRes.ok) {
        const bands: Array<{ label: string; min_cfs: number | null; max_cfs: number | null }> = await frRes.json()
        repinFlowBands.value = { too_low: { min: null, max: null }, running: { min: null, max: null }, high: { min: null, max: null }, very_high: { min: null, max: null } }
        for (const b of bands) {
          const k = b.label as keyof typeof repinFlowBands.value
          if (k in repinFlowBands.value) repinFlowBands.value[k] = { min: b.min_cfs ?? null, max: b.max_cfs ?? null }
        }
      }
    } catch { /* non-fatal */ }

    if (data.start_comid) {
      repinAnchorSnap.value = { comid: data.start_comid, name: data.river_name ?? '' }
      repinComIDEditMode.value = 'up'
      await fetchFlowlines(data.start_comid)
      if (data.put_in?.lat != null && data.put_in?.lng != null) {
        fetchNearbyGauges(data.put_in.lat, data.put_in.lng, data.start_comid)
        fetchTributaries(data.put_in.lat, data.put_in.lng)
      }
      if (data.end_comid) previewCenterline()
    } else if (data.put_in?.lat != null && data.put_in?.lng != null) {
      fetchNearbyGauges(data.put_in.lat, data.put_in.lng, null)
      onAnchorPick(data.put_in.lat, data.put_in.lng)
    }
  } catch (e: any) {
    loadError.value = e.message ?? 'Unknown error'
  } finally {
    loading.value = false
  }
}

async function fetchFlowlines(comid: string) {
  repinFlowlinesLoading.value = true
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/downstream?comid=${comid}&distance=800`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (res.ok) repinDownstream.value = (await res.json()).downstream_flowlines
  } catch { /* non-fatal */ }
  finally { repinFlowlinesLoading.value = false }
}

async function fetchNearbyGauges(lat: number, lng: number, comid?: string | null) {
  try {
    const token = await getToken()
    const comidParam = comid ? `&comid=${comid}` : ''
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/nearby-gauges?lat=${lat}&lng=${lng}&distance=100${comidParam}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (res.ok) repinGauges.value = await res.json()
  } catch { /* non-fatal */ }
}

async function fetchTributaries(lat: number, lng: number) {
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/upstream-tributaries?lat=${lat}&lng=${lng}&distance=50`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (res.ok) repinTributaries.value = (await res.json()).tributaries
  } catch { /* non-fatal */ }
}

// ── Event handlers ─────────────────────────────────────────────────────────────
async function onAnchorPick(lat: number, lng: number) {
  repinPickMode.value = false
  repinAnchorSnapping.value = true
  repinAnchorSnap.value = null
  repinTributaries.value = null
  repinAnchorError.value = ''
  repinPreviewCenterline.value = null
  try {
    const token = await getToken()
    if (!token) return
    const res = await fetch(`${apiBase}/api/v1/admin/nldi/upstream-tributaries?lat=${lat}&lng=${lng}&distance=50`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok) { repinAnchorError.value = `Snap failed: HTTP ${res.status}`; return }
    const data = await res.json()
    repinAnchorSnap.value = { comid: data.snap.comid, name: data.snap.name ?? '' }
    repinTributaries.value = data.tributaries
    repinComIDEditMode.value = 'up'
    await fetchFlowlines(data.snap.comid)
  } catch (e: any) {
    repinAnchorError.value = e.message ?? 'Snap failed'
  } finally {
    repinAnchorSnapping.value = false
  }
}

function onComIDSelect(comid: string, lat: number, lng: number) {
  if (!repinComIDEditMode.value) return
  if (repinComIDEditMode.value === 'up') {
    repinUpComID.value = comid
    repinStartLat.value = lat; repinStartLng.value = lng
    repinComIDEditMode.value = 'down'
  } else {
    repinDownComID.value = comid
    repinEndLat.value = lat; repinEndLng.value = lng
  }
  repinFlowlinesDirty.value = true
  repinPreviewCenterline.value = null
}

async function onGaugeSelect(externalId: string, source: string, name: string, lat: number, lng: number) {
  if (!repinReach.value) return
  repinGaugeSelectMode.value = false
  repinGaugeSaving.value = true
  repinGaugeError.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}/primary-gauge`, {
      method: 'PUT',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ external_id: externalId, source, name, lat, lng }),
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      repinGaugeError.value = body.error ?? `Error ${res.status}`
      return
    }
    const data = await res.json()
    repinPrimaryGauge.value = { id: data.gauge_id, external_id: externalId, name: name || externalId }
  } catch (e: any) {
    repinGaugeError.value = e?.message ?? 'Failed to set gauge'
  } finally {
    repinGaugeSaving.value = false
  }
}

// ── Save actions ───────────────────────────────────────────────────────────────
async function saveMeta() {
  if (!repinReach.value || !repinForm.value.name.trim()) return
  repinMetaSaving.value = true
  repinMetaMsg.value = ''
  try {
    const f = repinForm.value
    const days = (f.multiDay ?? 1) < 1 ? 1 : (f.multiDay ?? 1)
    const newSlug = f.slug.trim() || repinReach.value.slug
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}/meta`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({
        name:            f.name.trim(),
        new_slug:        newSlug !== repinReach.value.slug ? newSlug : undefined,
        common_name:     f.commonName.trim(),
        river_name:      f.riverName.trim(),
        river_id:        repinRiverId.value || null,
        class_min:       f.classMin,
        class_max:       f.classMax,
        permit_required: f.permitRequired,
        multi_day_days:  days,
      }),
    })
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      repinMetaMsg.value = d.error ?? `HTTP ${res.status}`
      return
    }
    const d = await res.json().catch(() => ({}))
    const savedSlug = d.slug ?? newSlug
    repinMetaMsg.value = 'Saved'
    repinReach.value.name = f.name.trim()
    repinReach.value.river_name = f.riverName.trim() || null
    if (savedSlug !== repinReach.value.slug) {
      repinReach.value.slug = savedSlug
      repinForm.value.slug = savedSlug
      emit('slug-changed', savedSlug)
    }
  } catch (e: any) {
    repinMetaMsg.value = e.message ?? 'Save failed'
  } finally {
    repinMetaSaving.value = false
  }
}

async function saveFlowBands() {
  if (!repinReach.value) return
  repinFlowBandsSaving.value = true
  repinFlowBandsMsg.value = ''
  try {
    const b = repinFlowBands.value
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/reaches/${repinReach.value.slug}/flow-ranges`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({
        too_low:   { min_cfs: null,            max_cfs: b.too_low.max   },
        running:   { min_cfs: b.running.min,   max_cfs: b.running.max   },
        high:      { min_cfs: b.high.min,      max_cfs: b.high.max      },
        very_high: { min_cfs: b.very_high.min, max_cfs: null            },
      }),
    })
    repinFlowBandsMsg.value = res.ok ? 'Saved' : `HTTP ${res.status}`
  } catch (e: any) {
    repinFlowBandsMsg.value = e.message ?? 'Save failed'
  } finally {
    repinFlowBandsSaving.value = false
  }
}

async function fetchRiverName() {
  const comid = repinUpComID.value ?? repinAnchorSnap.value?.comid ?? repinAnchorComID.value
  if (!comid || !repinReach.value) return
  repinRiverNameFetching.value = true
  try {
    const token = await getToken()
    if (!token) return
    const lat = repinStartLat.value ?? repinReach.value?.put_in?.lat
    const lng = repinStartLng.value ?? repinReach.value?.put_in?.lng
    const coordParams = lat != null && lng != null ? `&lat=${lat}&lng=${lng}` : ''
    const nameRes = await fetch(`${apiBase}/api/v1/admin/nldi/river-name?comid=${comid}${coordParams}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!nameRes.ok) return
    const nameData = await nameRes.json()
    if (!nameData.river_name) return
    const assignRes = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}/auto-river`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ river_name: nameData.river_name, gnis_id: nameData.gnis_id ?? '' }),
    })
    if (!assignRes.ok) return
    const assignData = await assignRes.json()
    repinForm.value.riverName = assignData.river_name
    repinRiverId.value = assignData.river_id
    emit('rivers-updated')
  } catch { /* non-fatal */ }
  finally { repinRiverNameFetching.value = false }
}

async function saveFlowLines() {
  if (!repinReach.value || !repinUpComID.value || !repinDownComID.value) return
  repinSaving.value = true
  repinError.value = ''; repinSuccess.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}/nldi-centerline-by-comid`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({
        up_comid:   repinUpComID.value,
        down_comid: repinDownComID.value,
        start_lat:  repinStartLat.value,
        start_lng:  repinStartLng.value,
        end_lat:    repinEndLat.value,
        end_lng:    repinEndLng.value,
      }),
    })
    const data = await res.json()
    if (!res.ok) { repinError.value = data.error ?? `HTTP ${res.status}`; return }
    const lengthStr = data.length_mi != null ? ` · ${data.length_mi} mi` : ''
    repinSuccess.value = `Flow lines saved${lengthStr}`
    repinOrigUpComID.value = repinUpComID.value
    repinOrigDownComID.value = repinDownComID.value
    repinFlowlinesDirty.value = false
    if (repinReach.value) {
      repinReach.value.start_comid = repinUpComID.value
      repinReach.value.end_comid   = repinDownComID.value
    }
  } catch (e: any) {
    repinError.value = e.message ?? 'Unknown error'
  } finally {
    repinSaving.value = false
  }
}

async function generateDescription() {
  if (!repinReach.value) return
  repinDescGenerating.value = true
  repinDescMsg.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}/generate-description`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    const data = await res.json()
    if (!res.ok) { repinDescMsg.value = data.error ?? `HTTP ${res.status}`; return }
    repinDescEdit.value = data.description ?? ''
  } catch (e: any) {
    repinDescMsg.value = e.message ?? 'Generate failed'
  } finally {
    repinDescGenerating.value = false
  }
}

async function saveDescription() {
  if (!repinReach.value) return
  repinDescSaving.value = true
  repinDescMsg.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/admin/reaches/${repinReach.value.slug}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({ description: repinDescEdit.value || null }),
    })
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      repinDescMsg.value = d.error ?? `HTTP ${res.status}`
      return
    }
    repinDescMsg.value = 'Description saved'
  } catch (e: any) {
    repinDescMsg.value = e.message ?? 'Save failed'
  } finally {
    repinDescSaving.value = false
  }
}

async function previewCenterline() {
  if (!repinUpComID.value || !repinDownComID.value) return
  repinPreviewLoading.value = true
  repinPreviewCenterline.value = null
  try {
    const token = await getToken()
    const startLat = repinStartLat.value ?? repinReach.value?.put_in?.lat
    const startLng = repinStartLng.value ?? repinReach.value?.put_in?.lng
    const endLat   = repinEndLat.value   ?? repinReach.value?.take_out?.lat
    const endLng   = repinEndLng.value   ?? repinReach.value?.take_out?.lng
    let url = `${apiBase}/api/v1/admin/nldi/preview-centerline?up_comid=${repinUpComID.value}&down_comid=${repinDownComID.value}`
    if (startLat != null && startLng != null && endLat != null && endLng != null)
      url += `&start_lat=${startLat}&start_lng=${startLng}&end_lat=${endLat}&end_lng=${endLng}`
    const res = await fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
    if (res.ok) repinPreviewCenterline.value = (await res.json()).geojson
  } catch { /* non-fatal */ }
  finally { repinPreviewLoading.value = false }
}
</script>
