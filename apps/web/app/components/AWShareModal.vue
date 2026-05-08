<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="open"
        class="fixed inset-0 z-[60] flex items-end sm:items-center justify-center p-4"
        @click.self="$emit('close')"
      >
        <div class="absolute inset-0 bg-black/50" @click="$emit('close')" />

        <div class="relative w-full max-w-sm bg-white dark:bg-neutral-900 rounded-2xl border border-neutral-200 dark:border-neutral-700 shadow-xl overflow-hidden">
          <!-- Header -->
          <div class="flex items-center justify-between px-5 py-4 border-b border-neutral-100 dark:border-neutral-800">
            <h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Share to American Whitewater</h2>
            <button class="p-1 rounded-md text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200" @click="$emit('close')">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>

          <!-- Band mapping setup (first time) -->
          <div v-if="!mappingReady" class="px-5 py-4 space-y-4">
            <p class="text-sm text-neutral-600 dark:text-neutral-400">
              AW uses 5 flow levels. Map your H2OFlows flow bands to AW levels once — we'll remember it.
            </p>

            <div class="space-y-3">
              <div v-for="band in ['low', 'running', 'high'] as const" :key="band" class="flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                  <span class="w-2 h-2 rounded-full shrink-0" :class="bandDotClass(band)" />
                  <span class="text-sm font-medium text-neutral-700 dark:text-neutral-200 capitalize">{{ band }}</span>
                </div>
                <select
                  v-model="mapping[band]"
                  class="text-xs rounded-md border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-700 dark:text-neutral-200 px-2 py-1"
                >
                  <option v-for="opt in awOptions[band]" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
            </div>

            <button
              class="w-full py-2 rounded-lg bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium transition-colors"
              :disabled="savingMapping"
              @click="saveMapping"
            >
              {{ savingMapping ? 'Saving…' : 'Save & continue' }}
            </button>
          </div>

          <!-- Cross-post panel -->
          <div v-else class="px-5 py-4 space-y-4">
            <!-- Formatted text preview -->
            <div class="rounded-lg bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 p-3">
              <p class="text-xs text-neutral-500 dark:text-neutral-400 mb-2 font-medium uppercase tracking-wide">Copy to AW journal</p>
              <p class="text-sm text-neutral-700 dark:text-neutral-200 leading-relaxed whitespace-pre-wrap">{{ awText }}</p>
            </div>

            <button
              class="w-full flex items-center justify-center gap-2 py-2 rounded-lg border border-neutral-200 dark:border-neutral-700 text-sm text-neutral-700 dark:text-neutral-200 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
              @click="copyAWText"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
              {{ textCopied ? 'Copied!' : 'Copy text' }}
            </button>

            <a
              href="https://www.americanwhitewater.org/content/Journal/"
              target="_blank"
              rel="noopener"
              class="w-full flex items-center justify-center gap-2 py-2 rounded-lg bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium transition-colors"
            >
              Open AW journal
              <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
            </a>

            <div class="border-t border-neutral-100 dark:border-neutral-800" />

            <button
              v-if="!synced"
              class="w-full py-2 rounded-lg border border-emerald-200 dark:border-emerald-800 text-emerald-700 dark:text-emerald-400 text-sm font-medium hover:bg-emerald-50 dark:hover:bg-emerald-950/30 transition-colors"
              :disabled="syncing"
              @click="markPosted"
            >
              {{ syncing ? 'Marking…' : 'I posted it ✓' }}
            </button>
            <div v-else class="text-center text-sm text-emerald-600 dark:text-emerald-400 font-medium">
              Marked as posted on AW
            </div>

            <button
              class="w-full text-xs text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 transition-colors"
              @click="mappingReady = false"
            >
              Change flow band mapping
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
interface Report {
  id: string
  slug: string
  name: string
  report_date: string
  report_time?: string
  content: string
  hazard_warning?: string
  paddled: boolean
  flow_cfs?: number
  flow_band?: string
  aw_synced_at?: string
  reach_name: string
  reach_slug: string
}

const props = defineProps<{ report: Report; open: boolean }>()
const emit = defineEmits<{ close: []; synced: [] }>()

const config = useRuntimeConfig()

const awOptions = {
  low:     [{ value: 'too-low', label: 'Too low' },  { value: 'low',    label: 'Low'    }],
  running: [{ value: 'low',    label: 'Low'    },    { value: 'medium', label: 'Medium' }, { value: 'high', label: 'High' }],
  high:    [{ value: 'high',   label: 'High'   },    { value: 'too-high', label: 'Too high' }],
}

const mapping = reactive<Record<string, string>>({
  low:     'too-low',
  running: 'medium',
  high:    'too-high',
})
const mappingReady = ref(false)
const savingMapping = ref(false)
const textCopied = ref(false)
const syncing = ref(false)
const synced = ref(false)

onMounted(async () => {
  try {
    const prefs = await $fetch<{ aw_band_mapping: Record<string, string> | null }>(
      `${config.public.apiBase}/api/v1/me/preferences`
    )
    if (prefs.aw_band_mapping) {
      Object.assign(mapping, prefs.aw_band_mapping)
      mappingReady.value = true
    }
  } catch {
    // no prefs yet
  }
  synced.value = !!props.report.aw_synced_at
})

async function saveMapping() {
  savingMapping.value = true
  try {
    await $fetch(`${config.public.apiBase}/api/v1/me/preferences`, {
      method: 'PATCH',
      body: { aw_band_mapping: mapping },
    })
    mappingReady.value = true
  } finally {
    savingMapping.value = false
  }
}

const awBandLabel = computed(() => {
  if (!props.report.flow_band) return ''
  return mapping[props.report.flow_band] ?? props.report.flow_band
})

const awText = computed(() => {
  const lines: string[] = []
  lines.push(props.report.name)
  lines.push(`${props.report.reach_name} — ${props.report.report_date}`)
  if (props.report.flow_cfs != null) {
    lines.push(`Flow: ${Math.round(props.report.flow_cfs).toLocaleString()} cfs (${awBandLabel.value})`)
  }
  if (props.report.paddled) lines.push('Paddled this reach')
  lines.push('')
  lines.push(props.report.content)
  if (props.report.hazard_warning) {
    lines.push('')
    lines.push(`⚠ Hazard: ${props.report.hazard_warning}`)
  }
  return lines.join('\n')
})

async function copyAWText() {
  await navigator.clipboard.writeText(awText.value)
  textCopied.value = true
  setTimeout(() => { textCopied.value = false }, 2000)
}

async function markPosted() {
  syncing.value = true
  try {
    await $fetch(`${config.public.apiBase}/api/v1/me/reports/${props.report.slug}/aw-sync`, {
      method: 'POST',
    })
    synced.value = true
    emit('synced')
  } finally {
    syncing.value = false
  }
}

function bandDotClass(band: string): string {
  if (band === 'low') return 'bg-sky-500'
  if (band === 'running') return 'bg-emerald-500'
  return 'bg-amber-500'
}
</script>

<style scoped>
.modal-enter-active, .modal-leave-active { transition: opacity 0.15s; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
</style>
