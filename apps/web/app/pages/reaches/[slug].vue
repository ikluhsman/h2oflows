<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">

    <AppHeader>
      <template v-if="reach">
        <span class="text-gray-300 dark:text-gray-700 shrink-0">/</span>
        <span class="text-sm font-medium truncate text-gray-700 dark:text-gray-200">{{ reach.common_name ?? reach.name }}</span>
      </template>
      <template #actions>
        <button
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-xs font-semibold transition-colors shrink-0"
          @click="openShareForm"
        >
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/>
          </svg>
          Share
        </button>
      </template>
    </AppHeader>

    <!-- Upstream / downstream pagination -->
    <div v-if="upstreamReach || downstreamReach" class="border-b border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-950">
      <div class="max-w-5xl mx-auto px-3 py-2 flex items-center justify-between gap-2">
        <!-- Upstream (left) -->
        <NuxtLink
          v-if="upstreamReach"
          :to="`/reaches/${upstreamReach.slug}`"
          class="flex items-center gap-1.5 min-w-0 text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors group"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 shrink-0 text-gray-400 group-hover:text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 5l-7 7 7 7"/></svg>
          <div class="min-w-0">
            <div class="text-[10px] text-gray-400 uppercase tracking-wide leading-none mb-0.5">upstream</div>
            <div class="text-sm font-medium truncate">{{ upstreamReach.name }}</div>
          </div>
        </NuxtLink>
        <div v-else class="flex-1" />

        <!-- Downstream (right) -->
        <NuxtLink
          v-if="downstreamReach"
          :to="`/reaches/${downstreamReach.slug}`"
          class="flex items-center gap-1.5 min-w-0 text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors group text-right"
        >
          <div class="min-w-0">
            <div class="text-[10px] text-gray-400 uppercase tracking-wide leading-none mb-0.5">downstream</div>
            <div class="text-sm font-medium truncate">{{ downstreamReach.name }}</div>
          </div>
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 shrink-0 text-gray-400 group-hover:text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
        </NuxtLink>
        <div v-else class="flex-1" />
      </div>
    </div>

    <!-- Admin bar — only visible to users with app_metadata.role === "admin" -->
    <!-- ClientOnly prevents SSR hydration mismatch (session only available client-side) -->
    <ClientOnly>
    <div v-if="reach && isAdmin" class="shrink-0 border-b border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900/60">
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

        <span class="text-gray-200 dark:text-gray-700">|</span>

        <!-- Delete reach -->
        <button
          class="text-xs text-red-400 hover:text-red-600 dark:hover:text-red-300 flex items-center gap-1 disabled:opacity-50"
          :disabled="deleting"
          @click="deleteReach"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/><path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
          <span v-if="deleting">Deleting…</span>
          <span v-else>Delete reach</span>
        </button>
      </div>
    </div>
    </ClientOnly>

    <div v-if="pending" class="max-w-5xl mx-auto px-3 py-12 text-center text-gray-400">
      Loading…
    </div>

    <div v-else-if="!reach" class="max-w-5xl mx-auto px-3 py-12 text-center text-gray-400">
      Reach not found.
    </div>

    <main v-else class="max-w-5xl mx-auto px-3 py-6 pb-20 sm:pb-6 space-y-8">

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
            </p>
          </div>

        </div>
      </section>

      <!-- Quick stats — consolidated -->
      <section>
        <div class="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 px-4 py-3">
          <div class="flex items-center divide-x divide-gray-200 dark:divide-gray-700 flex-wrap gap-y-3">
            <div class="pr-4">
              <div class="text-[10px] text-gray-400 uppercase tracking-wide mb-1">Difficulty</div>
              <div class="flex items-center gap-1.5">
                <span class="inline-block w-3 h-3 rounded-sm shrink-0" :style="{ backgroundColor: difficultyColor }" />
                <span class="text-xl font-bold" :class="difficultyTextClass">{{ classLabel }}</span>
              </div>
            </div>
            <div class="px-4">
              <div class="text-[10px] text-gray-400 uppercase tracking-wide mb-1">Length</div>
              <div class="text-xl font-bold text-gray-800 dark:text-gray-100">{{ reach.length_mi != null ? `${reach.length_mi} mi` : '—' }}</div>
            </div>
            <div class="px-4">
              <div class="text-[10px] text-gray-400 uppercase tracking-wide mb-1">Gradient</div>
              <div class="text-xl font-bold text-gray-800 dark:text-gray-100">{{ reach.gradient_fpm != null ? `${reach.gradient_fpm} ft/mi` : '—' }}</div>
            </div>
            <div v-if="allGauges.length > 0" class="pl-4 flex-1 flex items-center justify-between gap-3">
              <div>
                <div class="text-[10px] text-gray-400 uppercase tracking-wide mb-1">Flow</div>
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-xl font-bold tabular-nums" :class="cfsColorClass(allGauges[0].flow_status, allGauges[0].flow_band_label)">
                    {{ allGauges[0].current_cfs != null ? allGauges[0].current_cfs.toLocaleString() : '—' }}
                  </span>
                  <span class="text-xs text-gray-500">cfs</span>
                  <span :class="['hidden sm:inline-flex items-center rounded-md px-1.5 py-0.5 text-xs font-medium', flowBadgeClass(allGauges[0].flow_status, allGauges[0].flow_band_label)]">
                    {{ flowBandLabel(allGauges[0].flow_status, allGauges[0].flow_band_label) }}
                  </span>
                </div>
                <span :class="['inline-flex sm:hidden items-center rounded-md px-1.5 py-0.5 text-xs font-medium mt-1', flowBadgeClass(allGauges[0].flow_status, allGauges[0].flow_band_label)]">
                  {{ flowBandLabel(allGauges[0].flow_status, allGauges[0].flow_band_label) }}
                </span>
              </div>
              <button
                class="shrink-0 text-xs text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300 font-medium transition-colors"
                @click="openGaugeModal(allGauges[0])"
              >
                View flow →
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- River assistant — inline search box -->
      <section>
        <form class="flex gap-2" @submit.prevent="sendQuestion(chatInput)">
          <div class="relative flex-1">
            <svg xmlns="http://www.w3.org/2000/svg" class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-sky-400 pointer-events-none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
            <input
              v-model="chatInput"
              type="text"
              placeholder="Ask anything about this reach…"
              :disabled="chatLoading"
              class="w-full text-sm rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 pl-9 pr-3 py-2 text-gray-800 dark:text-gray-200 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-sky-500 disabled:opacity-50"
            />
          </div>
          <button
            type="submit"
            :disabled="chatLoading || !chatInput.trim()"
            class="shrink-0 rounded-lg bg-sky-500 hover:bg-sky-600 disabled:opacity-40 px-3 py-2 text-white transition-colors"
          >
            <svg v-if="!chatLoading" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
            </svg>
            <span v-else class="flex gap-1 items-center">
              <span class="w-1.5 h-1.5 rounded-full bg-white/80 animate-bounce" style="animation-delay:0ms"/>
              <span class="w-1.5 h-1.5 rounded-full bg-white/80 animate-bounce" style="animation-delay:150ms"/>
              <span class="w-1.5 h-1.5 rounded-full bg-white/80 animate-bounce" style="animation-delay:300ms"/>
            </span>
          </button>
        </form>

        <!-- Last assistant response -->
        <div v-if="lastAssistantMessage" class="mt-3 rounded-lg bg-gray-50 dark:bg-gray-900 border border-gray-100 dark:border-gray-800 px-4 py-3 text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap leading-relaxed">
          {{ lastAssistantMessage }}
        </div>
        <p v-if="chatError" class="mt-2 text-xs text-red-500">{{ chatError }}</p>
      </section>

      <!-- Reach map -->
      <section>
        <ClientOnly>
          <ReachMap
            :name="reach.name"
            :class-max="reach.class_max"
            :centerline="displayCenterline"
            :rapids="reach.rapids"
            :access="reach.access"
            :gauges="allGauges"
            :slug="(reach as any).slug"
            :river-name="(reach as any).river_name ?? undefined"
            @gauge-add="(id) => { const g = allGauges.find((x: any) => x.id === id); if (g) addToDashboard(g) }"
          />
        </ClientOnly>
      </section>

      <!-- Reach Description -->
      <section v-if="reach.description">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-2">Reach Description</h2>
        <div class="prose prose-sm dark:prose-invert max-w-none text-gray-700 dark:text-gray-300 whitespace-pre-line">
          {{ reach.description }}
        </div>
      </section>

      <!-- Gauge detail modal -->
      <GaugeDetailModal
        v-if="gaugeModalGauge"
        v-model:open="gaugeModalOpen"
        :gauge="gaugeModalGauge"
      />


      <!-- Features tabbed panel -->
      <section v-if="allFeatures.length > 0">
        <div class="border border-gray-200 dark:border-gray-700 rounded-xl overflow-hidden">

          <!-- Tab bar -->
          <div class="flex overflow-x-auto border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-950">
            <button
              v-for="tab in featureTabs"
              :key="tab.key"
              class="shrink-0 px-4 py-3 text-xs font-medium border-b-2 -mb-px transition-colors whitespace-nowrap flex items-center gap-1.5"
              :class="featuresTab === tab.key
                ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                : 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'"
              @click="featuresTab = tab.key"
            >
              {{ tab.label }}
              <span
                class="rounded-full px-1.5 py-px text-xs leading-none"
                :class="featuresTab === tab.key
                  ? 'bg-blue-100 dark:bg-blue-900 text-blue-600 dark:text-blue-400'
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-500'"
              >{{ tab.count }}</span>
            </button>
          </div>

          <!-- Feature rows — always expanded, no accordion -->
          <div
            ref="featureListRef"
            class="overflow-hidden"
          >
            <div v-if="filteredFeatures.length" class="divide-y divide-gray-100 dark:divide-gray-800">
              <div
                v-for="feat in filteredFeatures"
                :key="feat.key"
                class="px-4 py-3 flex items-start gap-3"
              >
                <!-- Icon circle (matches map pin symbols) -->
                <div
                  class="shrink-0 w-6 h-6 rounded-full flex items-center justify-center p-1 mt-0.5"
                  :style="{ background: featureIconColor(feat) }"
                  :title="featureTypeLabel(feat)"
                  v-html="featurePanelIcon(feat.type, { isHazard: feat.is_permanent_hazard })"
                />

                <!-- Content -->
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="text-sm font-medium text-gray-800 dark:text-gray-100">{{ feat.name }}</span>
                    <span
                      v-if="(feat.type === 'rapid' || feat.type === 'hazard') && feat.class_rating"
                      class="text-xs font-mono font-medium text-gray-500 dark:text-gray-400"
                    >{{ romanClass(feat.class_rating) }}<span v-if="feat.class_at_high && feat.class_at_high > feat.class_rating" class="text-gray-400">({{ romanClass(feat.class_at_high) }})</span></span>
                  </div>
                  <p v-if="feat.description" class="text-sm text-gray-600 dark:text-gray-400 mt-1 leading-relaxed">{{ feat.description }}</p>
                  <p v-if="feat.portage_description" class="text-xs text-amber-600 dark:text-amber-400 mt-0.5">
                    <span class="font-medium">Portage:</span> {{ feat.portage_description }}
                  </p>
                  <span
                    v-if="feat.is_permanent_hazard && feat.hazard_type"
                    class="inline-flex items-center rounded bg-red-50 dark:bg-red-950 px-1.5 py-0.5 text-xs font-medium text-red-700 dark:text-red-300 mt-0.5"
                  >⚠ {{ hazardTypeLabel(feat.hazard_type) }}</span>
                  <!-- Directions link for put-ins and take-outs -->
                  <a
                    v-if="(feat.type === 'put_in' || feat.type === 'take_out') && feat.lat != null && feat.lng != null"
                    :href="`https://www.google.com/maps/dir/?api=1&destination=${feat.lat},${feat.lng}`"
                    target="_blank"
                    rel="noopener"
                    class="inline-flex items-center gap-1 text-xs text-blue-500 hover:text-blue-600 dark:hover:text-blue-400 font-medium mt-1.5 transition-colors"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="3 11 22 2 13 21 11 13 3 11"/></svg>
                    Get directions
                  </a>
                </div>
              </div>
            </div>

            <div v-else class="px-4 py-8 text-center text-sm text-gray-400">
              No features in this category
            </div>
          </div>
        </div>
      </section>

      <!-- Share modal -->
      <ReachShareModal
        v-model:open="shareModalOpen"
        :reach-slug="(reach as any).slug"
        :reach-name="(reach as any).common_name ?? (reach as any).name"
        :current-cfs="(reach as any)?.gauge?.current_cfs ?? null"
        :flow-status="(reach as any)?.gauge?.flow_status ?? null"
      />

      <!-- Tributary / other related reaches -->
      <section v-if="tributaryReaches.length > 0">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Tributaries & Related</h2>
        <div class="flex flex-wrap gap-2">
          <NuxtLink
            v-for="rel in tributaryReaches"
            :key="rel.slug"
            :to="`/reaches/${rel.slug}`"
            class="flex items-center gap-2 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800 px-3 py-2 transition-colors"
          >
            <span class="text-xs text-gray-400">
              <template v-if="rel.relationship === 'tributary'">⤷</template>
              <template v-else>↔</template>
            </span>
            <span class="text-sm font-medium">{{ rel.name }}</span>
            <span class="text-xs text-gray-400 capitalize">{{ rel.relationship }}</span>
          </NuxtLink>
        </div>
      </section>


    </main>

    <!-- Scroll-to-top button -->
    <button
      ref="scrollTopBtn"
      class="fixed bottom-6 right-6 z-30 w-10 h-10 rounded-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 shadow-lg flex items-center justify-center text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors opacity-0 pointer-events-none"
      aria-label="Scroll to top"
      @click="scrollToTop"
    >
      <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M18 15l-6-6-6 6"/>
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useWatchlistStore } from '~/stores/watchlist'
import { gsap } from 'gsap'
import { featurePanelIcon } from '~/utils/featureIcons'

