<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">

    <!-- Nav bar -->
    <header class="sticky top-0 z-20 border-b border-gray-200 dark:border-gray-800 bg-white/90 dark:bg-gray-950/90 backdrop-blur-sm">
      <div class="max-w-3xl mx-auto px-4 py-3 flex items-center gap-3">
        <NuxtLink to="/" class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 flex items-center gap-1">
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 5l-7 7 7 7"/></svg>
          Dashboard
        </NuxtLink>
        <span class="text-gray-300 dark:text-gray-700">/</span>
        <span class="text-sm font-medium truncate">{{ reach?.name }}</span>
      </div>
    </header>

    <div v-if="pending" class="max-w-3xl mx-auto px-4 py-12 text-center text-gray-400">
      Loading…
    </div>

    <div v-else-if="!reach" class="max-w-3xl mx-auto px-4 py-12 text-center text-gray-400">
      Reach not found.
    </div>

    <main v-else class="max-w-3xl mx-auto px-4 py-6 space-y-8">

      <!-- Hero -->
      <section>
        <div class="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <h1 class="text-2xl font-bold">{{ reach.name }}</h1>
            <p class="text-gray-500 text-sm mt-0.5">
              {{ reach.region }}
              <span v-if="reach.length_mi"> · {{ reach.length_mi }} mi</span>
            </p>
          </div>

          <div class="flex items-center gap-2 flex-shrink-0">
            <!-- Class badge -->
            <span class="rounded-lg bg-gray-100 dark:bg-gray-800 px-3 py-1.5 font-bold text-sm">
              {{ classLabel }}
            </span>
            <!-- Flow status -->
            <UBadge v-if="reach.gauge.flow_status" :color="statusColor" variant="subtle" size="sm">
              {{ statusLabel }}
            </UBadge>
          </div>
        </div>

        <!-- Current CFS — prominent if we have a live reading -->
        <div v-if="reach.gauge.current_cfs != null" class="mt-4 flex items-end gap-2">
          <span class="text-4xl font-bold tabular-nums" :class="cfsClass">
            {{ reach.gauge.current_cfs.toLocaleString() }}
          </span>
          <span class="text-gray-500 mb-1">cfs</span>
          <span v-if="reach.gauge.last_reading_at" class="text-xs text-gray-400 mb-1.5">
            · {{ lastReadingRelative }}
          </span>
        </div>
        <div v-else class="mt-4 text-gray-400 text-sm">No recent gauge reading</div>
      </section>

      <!-- 48h graph + diurnal banner -->
      <section v-if="reach.gauge.id" class="border border-gray-200 dark:border-gray-700 rounded-xl p-4">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">48-Hour Flow</h2>
        <GaugeGraph :gauge-id="reach.gauge.id" :current-cfs="reach.gauge.current_cfs" />
      </section>

      <!-- Description -->
      <section v-if="reach.description">
        <div class="flex items-center gap-2 mb-2">
          <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide">About</h2>
          <DataSourceBadge
            :source="(reach.description_source as any) ?? 'ai_seed'"
            :verified="reach.description_verified"
            :confidence="reach.description_ai_confidence ?? undefined"
          />
        </div>
        <div class="prose prose-sm dark:prose-invert max-w-none text-gray-700 dark:text-gray-300 whitespace-pre-line">
          {{ reach.description }}
        </div>
      </section>

      <!-- Reach map -->
      <section v-if="reach.centerline || reach.rapids.some((r: any) => r.lng) || reach.access.some((a: any) => a.water_lng)">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Map</h2>
        <MapReachMap
          :centerline="reach.centerline"
          :rapids="reach.rapids"
          :access="reach.access"
        />
      </section>

      <!-- Access points -->
      <section v-if="reach.access.length > 0">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Access</h2>
        <div class="space-y-3">
          <div
            v-for="a in reach.access"
            :key="a.id"
            class="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-xs font-bold uppercase tracking-wide" :class="accessTypeClass(a.access_type)">
                    {{ accessTypeLabel(a.access_type) }}
                  </span>
                  <span class="font-semibold">{{ a.name ?? '—' }}</span>
                  <span v-if="a.entry_style" class="text-xs bg-gray-100 dark:bg-gray-800 rounded px-1.5 py-0.5 capitalize">
                    {{ a.entry_style.replace('_', ' ') }}
                  </span>
                </div>
              </div>
              <DataSourceBadge
                :source="(a.data_source as any)"
                :verified="a.verified"
                :confidence="a.ai_confidence ?? undefined"
              />
            </div>

            <p v-if="a.directions" class="text-sm text-gray-600 dark:text-gray-400 mt-2">
              {{ a.directions }}
            </p>

            <!-- Approach info for trail/technical -->
            <p v-if="a.approach_notes" class="text-sm text-gray-600 dark:text-gray-400 mt-1">
              {{ a.approach_notes }}
            </p>

            <!-- Metadata row -->
            <div class="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-400">
              <span v-if="a.road_type">Road: {{ a.road_type }}</span>
              <span v-if="a.approach_dist_mi">Walk: {{ a.approach_dist_mi }} mi</span>
              <span v-if="a.hike_to_water_min">~{{ a.hike_to_water_min }} min to water</span>
              <span v-if="a.parking_fee != null">Parking: {{ a.parking_fee === 0 ? 'Free' : `$${a.parking_fee}/day` }}</span>
              <span v-if="a.permit_required" class="text-amber-500">Permit required</span>
            </div>

            <p v-if="a.permit_info" class="text-xs text-amber-600 dark:text-amber-400 mt-1">
              {{ a.permit_info }}
              <a v-if="a.permit_url" :href="a.permit_url" target="_blank" class="underline ml-1">More info</a>
            </p>

            <p v-if="a.notes" class="text-xs text-gray-400 mt-1 italic">{{ a.notes }}</p>

            <!-- Waypoints for trail/technical access -->
            <div v-if="a.waypoints.length > 0" class="mt-3 border-t border-gray-100 dark:border-gray-800 pt-3 space-y-1.5">
              <p class="text-xs font-medium text-gray-500">Approach route:</p>
              <div
                v-for="wp in a.waypoints"
                :key="wp.sequence"
                class="flex gap-2 text-xs"
              >
                <span class="flex-shrink-0 w-5 h-5 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center font-bold text-gray-500">
                  {{ wp.sequence }}
                </span>
                <div>
                  <span class="font-medium">{{ wp.label }}</span>
                  <span v-if="wp.description" class="text-gray-400"> — {{ wp.description }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Rapids -->
      <section v-if="reach.rapids.length > 0">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">
          Rapids ({{ reach.rapids.length }})
        </h2>
        <div class="space-y-3">
          <div
            v-for="rapid in reach.rapids"
            :key="rapid.id"
            class="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-semibold">{{ rapid.name }}</span>
                  <span v-if="rapid.class_rating" class="text-xs bg-gray-100 dark:bg-gray-800 rounded px-1.5 py-0.5 font-bold">
                    Class {{ formatClass(rapid.class_rating) }}
                  </span>
                  <span
                    v-if="rapid.is_portage_recommended"
                    class="text-xs bg-red-50 dark:bg-red-950 text-red-600 dark:text-red-400 rounded px-1.5 py-0.5 font-medium"
                  >
                    Portage recommended
                  </span>
                </div>
                <p v-if="rapid.river_mile != null" class="text-xs text-gray-400 mt-0.5">
                  Mile {{ rapid.river_mile }}
                </p>
              </div>
              <DataSourceBadge
                :source="(rapid.data_source as any)"
                :verified="rapid.verified"
                :confidence="rapid.ai_confidence ?? undefined"
              />
            </div>

            <p v-if="rapid.description" class="text-sm text-gray-600 dark:text-gray-400 mt-2">
              {{ rapid.description }}
            </p>

            <div v-if="rapid.class_at_low || rapid.class_at_high" class="mt-2 flex gap-3 text-xs text-gray-400">
              <span v-if="rapid.class_at_low">Low: Class {{ formatClass(rapid.class_at_low) }}</span>
              <span v-if="rapid.class_at_high">High: Class {{ formatClass(rapid.class_at_high) }}</span>
            </div>

            <div v-if="rapid.portage_description" class="mt-2 text-xs text-amber-600 dark:text-amber-400">
              Portage: {{ rapid.portage_description }}
            </div>
          </div>
        </div>
      </section>

      <!-- Flow ranges -->
      <section v-if="(flowRanges?.length ?? 0) > 0">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Flow Bands</h2>
        <div class="rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
          <div
            v-for="band in (flowRanges ?? [])"
            :key="band.label"
            class="flex items-center justify-between px-4 py-2.5 border-b border-gray-100 dark:border-gray-800 last:border-0 transition-colors"
            :class="band.label === activeBand ? activeBandClass(band.label) : 'bg-white dark:bg-gray-900'"
          >
            <div class="flex items-center gap-2">
              <span
                class="text-xs font-bold uppercase tracking-wide"
                :class="bandLabelClass(band.label)"
              >{{ bandDisplayLabel(band.label) }}</span>
              <span v-if="band.label === activeBand" class="text-xs text-gray-400">← now</span>
              <span v-if="!band.verified" class="text-xs text-gray-400 italic">est.</span>
            </div>
            <span class="text-sm tabular-nums text-gray-500">{{ bandRange(band) }}</span>
          </div>
        </div>
      </section>

      <!-- Gauge attribution -->
      <section v-if="reach.gauge.external_id" class="text-xs text-gray-400">
        Flow data: {{ reach.gauge.source?.toUpperCase() }} gauge {{ reach.gauge.external_id }}
        <span v-if="reach.gauge.name"> · {{ reach.gauge.name }}</span>
      </section>

      <!-- KMZ Import -->
      <section class="border-t border-gray-100 dark:border-gray-800 pt-6 pb-8">
        <button
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 flex items-center gap-1.5"
          @click="showImport = !showImport"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/>
          </svg>
          Import KMZ / KML
        </button>

        <div v-if="showImport" class="mt-3 space-y-3">
          <p class="text-xs text-gray-400">
            Upload a Google My Maps KMZ export. Folders named <strong>Access Points</strong>, <strong>Rivers</strong>, and <strong>Rapids</strong> are supported. Rapids and access pins are matched to reaches automatically.
          </p>
          <div class="flex items-center gap-3 flex-wrap">
            <input
              ref="kmzInput"
              type="file"
              accept=".kml,.kmz"
              class="text-xs text-gray-500 file:mr-2 file:py-1 file:px-3 file:rounded file:border-0 file:text-xs file:bg-gray-100 dark:file:bg-gray-800 file:text-gray-700 dark:file:text-gray-300 hover:file:bg-gray-200 dark:hover:file:bg-gray-700 cursor-pointer"
              @change="onKmzSelect"
            />
            <UButton
              v-if="kmzFile"
              size="xs"
              color="primary"
              :loading="importing"
              @click="runImport"
            >
              Import
            </UButton>
          </div>

          <!-- Result -->
          <div v-if="importResult" class="rounded-lg bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 p-3 space-y-2">
            <p class="text-xs font-semibold text-gray-600 dark:text-gray-300">{{ importResult.map_name }}</p>
            <div v-for="(r, slug) in importResult.reaches" :key="slug" class="text-xs text-gray-500">
              <span class="font-medium text-gray-700 dark:text-gray-200">{{ r.name }}</span>
              — rapids: {{ r.rapids }}, put-ins: {{ r.put_ins }}, take-outs: {{ r.take_outs }}, parking: {{ r.parking }}
              <span v-if="r.errors?.length" class="text-red-500"> · {{ r.errors.length }} error(s)</span>
            </div>
            <p v-if="importError" class="text-xs text-red-500">{{ importError }}</p>
          </div>
          <p v-if="importError && !importResult" class="text-xs text-red-500">{{ importError }}</p>
        </div>
      </section>

    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const route  = useRoute()
const config = useRuntimeConfig()

// ---- Data -------------------------------------------------------------------

const { data: reach, pending } = await useAsyncData(
  `reach-${route.params.slug}`,
  () => $fetch(`${config.public.apiBase}/api/v1/reaches/${route.params.slug}`)
)

// Flow ranges — secondary fetch once we have the gauge ID
const { data: flowRanges } = await useAsyncData(
  `flow-ranges-${route.params.slug}`,
  async () => {
    const gaugeId = (reach.value as any)?.gauge?.id
    if (!gaugeId) return []
    return $fetch(`${config.public.apiBase}/api/v1/gauges/${gaugeId}/flow-ranges`)
  },
  { default: () => [] }
)

// ---- Derived display --------------------------------------------------------
// Declared before SEO so metaTitle/metaDesc can reference them without TDZ errors.

function romanClass(n: number): string {
  const map: Record<number, string> = {
    1: 'I', 1.5: 'I+', 2: 'II', 2.5: 'II+',
    3: 'III', 3.5: 'III+', 4: 'IV', 4.5: 'IV+',
    5: 'V', 5.5: 'V+', 6: 'VI',
  }
  return map[n] ?? String(n)
}

function formatClass(n: number): string { return romanClass(n) }

const classLabel = computed(() => {
  const r = reach.value
  if (!r?.class_min && !r?.class_max) return 'Unknown class'
  if (r.class_min === r.class_max)     return `Class ${romanClass(r.class_min!)}`
  return `Class ${romanClass(r.class_min!)}–${romanClass(r.class_max!)}`
})

const statusColor = computed(() => {
  switch (reach.value?.gauge.flow_status) {
    case 'runnable': return 'success'
    case 'caution':  return 'warning'
    case 'low':
    case 'flood':    return 'error'
    default:         return 'neutral'
  }
})

// Which flow band is currently active (matches current CFS)
const activeBand = computed(() => {
  const cfs = (reach.value as any)?.gauge?.current_cfs
  if (cfs == null) return null
  const bands = (flowRanges.value as any[]) ?? []
  for (const b of bands) {
    const aboveMin = b.min_cfs == null || cfs >= b.min_cfs
    const belowMax = b.max_cfs == null || cfs <  b.max_cfs
    if (aboveMin && belowMax) return b.label
  }
  return null
})

const statusLabel = computed(() => {
  if (activeBand.value) return bandDisplayLabel(activeBand.value)
  switch (reach.value?.gauge.flow_status) {
    case 'runnable': return 'Runnable'
    case 'caution':  return 'Caution'
    case 'low':      return 'Too Low'
    case 'flood':    return 'Flood Stage'
    default:         return 'Unknown'
  }
})

// ---- SEO --------------------------------------------------------------------

const metaTitle = computed(() => {
  if (!reach.value) return 'H2OFlow'
  const cfs = reach.value.gauge?.current_cfs
  return `${reach.value.name} | ${classLabel.value} | ${cfs != null ? `${cfs.toLocaleString()} cfs — ${statusLabel.value}` : reach.value.region}`
})

const metaDesc = computed(() => {
  if (!reach.value) return ''
  const cfs = reach.value.gauge?.current_cfs
  const parts = [
    reach.value.region,
    classLabel.value,
    reach.value.length_mi ? `${reach.value.length_mi} miles` : null,
    cfs != null ? `Currently ${cfs.toLocaleString()} cfs — ${statusLabel.value}` : null,
  ].filter(Boolean)
  return parts.join(' · ')
})

useSeoMeta({
  title:           () => metaTitle.value,
  ogTitle:         () => metaTitle.value,
  description:     () => metaDesc.value,
  ogDescription:   () => metaDesc.value,
})

const cfsClass = computed(() => ({
  'text-emerald-500': reach.value?.gauge.flow_status === 'runnable',
  'text-yellow-500':  reach.value?.gauge.flow_status === 'caution',
  'text-red-500':     ['low','flood'].includes(reach.value?.gauge.flow_status ?? ''),
  'text-gray-300':    reach.value?.gauge.flow_status === 'unknown',
}))

const lastReadingRelative = computed(() => {
  const t = reach.value?.gauge.last_reading_at
  if (!t) return ''
  const ms = Date.now() - new Date(t).getTime()
  const m = Math.floor(ms / 60_000)
  if (m < 1)  return 'just now'
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ${m % 60}m ago`
})

function accessTypeLabel(t: string): string {
  return { put_in: 'Put-in', take_out: 'Take-out', intermediate: 'Intermediate', shuttle_drop: 'Shuttle', camp: 'Camp' }[t] ?? t
}

function accessTypeClass(t: string): string {
  return {
    put_in:       'text-emerald-600 dark:text-emerald-400',
    take_out:     'text-blue-500 dark:text-blue-400',
    intermediate: 'text-gray-500',
    shuttle_drop: 'text-purple-500 dark:text-purple-400',
    camp:         'text-amber-500 dark:text-amber-400',
  }[t] ?? 'text-gray-500'
}

// ---- Flow band helpers -------------------------------------------------------

function bandDisplayLabel(label: string): string {
  return label.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
}

function bandLabelClass(label: string): string {
  return {
    too_low: 'text-gray-400',
    minimum: 'text-gray-600 dark:text-gray-300',
    fun:     'text-emerald-600 dark:text-emerald-400',
    optimal: 'text-emerald-600 dark:text-emerald-400',
    pushy:   'text-amber-500',
    high:    'text-blue-500',
    flood:   'text-blue-600',
  }[label] ?? 'text-gray-500'
}

function activeBandClass(label: string): string {
  return {
    too_low: 'bg-gray-50 dark:bg-gray-800/60',
    minimum: 'bg-gray-50 dark:bg-gray-800/60',
    fun:     'bg-emerald-50/60 dark:bg-emerald-950/30',
    optimal: 'bg-emerald-50/60 dark:bg-emerald-950/30',
    pushy:   'bg-amber-50/60 dark:bg-amber-950/30',
    high:    'bg-blue-50/60 dark:bg-blue-950/30',
    flood:   'bg-blue-50/60 dark:bg-blue-950/30',
  }[label] ?? 'bg-white dark:bg-gray-900'
}

function bandRange(band: any): string {
  if (band.min_cfs == null && band.max_cfs == null) return '—'
  if (band.min_cfs == null) return `< ${band.max_cfs.toLocaleString()} cfs`
  if (band.max_cfs == null) return `${band.min_cfs.toLocaleString()}+ cfs`
  return `${band.min_cfs.toLocaleString()}–${band.max_cfs.toLocaleString()} cfs`
}

// ---- KMZ import -------------------------------------------------------------

const showImport  = ref(false)
const kmzFile     = ref<File | null>(null)
const kmzInput    = ref<HTMLInputElement>()
const importing   = ref(false)
const importResult = ref<any>(null)
const importError  = ref<string | null>(null)

function onKmzSelect(e: Event) {
  const input = e.target as HTMLInputElement
  kmzFile.value = input.files?.[0] ?? null
  importResult.value = null
  importError.value = null
}

async function runImport() {
  if (!kmzFile.value) return
  importing.value = true
  importError.value = null
  importResult.value = null
  try {
    const form = new FormData()
    form.append('file', kmzFile.value)
    const res = await fetch(`${config.public.apiBase}/api/v1/import/kmz`, {
      method: 'POST',
      body: form,
    })
    const json = await res.json()
    if (!res.ok) {
      importError.value = json.error ?? `Server error ${res.status}`
    } else {
      importResult.value = json
    }
  } catch (err: any) {
    importError.value = err?.message ?? 'Network error'
  } finally {
    importing.value = false
  }
}
</script>
