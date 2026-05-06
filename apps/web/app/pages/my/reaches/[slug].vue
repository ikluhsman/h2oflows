<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950 flex flex-col lg:h-screen lg:overflow-hidden">

    <!-- Header -->
    <div class="sticky top-0 z-10 flex items-center justify-between px-4 py-3 bg-white/90 dark:bg-gray-950/90 backdrop-blur-sm border-b border-gray-200 dark:border-gray-800">
      <NuxtLink to="/my/reaches" class="flex items-center gap-1 text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
        <span class="i-heroicons-chevron-left w-3.5 h-3.5" />
        My Reaches
      </NuxtLink>
      <div v-if="reach" class="flex items-center gap-2 shrink-0">
        <UButton size="xs" variant="ghost" color="neutral" icon="i-heroicons-share" @click="shareOpen = true">Share</UButton>
        <UButton size="xs" variant="ghost" color="error" icon="i-heroicons-trash" @click="confirmDelete">Delete</UButton>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex-1 flex items-center justify-center">
      <div class="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="flex-1 flex items-center justify-center text-sm text-gray-500">{{ error }}</div>

    <!-- Content -->
    <div v-else-if="reach" class="flex-1 flex flex-col lg:flex-row lg:overflow-hidden">

      <!-- Map — mobile: half screen with side padding; desktop: sticky full-height column -->
      <div class="px-4 pt-3 pb-1 lg:p-0 lg:flex-1 lg:sticky lg:top-[51px] lg:h-[calc(100vh-51px)]">
        <div class="relative h-[50vh] lg:h-full">
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
            :preview-centerline="repinPreviewCenterline ?? reach.centerline ?? null"
            :gauge-select-mode="repinGaugeSelectMode"
            :disable-auto-fit="anyPickMode"
            map-height="100%"
            class="w-full h-full"
            @pick="onAnchorPick"
            @comid-select="onComIDSelect"
            @gauge-select="onGaugeSelect"
          />
        </div>
      </div>

      <!-- Form panel — mobile: natural scroll; desktop: fixed-height scrollable column -->
      <div class="lg:w-95 lg:overflow-y-auto lg:h-[calc(100vh-51px)] p-4 pb-20 space-y-4">

        <!-- Reach info + live flow -->
        <div class="rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-4">
          <h1 class="text-lg font-bold text-gray-900 dark:text-gray-100 leading-tight">{{ reach.name }}</h1>
          <p v-if="reach.river_name" class="text-sm text-gray-500 mt-0.5">{{ reach.river_name }}</p>
          <div
            v-if="reach.gauge_poll_health === 'stale' || reach.gauge_poll_health === 'unreachable'"
            class="mt-2 flex items-start gap-2 px-2.5 py-1.5 rounded-md text-xs border"
            :class="reach.gauge_poll_health === 'unreachable'
              ? 'bg-red-50 dark:bg-red-950/40 text-red-700 dark:text-red-300 border-red-200 dark:border-red-900'
              : 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 border-amber-200 dark:border-amber-900'"
          >
            <svg class="w-3.5 h-3.5 mt-0.5 shrink-0" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd"/></svg>
            <div>
              <span class="font-medium">{{ reach.gauge_poll_health === 'unreachable' ? 'Gauge unreachable' : 'Gauge data is stale' }}</span>
              <span v-if="reach.gauge_last_poll_success_at" class="block opacity-80">
                Last update {{ new Date(reach.gauge_last_poll_success_at).toLocaleDateString() }}
              </span>
            </div>
          </div>
          <template v-if="reach.gauge_name || reach.current_cfs != null">
            <div class="mt-3 pt-3 border-t border-gray-100 dark:border-gray-800 flex items-end gap-4">
              <div class="flex-1 min-w-0">
                <p v-if="reach.current_cfs != null" class="text-2xl font-bold font-mono" :class="cfsColorClass">
                  {{ reach.current_cfs.toLocaleString() }}<span class="text-sm font-normal text-gray-400 ml-1">cfs</span>
                </p>
                <p v-else class="text-sm text-gray-400">No reading</p>
                <p v-if="reach.gauge_name" class="text-xs text-gray-400 truncate mt-0.5">{{ reach.gauge_name }}</p>
              </div>
              <span
                v-if="reach.flow_band"
                class="text-xs font-medium px-2 py-0.5 rounded-full shrink-0 mb-0.5"
                :class="flowBandBadgeClass(reach.flow_band)"
              >{{ flowBandLabel(reach.flow_band) }}</span>
            </div>
          </template>
        </div>

        <!-- Flow lines & gauge -->
        <div class="rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-4 space-y-3">
          <p class="text-xs text-gray-400 uppercase tracking-wide font-medium">Flow Lines &amp; Gauge</p>

          <!-- Re-anchor row -->
          <div class="flex flex-wrap items-center gap-2">
            <button
              class="text-xs px-2.5 py-1 rounded-md font-medium border transition-colors"
              :class="repinPickMode
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-950 text-blue-700 dark:text-blue-300'
                : 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'"
              @click="togglePickMode"
            >{{ repinPickMode ? 'Cancel' : 'Re-anchor' }}</button>
            <span v-if="repinAnchorSnapping" class="text-xs text-blue-600 dark:text-blue-400 animate-pulse">Snapping to NHD…</span>
            <span v-if="repinAnchorError" class="text-xs text-red-500">{{ repinAnchorError }}</span>
          </div>

          <!-- Anchor snap card -->
          <div v-if="repinAnchorSnap" class="flex items-center gap-3 px-3 py-2 rounded-lg bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 text-xs">
            <span class="w-2.5 h-2.5 rounded-full bg-blue-600 shrink-0" />
            <span class="font-medium text-blue-800 dark:text-blue-200">Anchor ComID {{ repinAnchorSnap.comid }}</span>
            <span v-if="repinAnchorSnap.name" class="text-blue-600 dark:text-blue-300 truncate">{{ repinAnchorSnap.name }}</span>
          </div>

          <!-- ComID + gauge toggles (visible once anchor snapped or downstream loaded) -->
          <div v-if="repinAnchorSnap || repinDownstream" class="flex items-center gap-2 text-xs flex-wrap">
            <span class="text-gray-500 shrink-0">Click map for:</span>
            <button
              class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
              :class="repinComIDEditMode === 'up' && !repinGaugeSelectMode
                ? 'border-green-500 bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 font-medium'
                : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
              @click="repinGaugeSelectMode = false; repinComIDEditMode = 'up'"
            >
              <span class="w-2 h-2 rounded-full bg-green-500 shrink-0" />
              Upstream<template v-if="repinUpComID"> · <span class="font-mono">{{ repinUpComID }}</span></template>
            </button>
            <button
              class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
              :class="repinComIDEditMode === 'down' && !repinGaugeSelectMode
                ? 'border-red-500 bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 font-medium'
                : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
              @click="repinGaugeSelectMode = false; repinComIDEditMode = 'down'"
            >
              <span class="w-2 h-2 rounded-full bg-red-500 shrink-0" />
              Downstream<template v-if="repinDownComID"> · <span class="font-mono">{{ repinDownComID }}</span></template>
            </button>
            <button
              class="flex items-center gap-1.5 px-2 py-1 rounded-md border transition-colors"
              :class="reach.custom_gauge_id
                ? 'border-gray-200 dark:border-gray-700 text-gray-400 cursor-default'
                : repinGaugeSelectMode
                  ? 'border-amber-500 bg-amber-50 dark:bg-amber-950 text-amber-700 dark:text-amber-300 font-medium'
                  : 'border-gray-200 dark:border-gray-700 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
              :disabled="!!reach.custom_gauge_id"
              @click="!reach.custom_gauge_id && (repinGaugeSelectMode = !repinGaugeSelectMode, repinComIDEditMode = null, !repinGaugeSelectMode && (repinPendingGauge = null))"
            >
              <span class="w-2 h-2 rounded-full bg-amber-500 shrink-0" />
              <template v-if="reach.custom_gauge_id">Custom Gauge</template>
              <template v-else-if="repinGaugeSelectMode">Cancel</template>
              <template v-else-if="reach.gauge_id">Gauge · <span class="font-mono">{{ reach.gauge_external_id }}</span></template>
              <template v-else>Select gauge</template>
            </button>
            <button
              v-if="reach.gauge_id && !repinGaugeSelectMode"
              class="px-2 py-1 rounded-md border border-gray-200 dark:border-gray-700 text-gray-400 hover:text-red-500 hover:border-red-400 transition-colors text-xs"
              :disabled="repinGaugeSaving"
              title="Clear gauge"
              @click="clearGauge"
            >✕</button>
          </div>

          <!-- Pending gauge card -->
          <div v-if="repinPendingGauge" class="flex items-center gap-2 px-3 py-2 rounded-lg bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800 text-xs">
            <span class="w-2 h-2 rounded-full bg-amber-500 shrink-0" />
            <span class="flex-1 truncate font-medium text-amber-800 dark:text-amber-200">{{ repinPendingGauge.name || repinPendingGauge.externalId }}</span>
            <span class="font-mono text-amber-600 dark:text-amber-400 shrink-0">{{ repinPendingGauge.externalId }}</span>
            <UButton size="xs" :loading="repinGaugeSaving" @click="saveGauge">Save gauge</UButton>
            <button class="text-amber-400 hover:text-amber-600 dark:hover:text-amber-300 shrink-0 px-1" @click="repinPendingGauge = null">✕</button>
          </div>

          <p v-if="repinGaugeError" class="text-xs text-red-500">{{ repinGaugeError }}</p>
          <p v-if="repinGaugeSelectMode" class="text-xs text-amber-600 dark:text-amber-400">Click an amber dot on the map to assign a primary gauge.</p>
          <p v-if="repinFlowlinesLoading" class="text-xs text-blue-500 animate-pulse">Loading downstream mainstem…</p>

          <!-- Footer: preview / revert / save -->
          <div class="flex items-center gap-2 pt-1 flex-wrap">
            <span v-if="repinError" class="text-xs text-red-500">{{ repinError }}</span>
            <span v-if="repinSuccess" class="text-xs text-green-600 dark:text-green-400">{{ repinSuccess }}</span>
            <div class="flex-1" />
            <button
              v-if="reach.centerline && !repinFlowlinesDirty"
              class="text-xs px-2 py-1 rounded-md font-medium text-red-500 bg-red-50 dark:bg-red-950/30 hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors"
              @click="clearCenterline"
            >Clear centerline</button>
            <UButton
              v-if="repinUpComID && repinDownComID"
              size="xs"
              variant="outline"
              color="neutral"
              :loading="repinPreviewLoading"
              @click="previewCenterline"
            >{{ repinPreviewCenterline ? 'Refresh preview' : 'Preview centerline' }}</UButton>
            <UButton
              v-if="repinFlowlinesDirty"
              size="xs"
              variant="outline"
              color="neutral"
              @click="revertComIDs"
            >Revert</UButton>
            <UButton
              size="xs"
              :loading="repinSaving"
              :disabled="!repinUpComID || !repinDownComID"
              @click="saveFlowLines"
            >Save flow lines</UButton>
          </div>
        </div>

        <!-- Edit form + gauge display -->
        <div class="rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-4 space-y-3">
          <p class="text-xs text-gray-400 uppercase tracking-wide font-medium">Reach details</p>

          <div>
            <label class="block text-xs text-gray-500 mb-1">Reach name <span class="text-red-400">*</span></label>
            <input v-model="form.name" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" />
          </div>

          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="text-xs text-gray-500">River name</label>
              <button
                v-if="repinUpComID"
                type="button"
                :disabled="riverNameLooking"
                class="text-xs text-blue-500 hover:text-blue-700 dark:hover:text-blue-300 disabled:opacity-40"
                @click="lookupRiverName"
              >{{ riverNameLooking ? 'Looking up…' : 'Lookup from NLDI' }}</button>
            </div>
            <input v-model="form.riverName" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm" placeholder="e.g. South Platte River" />
          </div>

          <div>
            <label class="block text-xs text-gray-500 mb-1">Notes</label>
            <textarea v-model="form.note" rows="3" class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-2 py-1.5 text-sm resize-y" placeholder="Beta, access, hazards…" />
          </div>

          <!-- Gauge read-only display -->
          <div class="pt-2 border-t border-gray-100 dark:border-gray-800 space-y-1.5">
            <div class="flex items-center justify-between gap-2">
              <p class="text-xs text-gray-500">Linked gauge</p>
              <div class="relative">
                <UButton
                  v-if="!reach.custom_gauge_id && !reach.gauge_id"
                  size="2xs" variant="outline" color="neutral"
                  :loading="customGaugeSaving"
                  @click="customGaugePickerOpen = !customGaugePickerOpen"
                >
                  Link custom gauge
                </UButton>
                <!-- Custom gauge picker dropdown -->
                <div
                  v-if="customGaugePickerOpen && customGauges.length"
                  class="absolute right-0 top-full mt-1 z-50 w-56 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-lg overflow-hidden"
                >
                  <button
                    v-for="cg in customGauges"
                    :key="cg.id"
                    type="button"
                    class="w-full text-left px-3 py-2 text-sm hover:bg-blue-50 dark:hover:bg-blue-950/30 transition-colors"
                    @click="linkCustomGauge(cg)"
                  >{{ cg.name }}</button>
                </div>
                <p v-if="customGaugePickerOpen && !customGauges.length" class="text-xs text-gray-400">
                  No custom gauges yet. <NuxtLink to="/my/gauges/new" class="text-blue-500 underline">Create one</NuxtLink>
                </p>
              </div>
            </div>

            <!-- Real gauge linked -->
            <template v-if="reach.gauge_name">
              <p class="text-sm text-gray-700 dark:text-gray-300">{{ reach.gauge_name }}</p>
              <p v-if="reach.gauge_external_id" class="text-xs text-gray-400 font-mono">
                {{ reach.gauge_external_id }}<span v-if="reach.gauge_source" class="uppercase ml-1.5">{{ reach.gauge_source }}</span>
              </p>
            </template>

            <!-- Custom gauge linked -->
            <template v-else-if="reach.custom_gauge_name">
              <div class="flex items-center gap-1.5">
                <svg class="w-3.5 h-3.5 text-gray-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="4" y="2" width="16" height="20" rx="2"/><line x1="8" y1="6" x2="16" y2="6"/><line x1="8" y1="10" x2="10" y2="10"/><line x1="14" y1="10" x2="16" y2="10"/><line x1="8" y1="14" x2="10" y2="14"/><line x1="14" y1="14" x2="16" y2="14"/>
                </svg>
                <p class="text-sm text-gray-700 dark:text-gray-300">{{ reach.custom_gauge_name }}</p>
              </div>
              <NuxtLink
                :to="customGauges.find(g => g.id === reach.custom_gauge_id)?.slug
                  ? `/my/gauges/${customGauges.find(g => g.id === reach.custom_gauge_id)!.slug}`
                  : '/my/gauges'"
                class="text-xs text-blue-500 hover:underline"
              >Edit formula →</NuxtLink>
            </template>

            <p v-else class="text-xs text-gray-400 italic">None</p>
          </div>
        </div>

        <!-- Flow bands -->
        <div class="rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-4 space-y-2">
          <p class="text-xs text-gray-400 uppercase tracking-wide font-medium">Flow bands <span class="font-normal normal-case text-gray-400">(CFS)</span></p>
          <div class="space-y-2">
            <div v-for="band in flowBandDefs" :key="band.key" class="flex items-center gap-2">
              <span class="w-16 text-xs font-medium shrink-0" :class="band.labelClass">{{ band.label }}</span>
              <input
                v-model.number="form.ranges[band.key].min"
                type="number" min="0"
                class="w-20 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
                :placeholder="band.showMin ? 'min' : '—'"
                :disabled="!band.showMin"
              />
              <span class="text-gray-400 text-xs">–</span>
              <input
                v-model.number="form.ranges[band.key].max"
                type="number" min="0"
                class="w-20 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-1.5 py-1 text-xs"
                :placeholder="band.showMax ? 'max' : '—'"
                :disabled="!band.showMax"
              />
              <span class="text-xs text-gray-400">cfs</span>
            </div>
          </div>
        </div>

        <!-- Save metadata -->
        <div v-if="saveError" class="text-xs text-red-500 px-1">{{ saveError }}</div>
        <div class="flex justify-end gap-2 pb-6">
          <UButton :disabled="!form.name.trim()" :loading="saving" @click="save">Save changes</UButton>
        </div>

      </div>
    </div>
  </div>

  <!-- Share modal -->
  <UModal v-model:open="shareOpen" title="Share reach">
    <template #body>
      <p class="text-xs text-gray-500 mb-2">Copy this payload and paste it into the import dialog on another account.</p>
      <textarea
        readonly
        rows="12"
        class="w-full rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 px-3 py-2 text-xs font-mono resize-none focus:outline-none"
        :value="sharePayload"
        @click="($event.target as HTMLTextAreaElement).select()"
      />
      <div class="flex justify-end gap-2 mt-3">
        <UButton size="xs" variant="outline" color="neutral" @click="shareOpen = false">Close</UButton>
        <UButton size="xs" icon="i-heroicons-clipboard-document" @click="copyShare">{{ shareCopied ? 'Copied!' : 'Copy' }}</UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { flowBandBadgeClass, flowBandLabel } from '~/utils/flowBand'