const route  = useRoute()
const config = useRuntimeConfig()
const store  = useWatchlistStore()
const { addAndSync, removeAndSync } = useWatchlistSync()
const { isAdmin, getToken } = useAuth()

// ---- Scroll-to-top ----------------------------------------------------------

const scrollTopBtn = ref<HTMLButtonElement>()
let scrollTopShown = false

function onScroll() {
  const show = window.scrollY > 80
  if (show === scrollTopShown) return
  scrollTopShown = show
  if (!scrollTopBtn.value) return
  if (show) {
    gsap.fromTo(scrollTopBtn.value,
      { opacity: 0, y: 12, scale: 0.9, pointerEvents: 'none' },
      { opacity: 1, y: 0, scale: 1, pointerEvents: 'auto', duration: 0.3, ease: 'back.out(1.4)' })
  } else {
    gsap.to(scrollTopBtn.value,
      { opacity: 0, y: 8, pointerEvents: 'none', duration: 0.2, ease: 'power2.in' })
  }
}

function scrollToTop() {
  gsap.to(window, { scrollTo: { y: 0 }, duration: 0.4, ease: 'power2.inOut' })
}

onMounted(() => {
  import('gsap/ScrollToPlugin').then(({ ScrollToPlugin }) => {
    gsap.registerPlugin(ScrollToPlugin)
  })
  window.addEventListener('scroll', onScroll, { passive: true })
})
onUnmounted(() => window.removeEventListener('scroll', onScroll))

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

