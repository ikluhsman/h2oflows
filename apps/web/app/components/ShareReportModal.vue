<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4"
        @click.self="$emit('close')"
      >
        <div class="absolute inset-0 bg-black/40" @click="$emit('close')" />

        <div class="relative w-full max-w-sm bg-white dark:bg-neutral-900 rounded-2xl border border-neutral-200 dark:border-neutral-700 shadow-xl overflow-hidden">
          <!-- Header -->
          <div class="flex items-center justify-between px-5 py-4 border-b border-neutral-100 dark:border-neutral-800">
            <h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Share report</h2>
            <button class="p-1 rounded-md text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200" @click="$emit('close')">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>

          <!-- Share actions -->
          <div class="px-5 py-4 space-y-2">
            <!-- Copy link -->
            <button
              class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors text-left"
              @click="copyLink"
            >
              <span class="w-8 h-8 rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center shrink-0">
                <svg class="w-4 h-4 text-neutral-600 dark:text-neutral-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
              </span>
              <span class="text-sm text-neutral-700 dark:text-neutral-200">{{ linkCopied ? 'Copied!' : 'Copy link' }}</span>
            </button>

            <!-- Twitter / X -->
            <a
              :href="twitterUrl"
              target="_blank"
              rel="noopener"
              class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
            >
              <span class="w-8 h-8 rounded-full bg-black flex items-center justify-center shrink-0">
                <svg class="w-4 h-4 text-white" viewBox="0 0 24 24" fill="currentColor"><path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-4.714-6.231-5.401 6.231H2.74l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/></svg>
              </span>
              <span class="text-sm text-neutral-700 dark:text-neutral-200">Share on X</span>
            </a>

            <!-- SMS -->
            <a
              :href="smsUrl"
              class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
            >
              <span class="w-8 h-8 rounded-full bg-emerald-500 flex items-center justify-center shrink-0">
                <svg class="w-4 h-4 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
              </span>
              <span class="text-sm text-neutral-700 dark:text-neutral-200">Send via SMS</span>
            </a>

            <!-- Copy as message (for Discord etc.) -->
            <button
              class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors text-left"
              @click="copyMessage"
            >
              <span class="w-8 h-8 rounded-full bg-indigo-500 flex items-center justify-center shrink-0">
                <svg class="w-4 h-4 text-white" viewBox="0 0 24 24" fill="currentColor"><path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057c.003.022.01.043.02.062a19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03z"/></svg>
              </span>
              <span class="text-sm text-neutral-700 dark:text-neutral-200">{{ messageCopied ? 'Copied!' : 'Copy for Discord' }}</span>
            </button>

            <div class="border-t border-neutral-100 dark:border-neutral-800 my-1" />

            <!-- AW cross-post -->
            <button
              class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors text-left"
              @click="openAWShare"
            >
              <span class="w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900/50 flex items-center justify-center shrink-0">
                <svg class="w-4 h-4 text-primary-600 dark:text-primary-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9"/><path d="M12 22V5"/></svg>
              </span>
              <div>
                <div class="text-sm text-neutral-700 dark:text-neutral-200">Share to American Whitewater</div>
                <div v-if="report.aw_synced_at" class="text-xs text-neutral-400 dark:text-neutral-500">Posted {{ formatDate(report.aw_synced_at) }}</div>
              </div>
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <AWShareModal
      v-if="awOpen"
      :report="report"
      :open="awOpen"
      @close="awOpen = false"
      @synced="onAWSynced"
    />
  </Teleport>
</template>

<script setup lang="ts">
interface Report {
  id: string
  slug: string
  handle: string
  name: string
  report_date: string
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
const linkCopied = ref(false)
const messageCopied = ref(false)
const awOpen = ref(false)

const reportUrl = computed(() =>
  `${config.public.appUrl ?? 'https://h2oflow.org'}/reports/${props.report.id}`
)

const shareText = computed(() => {
  const flow = props.report.flow_cfs != null
    ? ` @ ${Math.round(props.report.flow_cfs).toLocaleString()} cfs`
    : ''
  return `${props.report.name} — ${props.report.reach_name}${flow}`
})

const twitterUrl = computed(() =>
  `https://twitter.com/intent/tweet?text=${encodeURIComponent(shareText.value)}&url=${encodeURIComponent(reportUrl.value)}`
)

const smsUrl = computed(() =>
  `sms:?body=${encodeURIComponent(`${shareText.value} ${reportUrl.value}`)}`
)

async function copyLink() {
  await navigator.clipboard.writeText(reportUrl.value)
  linkCopied.value = true
  setTimeout(() => { linkCopied.value = false }, 2000)
}

async function copyMessage() {
  const text = `${shareText.value}\n${reportUrl.value}`
  await navigator.clipboard.writeText(text)
  messageCopied.value = true
  setTimeout(() => { messageCopied.value = false }, 2000)
}

function openAWShare() {
  awOpen.value = true
}

function onAWSynced() {
  awOpen.value = false
  emit('synced')
}

function formatDate(d: string): string {
  return new Date(d).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}
</script>

<style scoped>
.modal-enter-active, .modal-leave-active { transition: opacity 0.15s; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
</style>