definePageMeta({ ssr: false })

const route  = useRoute()
const router = useRouter()
const { getToken } = useAuth()
const { apiBase }  = useRuntimeConfig().public

const slug = computed(() => route.params.slug as string)

const loading   = ref(false)
const error     = ref('')
const saving    = ref(false)
const saveError = ref('')

// ── Types ─────────────────────────────────────────────────────────────────────

interface NHDFC { type: string; features: any[] }
interface AnchorSnap { comid: string; name: string }
interface PendingGauge { externalId: string; source: string; name: string; lat: number; lng: number }
type ComIDEditMode = 'up' | 'down' | null

interface FlowRange {
  label: string; min_value: number | null; max_value: number | null
}

interface UserReachDetail {
  id:                string
  slug:              string
  name:              string
  river_name:        string | null
  put_in_lng:        number
  put_in_lat:        number
  take_out_lng:      number
  take_out_lat:      number
  up_comid:          string | null
  down_comid:        string | null
  gauge_id:                  string | null
  gauge_name:                string | null
  gauge_source:              string | null
  gauge_external_id:         string | null
  gauge_poll_health:         string | null
  gauge_last_poll_success_at: string | null
  custom_gauge_id:           string | null
  custom_gauge_name: string | null
  current_cfs:       number | null
  flow_band:         string | null
  note:              string | null
  flow_ranges:       FlowRange[]
  centerline:        object | null
}