// ---- River features (upstream→downstream timeline) --------------------------

interface RiverFeature {
  key:          string
  type:         'rapid' | 'put_in' | 'take_out' | 'hazard' | 'access' | 'camp' | 'parking'
  name:         string
  description?: string | null
  // rapids-specific
  class_rating?:          number | null
  class_at_high?:         number | null
  portage_description?:   string | null
  is_portage_recommended?: boolean
  is_permanent_hazard?:   boolean
  hazard_type?:           string | null
  // sorting — river_order is 0→1 along centerline (preferred); lng is fallback
  river_order?: number | null
  lng?:         number | null
  lat?:         number | null
}

// All features sorted upstream → downstream (including camps and parking).
const allFeatures = computed<RiverFeature[]>(() => {
  const r = reach.value as any
  if (!r) return []

  const items: RiverFeature[] = []

  for (const rap of r.rapids ?? []) {
    items.push({
      key:  `rapid-${rap.id}`,
      type: rap.is_permanent_hazard ? 'hazard' : 'rapid',
      name: stripRapidClass(rap.name),
      description: rap.description,
      class_rating: rap.class_rating,
      class_at_high: rap.class_at_high,
      portage_description: rap.portage_description,
      is_portage_recommended: rap.is_portage_recommended,
      is_permanent_hazard: rap.is_permanent_hazard,
      hazard_type: rap.hazard_type,
      river_order: rap.river_order,
      lng: rap.lng,
    })
  }

  for (const acc of r.access ?? []) {
    let type: RiverFeature['type'] = 'access'
    if (acc.access_type === 'put_in')   type = 'put_in'
    else if (acc.access_type === 'take_out') type = 'take_out'
    else if (acc.access_type === 'camp') type = 'camp'
    else if (acc.access_type === 'parking' || acc.access_type === 'shuttle_drop') type = 'parking'
    items.push({
      key:  `access-${acc.id}`,
      type,
      name: acc.name,
      description: acc.notes ?? acc.directions,
      river_order: acc.river_order,
      lng: acc.water_lng ?? acc.parking_lng,
      lat: acc.water_lat ?? acc.parking_lat,
    })
  }

  return items.sort((a, b) => {
    // put_in always floats to top; take_out always sinks to bottom.
    const typeRank = (t: string) => t === 'put_in' ? -1 : t === 'take_out' ? 1 : 0
    const ra = typeRank(a.type), rb = typeRank(b.type)
    if (ra !== rb) return ra - rb
    // Within the middle group, prefer centerline position (river_order 0→1).
    if (a.river_order != null && b.river_order != null) return a.river_order - b.river_order
    if (a.river_order != null) return -1
    if (b.river_order != null) return 1
    // Fall back to longitude when no centerline (Colorado rivers flow west→east)
    if (a.lng == null && b.lng == null) return 0
    if (a.lng == null) return 1
    if (b.lng == null) return -1
    return a.lng - b.lng
  })
})

