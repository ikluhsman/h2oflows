<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">

    <!-- Nav bar -->
    <header class="sticky top-0 z-20 border-b border-gray-200 dark:border-gray-800 bg-white/90 dark:bg-gray-950/90 backdrop-blur-sm">
      <div class="max-w-5xl mx-auto px-3 py-3 flex items-center gap-3">
        <NuxtLink to="/" class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 flex items-center gap-1">
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 5l-7 7 7 7"/></svg>
          Dashboard
        </NuxtLink>
        <span class="text-gray-300 dark:text-gray-700">/</span>
        <span class="text-sm font-medium truncate">{{ reach?.common_name ?? reach?.name }}</span>
      </div>
    </header>

    <!-- Admin bar -->
    <div v-if="reach" class="shrink-0 border-b border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900/60">
      <div class="max-w-5xl mx-auto px-3 py-2 flex items-center gap-4 flex-wrap">
        <!-- Fetch river line -->
        <div class="flex items-center gap-2">
          <div v-if="needsCoordsInput" class="flex items-center gap-1.5">
            <input
              v-model="manualLat"
              type="text"
              placeholder="lat"
              class="text-xs border border-gray-200 dark:border-gray-700 rounded px-2 py-1 w-24 bg-white dark:bg-gray-900 text-gray-700 dark:text-gray-200"
            />
            <input
              v-model="manualLng"
              type="text"
              placeholder="lng"
              class="text-xs border border-gray-200 dark:border-gray-700 rounded px-2 py-1 w-28 bg-white dark:bg-gray-900 text-gray-700 dark:text-gray-200"
            />
          </div>
          <button
            class="text-xs text-sky-500 hover:text-sky-600 dark:text-sky-400 dark:hover:text-sky-300 flex items-center gap-1 disabled:opacity-50"
            :disabled="fetchingCenterline || (needsCoordsInput && (!manualLat || !manualLng))"
            @click="fetchCenterline"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>
            <span v-if="fetchingCenterline">Fetching…</span>
            <span v-else-if="displayCenterline">Re-fetch river line</span>
            <span v-else>Fetch river line</span>
          </button>
          <span v-if="centerlineError" class="text-xs text-red-500">{{ centerlineError }}</span>
        </div>

        <span class="text-gray-200 dark:text-gray-700">|</span>

        <!-- Import KMZ -->
        <div class="flex items-center gap-2">
          <button
            class="text-xs text-sky-500 hover:text-sky-600 dark:text-sky-400 dark:hover:text-sky-300 flex items-center gap-1"
            @click="showImport = !showImport"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
            Import KMZ
          </button>
          <template v-if="showImport">
            <input
              ref="kmzInput"
              type="file"
              accept=".kml,.kmz"
              class="text-xs text-gray-500 file:mr-2 file:py-1 file:px-2 file:rounded file:border-0 file:text-xs file:bg-gray-100 dark:file:bg-gray-800 file:text-gray-700 dark:file:text-gray-300 hover:file:bg-gray-200 dark:hover:file:bg-gray-700 cursor-pointer"
              @change="onKmzSelect"
            />
            <UButton v-if="kmzFile" size="xs" color="primary" :loading="importing" @click="runImport">
              Import
            </UButton>
          </template>
        </div>

        <!-- Import result inline -->
        <div v-if="importResult" class="text-xs text-gray-500 flex items-center gap-2">
          <span class="text-emerald-500 font-medium">✓ Imported</span>
          <template v-for="(r, slug) in importResult.reaches" :key="slug">
            rapids: {{ r.rapids }}, put-ins: {{ r.put_ins }}, take-outs: {{ r.take_outs }}, parking: {{ r.parking }}
            <span v-if="r.errors?.length" class="text-red-500"> · {{ r.errors.length }} error(s)</span>
          </template>
        </div>
        <p v-if="importError" class="text-xs text-red-500">{{ importError }}</p>
      </div>
    </div>

    <div v-if="pending" class="max-w-5xl mx-auto px-3 py-12 text-center text-gray-400">
      Loading…
    </div>

    <div v-else-if="!reach" class="max-w-5xl mx-auto px-3 py-12 text-center text-gray-400">
      Reach not found.
    </div>

    <main v-else class="max-w-5xl mx-auto px-3 py-6 space-y-8">

      <!-- Hero -->
      <section>
        <div class="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <div v-if="reach.river_name" class="text-xs font-medium text-blue-500 uppercase tracking-wide mb-1">{{ reach.river_name }}</div>
            <h1 class="text-2xl font-bold">
              <template v-if="reach.put_in_name && reach.take_out_name">
                {{ reach.put_in_name }} to {{ reach.take_out_name }}
                <span v-if="reach.common_name" class="font-normal text-gray-400">({{ reach.common_name }})</span>
              </template>
              <template v-else>{{ reach.common_name ?? reach.name }}</template>
            </h1>
            <p class="text-gray-500 text-sm mt-0.5">
              {{ reach.region }}
              <span v-if="reach.length_mi"> · {{ reach.length_mi }} mi</span>
            </p>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <span class="rounded-lg bg-gray-100 dark:bg-gray-800 px-3 py-1.5 font-bold text-sm">{{ classLabel }}</span>
          </div>
        </div>
      </section>

      <!-- 48h graph + current CFS -->
      <section v-if="reach.gauge.id" class="border border-gray-200 dark:border-gray-700 rounded-xl p-4">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">48-Hour Flow</h2>
        <GaugeGraph :gauge-id="reach.gauge.id" :current-cfs="reach.gauge.current_cfs" />
        <!-- Current reading + status pill below graph -->
        <div v-if="reach.gauge.current_cfs != null" class="mt-3 flex items-end gap-2 border-t border-gray-100 dark:border-gray-800 pt-3">
          <span class="text-3xl font-bold tabular-nums" :class="cfsClass">
            {{ reach.gauge.current_cfs.toLocaleString() }}
          </span>
          <span class="text-gray-500 mb-0.5">cfs</span>
          <UBadge v-if="reach.gauge.flow_status" :color="statusColor" variant="subtle" size="sm" class="mb-0.5">{{ statusLabel }}</UBadge>
          <span v-if="reach.gauge.last_reading_at" class="text-xs text-gray-400 mb-1">
            · {{ lastReadingRelative }}
          </span>
        </div>
        <div v-else class="mt-3 text-gray-400 text-sm border-t border-gray-100 dark:border-gray-800 pt-3">No recent gauge reading</div>
      </section>

      <!-- Reach map -->
      <section>
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Map</h2>
        <ClientOnly>
          <ReachMap
            :name="reach.name"
            :class-max="reach.class_max"
            :centerline="displayCenterline"
            :rapids="reach.rapids"
            :access="reach.access"
            :gauge-lng="reach.gauge.lng"
            :gauge-lat="reach.gauge.lat"
          />
        </ClientOnly>
      </section>

      <!-- River assistant -->
      <section class="border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden">
        <button
          class="w-full flex items-center justify-between px-4 py-3 bg-gray-50 dark:bg-gray-900 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          @click="showChat = !showChat"
        >
          <div class="flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-sky-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
            <span class="text-sm font-semibold text-gray-700 dark:text-gray-200">Ask about this reach</span>
            <span class="text-xs text-gray-400 hidden sm:inline">· AI answers based on reach data</span>
          </div>
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-gray-400 transition-transform" :class="{ 'rotate-180': showChat }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M6 9l6 6 6-6"/>
          </svg>
        </button>

        <div v-if="showChat" class="p-4 space-y-4">
          <!-- Message thread -->
          <div v-if="chatMessages.length" class="space-y-3 max-h-96 overflow-y-auto pr-1">
            <div
              v-for="(msg, i) in chatMessages"
              :key="i"
              class="flex gap-2.5"
              :class="msg.role === 'user' ? 'justify-end' : 'justify-start'"
            >
              <!-- Assistant avatar -->
              <div v-if="msg.role === 'assistant'" class="w-6 h-6 rounded-full bg-sky-100 dark:bg-sky-900 flex items-center justify-center shrink-0 mt-0.5">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5 text-sky-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
                </svg>
              </div>

              <div
                class="rounded-xl px-3 py-2 text-sm max-w-prose"
                :class="msg.role === 'user'
                  ? 'bg-sky-500 text-white rounded-br-sm'
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-200 rounded-bl-sm'"
              >
                <div class="whitespace-pre-wrap">{{ msg.content }}</div>
              </div>
            </div>

            <!-- Typing indicator -->
            <div v-if="chatLoading" class="flex gap-2.5 justify-start">
              <div class="w-6 h-6 rounded-full bg-sky-100 dark:bg-sky-900 flex items-center justify-center shrink-0">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5 text-sky-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
                </svg>
              </div>
              <div class="rounded-xl rounded-bl-sm bg-gray-100 dark:bg-gray-800 px-3 py-2">
                <span class="flex gap-1 items-center h-5">
                  <span class="w-1.5 h-1.5 rounded-full bg-gray-400 animate-bounce" style="animation-delay: 0ms"/>
                  <span class="w-1.5 h-1.5 rounded-full bg-gray-400 animate-bounce" style="animation-delay: 150ms"/>
                  <span class="w-1.5 h-1.5 rounded-full bg-gray-400 animate-bounce" style="animation-delay: 300ms"/>
                </span>
              </div>
            </div>
          </div>

          <!-- Suggested questions (only before first message) -->
          <div v-if="!chatMessages.length" class="flex flex-wrap gap-2">
            <button
              v-for="q in suggestedQuestions"
              :key="q"
              class="text-xs rounded-full border border-gray-200 dark:border-gray-700 px-3 py-1.5 text-gray-600 dark:text-gray-400 hover:border-sky-400 hover:text-sky-600 dark:hover:text-sky-400 transition-colors"
              @click="sendQuestion(q)"
            >
              {{ q }}
            </button>
          </div>

          <!-- Input -->
          <form class="flex gap-2" @submit.prevent="sendQuestion(chatInput)">
            <input
              v-model="chatInput"
              type="text"
              placeholder="Ask anything about this reach…"
              :disabled="chatLoading"
              class="flex-1 text-sm rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-gray-800 dark:text-gray-200 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-sky-500 disabled:opacity-50"
            />
            <button
              type="submit"
              :disabled="chatLoading || !chatInput.trim()"
              class="rounded-lg bg-sky-500 hover:bg-sky-600 disabled:opacity-40 px-3 py-2 text-white transition-colors"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
              </svg>
            </button>
          </form>

          <p v-if="chatError" class="text-xs text-red-500">{{ chatError }}</p>
        </div>
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


      <!-- Related reaches -->
      <section v-if="reach.related?.length > 0">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Related Reaches</h2>
        <div class="flex flex-wrap gap-2">
          <NuxtLink
            v-for="rel in reach.related"
            :key="rel.slug"
            :to="`/reaches/${rel.slug}`"
            class="flex items-center gap-2 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800 px-3 py-2 transition-colors"
          >
            <span class="text-xs text-gray-400">
              <template v-if="rel.relationship === 'upstream'">↑</template>
              <template v-else-if="rel.relationship === 'downstream'">↓</template>
              <template v-else-if="rel.relationship === 'tributary'">⤷</template>
              <template v-else>↔</template>
            </span>
            <span class="text-sm font-medium">{{ rel.name }}</span>
            <span class="text-xs text-gray-400 capitalize">{{ rel.relationship }}</span>
          </NuxtLink>
        </div>
      </section>

      <!-- Gauge attribution -->
      <section v-if="reach.gauge.external_id" class="text-xs text-gray-400">
        Flow data: {{ reach.gauge.source?.toUpperCase() }} gauge {{ reach.gauge.external_id }}
        <span v-if="reach.gauge.name"> · {{ reach.gauge.name }}</span>
      </section>

    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const route  = useRoute()