interface CustomGaugeSummary { id: string; slug: string; name: string }

const reach = ref<UserReachDetail | null>(null)

// ── Form state ────────────────────────────────────────────────────────────────

const form = ref({
  name:      '',
  riverName: '',
  note:      '',
  ranges: {
    low:     { min: null as number | null, max: null as number | null },
    running: { min: null as number | null, max: null as number | null },
    high:    { min: null as number | null, max: null as number | null },
  },
})

const flowBandDefs = [
  { key: 'low',     label: 'Too Low',  labelClass: 'text-gray-400',    showMin: false, showMax: true  },
  { key: 'running', label: 'Runnable', labelClass: 'text-emerald-500', showMin: true,  showMax: true  },
  { key: 'high',    label: 'High',     labelClass: 'text-sky-400',     showMin: true,  showMax: false },
] as const

// ── Repin state (mirrors admin ReachEditor) ───────────────────────────────────

const repinPickMode        = ref(false)
const repinAnchorSnap      = ref<AnchorSnap | null>(null)
const repinAnchorSnapping  = ref(false)
const repinAnchorError     = ref('')
const repinTributaries     = ref<NHDFC | null>(null)
const repinDownstream      = ref<NHDFC | null>(null)
const repinFlowlinesLoading = ref(false)
const repinComIDEditMode   = ref<ComIDEditMode>(null)
const repinUpComID         = ref<string | null>(null)
const repinDownComID       = ref<string | null>(null)
const repinOrigUpComID     = ref<string | null>(null)
const repinOrigDownComID   = ref<string | null>(null)
const repinStartLat        = ref<number | null>(null)
const repinStartLng        = ref<number | null>(null)
const repinEndLat          = ref<number | null>(null)
const repinEndLng          = ref<number | null>(null)
const repinFlowlinesDirty  = ref(false)
const repinPreviewCenterline = ref<object | null>(null)
const repinPreviewLoading  = ref(false)
const repinSaving          = ref(false)
const repinError           = ref('')
const repinSuccess         = ref('')