// ---- Features tab state -------------------------------------------------------

const featuresTab = ref<string>('all')
const featureListRef = ref<HTMLElement | null>(null)

// Animate height when tab changes, then scroll up if container shrank past viewport bottom.
watch(featuresTab, async () => {
  const el = featureListRef.value
  if (!el) return
  const fromH = el.offsetHeight
  await nextTick()
  const toH = el.scrollHeight
  if (fromH === toH) return

  gsap.fromTo(
    el,
    { height: fromH },
    {
      height: toH,
      duration: 0.25,
      ease: 'power2.out',
      onComplete: () => {
        gsap.set(el, { clearProps: 'height' })
        // Scroll up if the container's new bottom is above the viewport bottom.
        const rect = el.getBoundingClientRect()
        const containerDocBottom = rect.top + window.scrollY + toH
        const viewportBottom = window.scrollY + window.innerHeight
        if (containerDocBottom < viewportBottom) {
          const targetY = Math.max(0, containerDocBottom - window.innerHeight + 24)
          if (targetY < window.scrollY) {
            window.scrollTo({ top: targetY, behavior: 'smooth' })
          }
        }
      },
    }
  )
})

const featureTabs = computed(() => {
  const f = allFeatures.value
  const tabs = [
    { key: 'all',     label: 'All',           count: f.length },
    { key: 'access',  label: 'Access Points', count: f.filter(x => ['put_in','take_out','access'].includes(x.type)).length },
    { key: 'rapids',  label: 'Rapids',        count: f.filter(x => x.type === 'rapid' || x.type === 'hazard').length },
    { key: 'camps',   label: 'Camps',         count: f.filter(x => x.type === 'camp').length },
    { key: 'parking', label: 'Parking',       count: f.filter(x => x.type === 'parking').length },
  ]
  // Only show specific tabs when they have entries
  return tabs.filter(t => t.key === 'all' || t.count > 0)
})

