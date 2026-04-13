<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader>
      <template v-if="report">
        <span class="text-gray-300 dark:text-gray-700 shrink-0">/</span>
        <span class="text-sm font-medium truncate text-gray-700 dark:text-gray-200">Trip report</span>
      </template>
    </AppHeader>

    <main class="max-w-3xl mx-auto px-4 sm:px-6 py-8 space-y-6">

      <!-- Loading -->
      <div v-if="pending" class="flex items-center justify-center h-60 text-gray-400 text-sm">Loading…</div>

      <!-- Not found -->
      <div v-else-if="!report" class="flex flex-col items-center justify-center h-60 gap-3 text-center">
        <p class="text-gray-500 text-sm">This trip report wasn't found or isn't published yet.</p>
        <NuxtLink to="/" class="text-blue-600 hover:underline text-sm">Back to h2oflows</NuxtLink>
      </div>

      <template v-else>

        <!-- Reach link -->
        <NuxtLink
          :to="`/reaches/${report.reach_slug}`"
          class="inline-flex items-center gap-1.5 text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 transition-colors font-medium"
        >
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 5l-7 7 7 7"/></svg>
          {{ report.reach_name }}
        </NuxtLink>

        <!-- Title + meta -->
        <div class="space-y-1">
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ report.title ?? 'Trip Report' }}
          </h1>
          <div class="flex items-center flex-wrap gap-3 text-sm text-gray-500 dark:text-gray-400">
            <span>{{ observedLabel }}</span>
            <span v-if="report.cfs_at_time != null" :class="cfsClass">
              {{ report.cfs_at_time.toLocaleString() }} cfs
            </span>
            <span
              v-if="report.flow_impression"
              class="px-2 py-0.5 rounded-full text-xs font-medium"
              :class="impressionClass"
            >{{ impressionLabel }}</span>
          </div>
        </div>

        <!-- Body -->
        <div
          v-if="report.body"
          class="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 px-5 py-4 text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap"
        >{{ report.body }}</div>
        <div
          v-else
          class="bg-white dark:bg-gray-900 rounded-xl border border-gray-100 dark:border-gray-800 px-5 py-4 text-sm text-gray-400 italic"
        >No description provided.</div>

        <!-- Photos placeholder -->
        <div class="bg-gray-100 dark:bg-gray-800/50 rounded-xl border border-dashed border-gray-200 dark:border-gray-700 px-5 py-6 text-center text-sm text-gray-400">
          Photo uploads coming soon
        </div>

        <!-- CTA -->
        <div class="border-t border-gray-100 dark:border-gray-800 pt-4 flex flex-col sm:flex-row items-start sm:items-center gap-3 justify-between">
          <p class="text-xs text-gray-400">Know this reach? Share conditions with the community.</p>
          <NuxtLink
            :to="`/reaches/${report.reach_slug}`"
            class="shrink-0 inline-flex items-center gap-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white px-3 py-1.5 text-xs font-semibold transition-colors"
          >
            View {{ report.reach_name }}
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
          </NuxtLink>
        </div>

      </template>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const route  = useRoute()
const config = useRuntimeConfig()

const { data: report, pending } = await useAsyncData(
  `trip-report-${route.params.slug}`,
  () => $fetch<any>(`${config.public.apiBase}/api/v1/trip-reports/${route.params.slug}`)
    .catch(() => null)
)

// ---- Display helpers -------------------------------------------------------

const observedLabel = computed(() => {
  if (!report.value?.observed_at) return ''
  const d = new Date(report.value.observed_at)
  return d.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
})

const cfsClass = computed(() => {
  // No flow status on this response, so just style neutrally
  return 'font-medium text-sky-600 dark:text-sky-400'
})

const impressionLabel = computed(() => {
  switch (report.value?.flow_impression) {
    case 'too_low': return 'Too Low'
    case 'good':    return 'Good'
    case 'high':    return 'High'
    default:        return ''
  }
})

const impressionClass = computed(() => {
  switch (report.value?.flow_impression) {
    case 'too_low': return 'bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300'
    case 'good':    return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
    case 'high':    return 'bg-sky-100 text-sky-700 dark:bg-sky-950 dark:text-sky-300'
    default:        return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
  }
})

// ---- SEO -------------------------------------------------------------------

const metaTitle = computed(() =>
  report.value
    ? `${report.value.title ?? 'Trip Report'} — ${report.value.reach_name} | h2oflows`
    : 'Trip Report | h2oflows'
)

const metaDesc = computed(() => {
  if (!report.value) return ''
  const parts = [
    report.value.reach_name,
    observedLabel.value,
    report.value.cfs_at_time != null ? `${report.value.cfs_at_time.toLocaleString()} cfs` : null,
    impressionLabel.value || null,
  ].filter(Boolean)
  return parts.join(' · ')
})

useSeoMeta({
  title:           () => metaTitle.value,
  ogTitle:         () => metaTitle.value,
  description:     () => metaDesc.value,
  ogDescription:   () => metaDesc.value,
  ogType:          'article',
  twitterCard:     'summary',
})
</script>