const repinGauges          = ref<NHDFC | null>(null)
const repinGaugeSelectMode = ref(false)
const repinPendingGauge    = ref<PendingGauge | null>(null)
const repinGaugeSaving     = ref(false)
const repinGaugeError      = ref('')

const riverNameLooking     = ref(false)

const customGauges           = ref<CustomGaugeSummary[]>([])
const customGaugePickerOpen  = ref(false)
const customGaugeSaving      = ref(false)

const shareOpen   = ref(false)
const shareCopied = ref(false)

// ── Computed ──────────────────────────────────────────────────────────────────

const repinPutInPin = computed(() => {
  if (repinStartLat.value != null && repinStartLng.value != null)
    return { lat: repinStartLat.value, lng: repinStartLng.value, label: 'Put-in' }
  if (!reach.value) return null
  return { lat: reach.value.put_in_lat, lng: reach.value.put_in_lng, label: 'Put-in' }
})

const repinTakeOutPin = computed(() => {
  if (repinEndLat.value != null && repinEndLng.value != null)
    return { lat: repinEndLat.value, lng: repinEndLng.value, label: 'Take-out' }
  if (!reach.value) return null
  return { lat: reach.value.take_out_lat, lng: reach.value.take_out_lng, label: 'Take-out' }
})

const anyPickMode = computed(() =>
  repinPickMode.value || repinGaugeSelectMode.value || repinComIDEditMode.value !== null,
)