const filteredFeatures = computed(() => {
  switch (featuresTab.value) {
    case 'access':  return allFeatures.value.filter(f => ['put_in','take_out','access'].includes(f.type))
    case 'rapids':  return allFeatures.value.filter(f => f.type === 'rapid' || f.type === 'hazard')
    case 'camps':   return allFeatures.value.filter(f => f.type === 'camp')
    case 'parking': return allFeatures.value.filter(f => f.type === 'parking')
    default:        return allFeatures.value
  }
})

function featureTypeLabel(feat: RiverFeature): string {
  if (feat.is_permanent_hazard) return 'Hazard'
  switch (feat.type) {
    case 'rapid':    return 'Rapid'
    case 'put_in':   return 'Put-in'
    case 'take_out': return 'Take-out'
    case 'camp':     return 'Campsite'
    case 'parking':  return 'Parking'
    case 'access':   return 'Access'
    default:         return 'Feature'
  }
}

// Icon circle color — mirrors the pin colors used in ReachMap.vue
function featureIconColor(feat: RiverFeature): string {
  if (feat.is_permanent_hazard) return '#ef4444'
  switch (feat.type) {
    case 'rapid':    return '#3b82f6'
    case 'put_in':   return '#22c55e'
    case 'take_out': return '#ef4444'
    case 'camp':     return '#f59e0b'
    case 'parking':  return '#dc2626'
    case 'access':   return '#94a3b8'
    default:         return '#94a3b8'
  }
}