const config = useRuntimeConfig()

// ---- Data -------------------------------------------------------------------

const { data: reach, pending, refresh: refreshReach } = await useAsyncData(
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
  if (!reach.value) return 'H2OFlows'
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

// ---- Flow band helpers -------------------------------------------------------

function bandDisplayLabel(label: string): string {
  return label.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
}

// ---- OSM centerline fetch ---------------------------------------------------

const fetchingCenterline  = ref(false)
const centerlineError     = ref<string | null>(null)
const liveCenterline      = ref<any>(null)
const manualLat           = ref('')
const manualLng           = ref('')

// Show the lat/lng input after the server tells us it has no location to work from.
const needsCoordsInput = computed(() =>
  centerlineError.value?.includes('no location available') ?? false
)

const displayCenterline = computed(() =>
  liveCenterline.value ?? (reach.value as any)?.centerline ?? null
)

async function fetchCenterline() {
  fetchingCenterline.value = true
  centerlineError.value = null
  try {
    let url = `${config.public.apiBase}/api/v1/reaches/${route.params.slug}/fetch-centerline`
    if (manualLat.value && manualLng.value) {
      url += `?lat=${encodeURIComponent(manualLat.value)}&lng=${encodeURIComponent(manualLng.value)}`
    }
    const res = await fetch(url, { method: 'POST' })
    const text = await res.text()
    let json: any
    try { json = JSON.parse(text) } catch { json = null }
    if (!res.ok || !json) {
      centerlineError.value = json?.error ?? `Server error ${res.status}`
    } else {
      liveCenterline.value = json.centerline
    }
  } catch (err: any) {
    centerlineError.value = err?.message ?? 'Network error'
  } finally {
    fetchingCenterline.value = false
  }
}

// ---- River assistant chat ---------------------------------------------------

const showChat      = ref(false)
const chatMessages  = ref<{ role: 'user' | 'assistant'; content: string }[]>([])
const chatInput     = ref('')
const chatLoading   = ref(false)
const chatError     = ref<string | null>(null)

const suggestedQuestions = computed(() => {
  const r = reach.value as any
  if (!r) return []
  const base = [
    'What flows are best?',
    'What should I scout?',
    'How do I get to the put-in?',
  ]
  if ((r.class_max ?? 0) >= 4) base.push('How committing is this run?')
  return base
})

async function sendQuestion(question: string) {
  const q = question.trim()
  if (!q || chatLoading.value) return
  chatInput.value = ''
  chatError.value = null
  chatMessages.value.push({ role: 'user', content: q })
  chatLoading.value = true
  try {
    const res = await fetch(
      `${config.public.apiBase}/api/v1/reaches/${route.params.slug}/ask`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question: q }),
      }
    )
    const json = await res.json()
    if (!res.ok) throw new Error(json.error ?? `Server error ${res.status}`)
    chatMessages.value.push({ role: 'assistant', content: json.answer })
  } catch (err: any) {
    chatError.value = err?.message ?? 'Something went wrong'
    chatMessages.value.pop() // remove the user message if we got nothing back
  } finally {
    chatLoading.value = false
  }
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
      // Hard reload to pick up new access points / rapids on the map
      window.location.reload()
    }
  } catch (err: any) {
    importError.value = err?.message ?? 'Network error'
  } finally {
    importing.value = false
  }
}
</script>