const cfsColorClass = computed(() => {
  const b = reach.value?.flow_band
  if (b === 'running') return 'text-emerald-500'
  if (b === 'high')    return 'text-sky-400'
  return 'text-gray-700 dark:text-gray-200'
})

const sharePayload = computed(() => {
  const r = reach.value
  if (!r) return ''
  const fr = form.value.ranges
  const payload: Record<string, unknown> = {
    name:       r.name,
    river_name: r.river_name ?? '',
    put_in:  { lat: r.put_in_lat,    lng: r.put_in_lng    },
    take_out: { lat: r.take_out_lat, lng: r.take_out_lng  },
    up_comid:   r.up_comid   ?? '',
    down_comid: r.down_comid ?? '',
    note:       r.note ?? '',
    flow_ranges: {
      low:     { min_value: null,          max_value: fr.low.max     },
      running: { min_value: fr.running.min, max_value: fr.running.max },
      high:    { min_value: fr.high.min,   max_value: null           },
    },
  }
  return JSON.stringify(payload, null, 2)
})

async function copyShare() {
  if (!sharePayload.value) return
  await navigator.clipboard.writeText(sharePayload.value)
  shareCopied.value = true
  setTimeout(() => { shareCopied.value = false }, 2000)
}

// ── Auth helper ───────────────────────────────────────────────────────────────