function featurePillClass(feat: RiverFeature): string {
  if (feat.is_permanent_hazard)
    return 'bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300'
  switch (feat.type) {
    case 'rapid':
    case 'hazard':
      return 'bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-300'
    case 'put_in':
    case 'take_out':
    case 'access':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
    case 'camp':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
    case 'parking':
      return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-300'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-300'
  }
}

function featurePinBg(feat: RiverFeature): string {
  if (feat.is_permanent_hazard) return 'bg-red-100 dark:bg-red-950'
  switch (feat.type) {
    case 'rapid':   return 'bg-blue-100 dark:bg-blue-950'
    case 'hazard':  return 'bg-red-100 dark:bg-red-950'
    case 'put_in':
    case 'take_out':
    case 'access':  return 'bg-emerald-100 dark:bg-emerald-950'
    case 'camp':    return 'bg-amber-100 dark:bg-amber-950'
    case 'parking': return 'bg-gray-100 dark:bg-gray-800'
    default:        return 'bg-gray-100 dark:bg-gray-800'
  }
}

function featurePinIcon(feat: RiverFeature): string {
  if (feat.is_permanent_hazard) return 'text-red-500'
  switch (feat.type) {
    case 'rapid':   return 'text-blue-500'
    case 'hazard':  return 'text-red-500'
    case 'put_in':
    case 'take_out':
    case 'access':  return 'text-emerald-500'
    case 'camp':    return 'text-amber-500'
    case 'parking': return 'text-gray-400'
    default:        return 'text-gray-400'
  }
}

function hazardTypeLabel(type: string): string {
  const map: Record<string, string> = {
    low_head_dam:  'Low-head dam',
    dam:           'Dam',
    rebar:         'Rebar / concrete',
    strainer:      'Strainer',
    bridge_piling: 'Bridge piling',
    other:         'Permanent hazard',
  }
  return map[type] ?? type
}

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

// ---- Related reach navigation -----------------------------------------------

const upstreamReach = computed(() =>
  (reach.value as any)?.related?.find((r: any) => r.relationship === 'upstream') ?? null
)
const downstreamReach = computed(() =>
  (reach.value as any)?.related?.find((r: any) => r.relationship === 'downstream') ?? null
)
const tributaryReaches = computed(() =>
  ((reach.value as any)?.related ?? []).filter(
    (r: any) => r.relationship !== 'upstream' && r.relationship !== 'downstream'
  )
)

const classLabel = computed(() => {
  const r = reach.value as any
  if (!r?.class_min && !r?.class_max) return 'Unknown class'
  const base = r.class_min === r.class_max
    ? `Class ${romanClass(r.class_min!)}`
    : `Class ${romanClass(r.class_min!)}–${romanClass(r.class_max!)}`
  if (r.class_hardest != null && r.class_hardest > (r.class_max ?? 0))
    return `${base} (${romanClass(r.class_hardest)})`
  return base
})

const difficultyColor = computed(() => {
  const c = (reach.value as any)?.class_max
  if (c == null) return '#6b7280'
  if (c < 3.0) return '#16a34a'
  if (c < 4.0) return '#3b82f6'
  if (c < 5.0) return '#1f2937'
  return '#dc2626'
})

// Dark-mode-safe text classes for difficulty — mirrors the badge color intent
// but uses Tailwind responsive classes so near-black stays readable on dark bg.
const difficultyTextClass = computed(() => {
  const c = (reach.value as any)?.class_max
  if (c == null) return 'text-gray-500 dark:text-gray-400'
  if (c < 3.0)  return 'text-green-600 dark:text-green-400'
  if (c < 4.0)  return 'text-blue-500 dark:text-blue-400'
  if (c < 5.0)  return 'text-gray-700 dark:text-gray-200'   // near-black → visible in both modes
  return 'text-red-600 dark:text-red-400'
})

