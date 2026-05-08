<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-neutral-950">
    <AppHeader>
      <template v-if="report">
        <span class="text-neutral-300 dark:text-neutral-700 shrink-0">/</span>
        <span class="text-sm font-medium truncate text-neutral-700 dark:text-neutral-200">{{ report.name }}</span>
      </template>
    </AppHeader>

    <div v-if="pending" class="max-w-2xl mx-auto px-4 py-20 flex justify-center">
      <div class="w-6 h-6 rounded-full border-2 border-primary-500 border-t-transparent animate-spin" />
    </div>

    <div v-else-if="!report" class="max-w-2xl mx-auto px-4 py-20 text-center text-neutral-400">
      Report not found.
    </div>

    <main v-else class="max-w-2xl mx-auto px-4 py-8 pb-20 sm:pb-8 space-y-6">

      <!-- Hazard callout -->
      <div
        v-if="report.hazard_warning"
        class="flex items-start gap-3 rounded-xl border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 px-4 py-3"
      >
        <svg class="w-5 h-5 mt-0.5 shrink-0 text-red-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
        <div>
          <p class="text-sm font-semibold text-red-700 dark:text-red-400 mb-0.5">Hazard warning</p>
          <p class="text-sm text-red-700 dark:text-red-400">{{ report.hazard_warning }}</p>
        </div>
      </div>

      <!-- Header card -->
      <div class="bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-700 px-5 py-4 space-y-3">
        <div class="flex items-start justify-between gap-3 flex-wrap">
          <div>
            <h1 class="text-lg font-bold text-neutral-900 dark:text-white">{{ report.name }}</h1>
            <p v-if="report.handle" class="text-xs text-neutral-400 mt-0.5">@{{ report.handle }}</p>
          </div>
          <div class="text-right shrink-0">
            <div class="text-sm font-medium text-neutral-700 dark:text-neutral-300">{{ formatDate(report.report_date) }}</div>
            <div v-if="report.report_time" class="text-xs text-neutral-400">{{ report.report_time }}</div>
          </div>
        </div>

        <!-- Reach link -->
        <NuxtLink
          :to="`/reaches/${report.reach_slug}`"
          class="inline-flex items-center gap-1.5 text-sm text-primary-600 dark:text-primary-400 hover:underline"
        >
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9"/></svg>
          {{ report.reach_name }}
        </NuxtLink>

        <!-- Flow / paddled badges -->
        <div class="flex items-center gap-2 flex-wrap">
          <span
            v-if="report.flow_cfs != null"
            class="inline-flex items-center gap-1 rounded-md bg-neutral-100 dark:bg-neutral-800 px-2 py-0.5 text-xs font-medium text-neutral-600 dark:text-neutral-300"
          >
            {{ Math.round(report.flow_cfs).toLocaleString() }} cfs
            <span v-if="report.flow_band" :class="bandBadgeClass(report.flow_band)" class="font-medium capitalize">{{ report.flow_band }}</span>
          </span>
          <span
            v-if="report.paddled"
            class="inline-flex items-center gap-1 rounded-md bg-primary-50 dark:bg-primary-950/50 px-2 py-0.5 text-xs font-medium text-primary-600 dark:text-primary-400"
          >
            <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 12c2-4 4-6 6-6s4 6 6 6 4-6 6-6"/></svg>
            Paddled this reach
          </span>
        </div>
      </div>

      <!-- Content -->
      <div class="bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-700 px-5 py-4">
        <div class="report-content text-sm text-neutral-700 dark:text-neutral-300 leading-relaxed" v-html="renderedContent" />
      </div>

    </main>
  </div>
</template>

<script setup lang="ts">
import MarkdownIt from 'markdown-it'

const route = useRoute()
const config = useRuntimeConfig()

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

interface Report {
  id: string
  slug: string
  handle: string
  name: string
  report_date: string
  report_time?: string
  content: string
  hazard_warning?: string
  paddled: boolean
  flow_cfs?: number
  flow_band?: string
  created_at: string
  reach_name: string
  reach_slug: string
}

const { data: report, pending } = await useAsyncData<Report | null>(
  `report-${route.params.id}`,
  () => $fetch<Report>(`${config.public.apiBase}/api/v1/reports/${route.params.id}`)
    .catch(() => null)
)

const renderedContent = computed(() =>
  report.value ? md.render(report.value.content || '') : ''
)

function formatDate(d: string): string {
  const [y, m, day] = d.split('-').map(Number)
  return new Date(y, m - 1, day).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function bandBadgeClass(band: string): string {
  if (band === 'low') return 'text-sky-600 dark:text-sky-400'
  if (band === 'running') return 'text-emerald-600 dark:text-emerald-400'
  if (band === 'high') return 'text-amber-600 dark:text-amber-400'
  return 'text-neutral-500'
}
</script>

<style>
.report-content > * + * { margin-top: 0.5em; }
.report-content strong { font-weight: 600; }
.report-content em { font-style: italic; }
.report-content s { text-decoration: line-through; }
.report-content ul { list-style-type: disc; padding-left: 1.4em; }
.report-content ol { list-style-type: decimal; padding-left: 1.4em; }
.report-content li + li { margin-top: 0.2em; }
.report-content blockquote {
  border-left: 3px solid #d1d5db;
  padding-left: 0.75em;
  color: #6b7280;
  font-style: italic;
}
.dark .report-content blockquote { border-left-color: #4b5563; color: #9ca3af; }
.report-content code {
  background: #f3f4f6;
  border-radius: 3px;
  padding: 0.1em 0.3em;
  font-size: 0.85em;
  font-family: ui-monospace, monospace;
}
.dark .report-content code { background: #1f2937; }
.report-content pre {
  background: #f3f4f6;
  border-radius: 6px;
  padding: 0.75em 1em;
  overflow-x: auto;
}
.dark .report-content pre { background: #111827; }
.report-content pre code { background: transparent; padding: 0; }
.report-content a { color: #2563eb; text-decoration: underline; }
.dark .report-content a { color: #60a5fa; }
</style>