async function authHeaders(): Promise<Record<string, string>> {
  const token = await getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

// ── NLDI fetches ──────────────────────────────────────────────────────────────

async function fetchTributaries(lat: number, lng: number): Promise<AnchorSnap | null> {
  const headers = await authHeaders()
  const res = await fetch(
    `${apiBase}/api/v1/nldi/upstream-tributaries?lat=${lat}&lng=${lng}&distance=50`,
    { headers },
  )
  if (!res.ok) {
    repinAnchorError.value = `Snap failed: HTTP ${res.status}`
    return null
  }
  const data = await res.json()
  repinTributaries.value = data.tributaries ?? null
  return data.snap ? { comid: data.snap.comid, name: data.snap.name ?? '' } : null
}

async function fetchDownstream(comid: string): Promise<void> {
  repinFlowlinesLoading.value = true
  try {
    const headers = await authHeaders()
    const res = await fetch(`${apiBase}/api/v1/nldi/downstream?comid=${comid}&distance=800`, { headers })
    if (res.ok) repinDownstream.value = (await res.json()).downstream_flowlines ?? null
  } catch { /* non-fatal */ }
  finally { repinFlowlinesLoading.value = false }
}

async function fetchNearbyGauges(lat: number, lng: number, comid?: string | null): Promise<void> {
  try {
    const headers = await authHeaders()
    const comidParam = comid ? `&comid=${comid}` : ''
    const res = await fetch(
      `${apiBase}/api/v1/nldi/nearby-gauges?lat=${lat}&lng=${lng}&distance=100${comidParam}`,
      { headers },
    )
    if (res.ok) repinGauges.value = await res.json()
  } catch { /* non-fatal */ }
}

async function lookupRiverName(): Promise<void> {
  const comid = repinUpComID.value
  if (!comid) return
  const lat = repinStartLat.value ?? reach.value?.put_in_lat
  const lng = repinStartLng.value ?? reach.value?.put_in_lng
  riverNameLooking.value = true
  try {
    const params = new URLSearchParams({ comid })
    if (lat != null && lng != null) { params.set('lat', String(lat)); params.set('lng', String(lng)) }
    const res = await fetch(`${apiBase}/api/v1/nldi/river-name?${params}`)
    if (res.ok) {
      const data = await res.json()
      if (data.river_name) form.value.riverName = data.river_name
    }
  } catch { /* non-fatal */ }
  finally { riverNameLooking.value = false }
}

// ── Geometry pick handlers ────────────────────────────────────────────────────

function resetGeometryState() {
  repinPickMode.value       = false
  repinAnchorSnap.value     = null
  repinAnchorSnapping.value = false
  repinAnchorError.value    = ''
  repinTributaries.value    = null
  repinDownstream.value     = null
  repinComIDEditMode.value  = null
  repinStartLat.value       = null
  repinStartLng.value       = null
  repinEndLat.value         = null
  repinEndLng.value         = null
  repinPreviewCenterline.value = null
  repinFlowlinesDirty.value = false
  repinError.value          = ''
  repinSuccess.value        = ''
}

function revertComIDs() {
  repinUpComID.value       = repinOrigUpComID.value
  repinDownComID.value     = repinOrigDownComID.value
  repinComIDEditMode.value = null
  repinStartLat.value      = null
  repinStartLng.value      = null
  repinEndLat.value        = null
  repinEndLng.value        = null
  repinPreviewCenterline.value = null
  repinFlowlinesDirty.value = false
  repinError.value         = ''
  repinSuccess.value       = ''
}

function togglePickMode() {
  if (repinPickMode.value) { repinPickMode.value = false; repinAnchorError.value = ''; return }
  repinGaugeSelectMode.value = false
  repinPickMode.value        = true
  repinAnchorError.value     = ''
}

async function onAnchorPick(lat: number, lng: number) {
  repinPickMode.value       = false
  repinAnchorSnapping.value = true
  repinAnchorSnap.value     = null
  repinTributaries.value    = null
  repinDownstream.value     = null
  repinAnchorError.value    = ''
  repinPreviewCenterline.value = null
  try {
    const snap = await fetchTributaries(lat, lng)
    if (!snap) return
    repinAnchorSnap.value    = snap
    repinComIDEditMode.value = 'up'
    await fetchDownstream(snap.comid)
    fetchNearbyGauges(lat, lng, snap.comid)
  } finally {
    repinAnchorSnapping.value = false
  }
}

function onComIDSelect(comid: string, lat: number, lng: number) {
  if (!repinComIDEditMode.value) return
  if (repinComIDEditMode.value === 'up') {
    repinUpComID.value      = comid
    repinStartLat.value     = lat
    repinStartLng.value     = lng
    repinComIDEditMode.value = 'down'
  } else {
    repinDownComID.value = comid
    repinEndLat.value    = lat
    repinEndLng.value    = lng
  }
  repinFlowlinesDirty.value    = true
  repinPreviewCenterline.value = null
}

async function previewCenterline() {
  if (!repinUpComID.value || !repinDownComID.value) return
  repinPreviewLoading.value    = true
  repinPreviewCenterline.value = null
  try {
    const headers = await authHeaders()
    const startLat = repinStartLat.value ?? reach.value?.put_in_lat
    const startLng = repinStartLng.value ?? reach.value?.put_in_lng
    const endLat   = repinEndLat.value   ?? reach.value?.take_out_lat
    const endLng   = repinEndLng.value   ?? reach.value?.take_out_lng
    let url = `${apiBase}/api/v1/nldi/preview-centerline?up_comid=${repinUpComID.value}&down_comid=${repinDownComID.value}`
    if (startLat != null && startLng != null && endLat != null && endLng != null)
      url += `&start_lat=${startLat}&start_lng=${startLng}&end_lat=${endLat}&end_lng=${endLng}`
    const res = await fetch(url, { headers })
    if (res.ok) repinPreviewCenterline.value = (await res.json()).geojson ?? null
  } catch { /* non-fatal */ }
  finally { repinPreviewLoading.value = false }
}

async function saveFlowLines() {
  if (!repinUpComID.value || !repinDownComID.value) return
  repinError.value   = ''
  repinSuccess.value = ''
  repinSaving.value  = true
  try {
    if (!repinPreviewCenterline.value) {
      await previewCenterline()
      if (!repinPreviewCenterline.value) { repinError.value = 'Preview failed — try "Preview centerline" first.'; return }
    }
    const headers = { 'Content-Type': 'application/json', ...(await authHeaders()) }
    const putIn   = repinPutInPin.value
    const takeOut = repinTakeOutPin.value
    const patchRes = await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}`, {
      method: 'PATCH',
      headers,
      body: JSON.stringify({
        put_in:     putIn   ? { lat: putIn.lat,   lng: putIn.lng   } : undefined,
        take_out:   takeOut ? { lat: takeOut.lat, lng: takeOut.lng } : undefined,
        up_comid:   repinUpComID.value,
        down_comid: repinDownComID.value,
      }),
    })
    if (!patchRes.ok) { repinError.value = `Save failed: HTTP ${patchRes.status}`; return }

    const clRes = await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}/centerline`, {
      method: 'POST',
      headers,
      body: JSON.stringify({ geojson: repinPreviewCenterline.value }),
    })
    if (!clRes.ok) { repinError.value = `Centerline save failed: HTTP ${clRes.status}`; return }

    repinOrigUpComID.value   = repinUpComID.value
    repinOrigDownComID.value = repinDownComID.value
    repinFlowlinesDirty.value = false
    repinSuccess.value = 'Flow lines saved'
    await load()
  } finally {
    repinSaving.value = false
  }
}