const statusColor = computed(() => {
  switch (reach.value?.gauge.flow_status) {
    case 'runnable': return 'success'
    case 'caution':  return 'error'
    case 'low':      return 'error'
    case 'flood':    return 'info'
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
  'text-red-500':     ['caution','low'].includes(reach.value?.gauge.flow_status ?? ''),
  'text-sky-500':     reach.value?.gauge.flow_status === 'flood',
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

// ---- Multi-gauge helpers ----------------------------------------------------

// All gauges for this reach: the API returns a `gauges` array with primary first,
// then any secondary gauges linked via reach_id.
const allGauges = computed<any[]>(() => {
  const r = reach.value as any
  if (!r) return []
  // Prefer the flat `gauges` array (new field). Fall back to wrapping the primary gauge.
  if (Array.isArray(r.gauges) && r.gauges.length > 0) return r.gauges
  if (r.gauge?.id) return [r.gauge]
  return []
})

function gaugeRelLabel(rel: string | null | undefined): string {
  switch (rel) {
    case 'upstream_indicator':   return 'Upstream gauge'
    case 'downstream_indicator': return 'Downstream gauge'
    case 'tributary':            return 'Tributary gauge'
    case 'primary':
    default:                     return 'Flow gauge'
  }
}

const confirmingRemove = ref<string | null>(null)

function onDashboard(gaugeId: string): boolean {
  const reachSlug = (reach.value as any)?.slug ?? null
  return store.gauges.some(g => g.id === gaugeId && (g.contextReachSlug ?? null) === reachSlug)
}

function confirmRemoveDashboard(gaugeId: string) {
  removeFromDashboard(gaugeId)
  confirmingRemove.value = null
}

function addToDashboard(g: any) {
  const r = reach.value as any
  const putIn   = r?.put_in_name  ?? null
  const takeOut = r?.take_out_name ?? null
  addAndSync({
    id:                     g.id,
    externalId:             g.external_id,
    source:                 g.source ?? '',
    name:                   g.name ?? null,
    contextReachSlug:       r?.slug ?? null,
    contextReachCommonName: r?.common_name ?? r?.name ?? null,
    contextReachFullName:   putIn && takeOut ? `${putIn} to ${takeOut}` : null,
    contextReachRiverName:  r?.river_name ?? null,
    reachId:                r?.id ?? null,
    reachName:              r?.common_name ?? r?.name ?? null,
    reachNames:             [],
    reachSlug:              r?.slug ?? null,
    reachSlugs:             [],
    reachCommonNames:       [],
    reachRelationship:      g.reach_relationship ?? 'primary',
    watershedName:          r?.watershed_name ?? null,
    basinName:              null,
    riverName:              r?.river_name ?? null,
    stateAbbr:              null,
    lat:                    g.lat ?? null,
    lng:                    g.lng ?? null,
    currentCfs:             g.current_cfs ?? null,
    flowStatus:             g.flow_status ?? 'unknown',
    flowBandLabel:          null,
    lastReadingAt:          g.last_reading_at ?? null,
  })
}

function removeFromDashboard(gaugeId: string) {
  const reachSlug = (reach.value as any)?.slug ?? null
  removeAndSync(gaugeId, reachSlug)
}

// ---- Gauge flow modal -------------------------------------------------------

import type { WatchedGauge } from '~/stores/watchlist'
const gaugeModalOpen  = ref(false)
const gaugeModalGauge = ref<WatchedGauge | null>(null)

function openGaugeModal(g: any) {
  const r = reach.value as any
  const putIn   = r?.put_in_name  ?? null
  const takeOut = r?.take_out_name ?? null
  gaugeModalGauge.value = {
    id:                     g.id,
    externalId:             g.external_id,
    source:                 g.source ?? '',
    name:                   g.name ?? null,
    contextReachSlug:       r?.slug ?? null,
    contextReachCommonName: r?.common_name ?? r?.name ?? null,
    contextReachFullName:   putIn && takeOut ? `${putIn} to ${takeOut}` : null,
    contextReachRiverName:  r?.river_name ?? null,
    reachId:                r?.id ?? null,
    reachName:              r?.common_name ?? r?.name ?? null,
    reachNames:             [],
    reachSlug:              r?.slug ?? null,
    reachSlugs:             [],
    reachCommonNames:       [],
    reachRelationship:      g.reach_relationship ?? 'primary',
    watershedName:          r?.watershed_name ?? null,
    basinName:              null,
    riverName:              r?.river_name ?? null,
    stateAbbr:              null,
    lat:                    g.lat ?? null,
    lng:                    g.lng ?? null,
    currentCfs:             g.current_cfs ?? null,
    flowStatus:             g.flow_status ?? 'unknown',
    flowBandLabel:          null,
    lastReadingAt:          g.last_reading_at ?? null,
    contextReachBasinGroup: null,
  }
  gaugeModalOpen.value = true
}

// ---- Rapid name helpers ----------------------------------------------------

/** Strip trailing class notation from rapid names: "Gorilla (Class V)" → "Gorilla" */
function stripRapidClass(name: string | null): string | null {
  if (!name) return name
  return name.replace(/\s*\((?:class\s+)?[IVX]+[+]?\)\s*$/i, '').trim() || name
}

function flowBadgeClass(status: string, band?: string | null): string {
  if (band === 'below_recommended') return 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400'
  if (band === 'low_runnable')      return 'bg-lime-100 dark:bg-lime-950/50 text-lime-700 dark:text-lime-400'
  if (band === 'runnable')          return 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400'
  if (band === 'med_runnable')      return 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400'
  if (band === 'high_runnable')     return 'bg-green-100 dark:bg-green-950/50 text-green-700 dark:text-green-500'
  if (band === 'above_recommended') return 'bg-blue-100 dark:bg-blue-950/50 text-blue-700 dark:text-blue-400'
  switch (status) {
    case 'runnable': return 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400'
    case 'caution':  return 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400'
    case 'low':      return 'bg-red-100 dark:bg-red-950/50 text-red-600 dark:text-red-400'
    case 'flood':    return 'bg-blue-100 dark:bg-blue-950/50 text-blue-700 dark:text-blue-400'
    default:         return 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
  }
}

function flowBandLabel(status: string, band?: string | null): string {
  if (band) {
    return band.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
  }
  switch (status) {
    case 'runnable': return 'Runnable'
    case 'caution':  return 'Below Recommended'
    case 'low':      return 'Too Low'
    case 'flood':    return 'Above Recommended'
    default:         return 'Unknown'
  }
}

function cfsColorClass(status: string, band?: string | null): string {
  if (band === 'low_runnable')  return 'text-lime-500'
  if (band === 'med_runnable')  return 'text-emerald-500'
  if (band === 'high_runnable') return 'text-green-600 dark:text-green-500'
  if (band === 'runnable')      return 'text-emerald-500'
  if (band === 'below_recommended') return 'text-red-500'
  if (band === 'above_recommended') return 'text-blue-500'
  switch (status) {
    case 'runnable': return 'text-emerald-500'
    case 'caution':  return 'text-red-500'
    case 'low':      return 'text-red-500'
    case 'flood':    return 'text-blue-500'
    default:         return 'text-gray-700 dark:text-gray-200'
  }
}

function relativeTime(t: string | null): string {
  if (!t) return ''
  const ms = Date.now() - new Date(t).getTime()
  const m = Math.floor(ms / 60_000)
  if (m < 1)  return 'just now'
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ${m % 60}m ago`
}

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
    const res = await fetch(url, { method: 'POST', headers: { Authorization: `Bearer ${await getToken()}` } })
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

const chatMessages  = ref<{ role: 'user' | 'assistant'; content: string }[]>([])
const chatInput     = ref('')
const chatLoading   = ref(false)
const chatError     = ref<string | null>(null)

const lastAssistantMessage = computed(() => {
  const msgs = chatMessages.value.filter(m => m.role === 'assistant')
  return msgs.length ? msgs[msgs.length - 1].content : null
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

// ---- Share modal ------------------------------------------------------------

const shareModalOpen = ref(false)

function openShareForm() {
  shareModalOpen.value = true
}

// ---- Delete reach -----------------------------------------------------------

const deleting = ref(false)

async function deleteReach() {
  if (!confirm(`Permanently delete "${(reach.value as any)?.common_name ?? (reach.value as any)?.name}"?\n\nThis removes all rapids, access points, and features. Gauges are unlinked but kept.`)) return
  deleting.value = true
  try {
    const res = await fetch(`${config.public.apiBase}/api/v1/reaches/${route.params.slug}`, { method: 'DELETE', headers: { Authorization: `Bearer ${await getToken()}` } })
    if (!res.ok) throw new Error(`Server error ${res.status}`)
    navigateTo('/')
  } catch (err: any) {
    alert(err?.message ?? 'Delete failed')
  } finally {
    deleting.value = false
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
      headers: { Authorization: `Bearer ${await getToken()}` },
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