async function clearCenterline() {
  if (!confirm('Clear the centerline? This cannot be undone.')) return
  try {
    const headers = await authHeaders()
    await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}/centerline`, { method: 'DELETE', headers })
    resetGeometryState()
    await load()
  } catch { /* non-fatal */ }
}

// ── Gauge handlers ────────────────────────────────────────────────────────────

function onGaugeSelect(externalId: string, source: string, name: string, lat: number, lng: number) {
  repinGaugeSelectMode.value = false
  repinGaugeError.value      = ''
  repinPendingGauge.value    = { externalId, source, name, lat, lng }
}

async function saveGauge() {
  if (!repinPendingGauge.value) return
  repinGaugeSaving.value = true
  repinGaugeError.value  = ''
  const { externalId, source, name, lat, lng } = repinPendingGauge.value
  try {
    const headers = { 'Content-Type': 'application/json', ...(await authHeaders()) }
    const res = await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}/gauge`, {
      method: 'PUT', headers,
      body:   JSON.stringify({ external_id: externalId, source, name, lat, lng }),
    })
    if (!res.ok) { repinGaugeError.value = `HTTP ${res.status}`; return }
    repinPendingGauge.value = null
    await load()
  } catch (e: any) {
    repinGaugeError.value = e?.message ?? 'Failed to set gauge'
  } finally {
    repinGaugeSaving.value = false
  }
}

async function clearGauge() {
  if (!confirm('Remove the linked gauge?')) return
  repinGaugeSaving.value = true
  try {
    const headers = await authHeaders()
    await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}/gauge`, { method: 'DELETE', headers })
    await load()
  } catch { /* non-fatal */ }
  finally { repinGaugeSaving.value = false }
}

async function loadCustomGauges() {
  const headers = await authHeaders()
  const res = await fetch(`${apiBase}/api/v1/me/custom-gauges`, { headers }).catch(() => null)
  if (!res?.ok) return
  const data = await res.json()
  customGauges.value = data.items ?? []
}

async function linkCustomGauge(cg: CustomGaugeSummary) {
  customGaugeSaving.value = true
  customGaugePickerOpen.value = false
  try {
    const headers = await authHeaders()
    const res = await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}/gauge`, {
      method: 'PUT',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ custom_gauge_id: cg.id }),
    })
    if (res.ok) await load()
  } catch { /* non-fatal */ }
  finally { customGaugeSaving.value = false }
}

// ── Load ──────────────────────────────────────────────────────────────────────

function populateForm(r: UserReachDetail) {
  form.value.name      = r.name
  form.value.riverName = r.river_name ?? ''
  form.value.note      = r.note ?? ''
  const low     = r.flow_ranges.find(x => x.label === 'low')
  const running = r.flow_ranges.find(x => x.label === 'running')
  const high    = r.flow_ranges.find(x => x.label === 'high')
  form.value.ranges = {
    low:     { min: null,                       max: low?.max_value      ?? null },
    running: { min: running?.min_value ?? null,  max: running?.max_value ?? null },
    high:    { min: high?.min_value    ?? null,  max: null               },
  }
}

async function load() {
  loading.value = true
  error.value   = ''
  try {
    const headers = await authHeaders()
    const res = await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}`, { headers })
    if (res.status === 404) { error.value = 'Reach not found.'; return }
    if (!res.ok) throw new Error(`${res.status}`)
    const data: UserReachDetail = await res.json()
    reach.value = data
    populateForm(data)

    // Seed ComID state from saved reach data.
    repinUpComID.value       = data.up_comid
    repinDownComID.value     = data.down_comid
    repinOrigUpComID.value   = data.up_comid
    repinOrigDownComID.value = data.down_comid

    // Auto-load NHD network when saved ComIDs exist.
    if (data.up_comid && data.put_in_lat != null) {
      repinAnchorSnap.value    = { comid: data.up_comid, name: data.river_name ?? '' }
      repinComIDEditMode.value = null
      fetchDownstream(data.up_comid)
      fetchTributaries(data.put_in_lat, data.put_in_lng)
      fetchNearbyGauges(data.put_in_lat, data.put_in_lng, data.up_comid)
    }
  } catch (e: any) {
    error.value = e.message ?? 'Failed to load.'
  } finally {
    loading.value = false
  }
}

onMounted(() => { load(); loadCustomGauges() })

// ── Save metadata + flow bands ────────────────────────────────────────────────

async function save() {
  if (!form.value.name.trim()) return
  saving.value    = true
  saveError.value = ''
  try {
    const headers = { 'Content-Type': 'application/json', ...(await authHeaders()) }

    const patchRes = await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}`, {
      method: 'PATCH', headers,
      body: JSON.stringify({
        name:       form.value.name.trim(),
        river_name: form.value.riverName.trim() || null,
        note:       form.value.note.trim()      || null,
      }),
    })
    if (!patchRes.ok) throw new Error(`Save failed: ${patchRes.status}`)

    const r = form.value.ranges
    await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}/flow-ranges`, {
      method: 'PUT', headers,
      body: JSON.stringify({
        low:     { min_value: null,           max_value: r.low.max     },
        running: { min_value: r.running.min,  max_value: r.running.max },
        high:    { min_value: r.high.min,     max_value: null          },
      }),
    })

    await load()
  } catch (e: any) {
    saveError.value = e.message ?? 'Save failed.'
  } finally {
    saving.value = false
  }
}

// ── Delete ────────────────────────────────────────────────────────────────────

async function confirmDelete() {
  if (!confirm(`Delete "${reach.value?.name}"? This cannot be undone.`)) return
  const headers = await authHeaders()
  const res = await fetch(`${apiBase}/api/v1/me/reaches/${slug.value}`, { method: 'DELETE', headers })
  if (res.ok) router.push('/my/reaches')
}
</script>
