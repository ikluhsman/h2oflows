<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-neutral-950">
    <AppHeader>
      <span class="text-neutral-300 dark:text-neutral-700 shrink-0">/</span>
      <span class="text-sm font-medium text-neutral-700 dark:text-neutral-200">New Report</span>
    </AppHeader>

    <div v-if="!authReady" class="max-w-2xl mx-auto px-4 py-20 flex justify-center">
      <div class="w-6 h-6 rounded-full border-2 border-primary-500 border-t-transparent animate-spin" />
    </div>

    <div v-else-if="!isAuthenticated" class="max-w-2xl mx-auto px-4 py-20 flex flex-col items-center gap-3 text-center">
      <svg class="w-10 h-10 text-neutral-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
      </svg>
      <h2 class="text-lg font-semibold">Sign in to submit a report</h2>
      <NuxtLink to="/login" class="text-sm text-primary-600 dark:text-primary-400 hover:underline">Sign in</NuxtLink>
    </div>

    <main v-else class="max-w-2xl mx-auto px-4 py-8 pb-24 sm:pb-8 space-y-6">
      <h1 class="text-xl font-bold text-neutral-900 dark:text-white">New Reach Report</h1>

      <!-- Public notice -->
      <div class="flex items-start gap-2 rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 px-4 py-3 text-sm text-amber-800 dark:text-amber-300">
        <svg class="w-4 h-4 mt-0.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
        All reach reports are public on this site. Please be courteous.
      </div>

      <form class="space-y-5" @submit.prevent="submit">

        <!-- Reach picker -->
        <div>
          <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">Reach <span class="text-red-500">*</span></label>
          <div class="relative">
            <input
              v-model="reachQuery"
              type="text"
              placeholder="Search for a reach…"
              class="w-full rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm text-neutral-900 dark:text-white placeholder-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
              autocomplete="off"
              @input="selectedReach = null"
              @focus="showReachDropdown = true"
              @blur="onReachBlur"
            />
            <div
              v-if="showReachDropdown && filteredReaches.length > 0"
              class="absolute z-20 left-0 right-0 top-full mt-1 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-700 rounded-lg shadow-lg max-h-56 overflow-y-auto"
            >
              <button
                v-for="r in filteredReaches.slice(0, 12)"
                :key="r.slug"
                type="button"
                class="w-full text-left px-3 py-2 text-sm hover:bg-primary-50 dark:hover:bg-primary-950/30 flex flex-col gap-0.5"
                @mousedown.prevent="selectReach(r)"
              >
                <span class="font-medium text-neutral-800 dark:text-neutral-100">{{ reachDisplayName(r) }}</span>
                <span v-if="r.river_name" class="text-xs text-neutral-400">{{ r.river_name }}</span>
              </button>
            </div>
          </div>
          <p v-if="selectedReach" class="mt-1 text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
            <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M20 6 9 17l-5-5"/></svg>
            {{ reachDisplayName(selectedReach) }}
          </p>
        </div>

        <!-- Date + Time -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">Date <span class="text-red-500">*</span></label>
            <input
              v-model="form.report_date"
              type="date"
              :max="today"
              required
              class="w-full rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">Time <span class="text-neutral-400 font-normal">(optional)</span></label>
            <input
              v-model="form.report_time"
              type="time"
              class="w-full rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
        </div>

        <!-- Name -->
        <div>
          <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">Your name <span class="text-red-500">*</span></label>
          <input
            v-model="form.name"
            type="text"
            placeholder="e.g. Jane Paddler"
            maxlength="80"
            required
            class="w-full rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm text-neutral-900 dark:text-white placeholder-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>

        <!-- Content -->
        <div>
          <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">Report <span class="text-red-500">*</span></label>
          <MarkdownEditor
            v-model="form.content"
            :rows="6"
            placeholder="Describe conditions, flow, any notable observations…"
            :required="true"
          />
        </div>

        <!-- Hazard warning -->

        <div>
          <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">
            Hazard warning
            <span class="text-neutral-400 font-normal">(optional — shown prominently)</span>
          </label>
          <textarea
            v-model="form.hazard_warning"
            rows="2"
            placeholder="e.g. Strainer at the bottom of Gorge rapid, river left"
            class="w-full rounded-lg border border-red-200 dark:border-red-800 bg-white dark:bg-neutral-900 px-3 py-2 text-sm text-neutral-900 dark:text-white placeholder-neutral-400 focus:outline-none focus:ring-2 focus:ring-red-400 resize-y"
          />
        </div>

        <!-- Paddled toggle -->
        <div class="flex items-center gap-3">
          <button
            type="button"
            class="relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
            :class="form.paddled ? 'bg-primary-500' : 'bg-neutral-200 dark:bg-neutral-700'"
            role="switch"
            :aria-checked="form.paddled"
            @click="form.paddled = !form.paddled"
          >
            <span
              class="inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform"
              :class="form.paddled ? 'translate-x-4' : 'translate-x-0'"
            />
          </button>
          <span class="text-sm text-neutral-700 dark:text-neutral-300">I paddled this reach</span>
        </div>

        <!-- Photo stub -->
        <div>
          <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">Photos <span class="text-neutral-400 font-normal">(coming soon)</span></label>
          <div class="flex items-center gap-2 rounded-lg border border-dashed border-neutral-200 dark:border-neutral-700 bg-neutral-50 dark:bg-neutral-900/50 px-4 py-3 text-sm text-neutral-400">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 15-5-5L5 21"/></svg>
            Photo upload not yet available
          </div>
        </div>

        <!-- Error -->
        <div v-if="error" class="rounded-lg bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 px-4 py-3 text-sm text-red-700 dark:text-red-400">
          {{ error }}
        </div>

        <!-- Submit -->
        <div class="flex items-center justify-end gap-3 pt-2">
          <NuxtLink
            v-if="prefillSlug"
            :to="`/reaches/${prefillSlug}`"
            class="text-sm text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300 transition-colors"
          >
            Cancel
          </NuxtLink>
          <button
            v-else
            type="button"
            class="text-sm text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300 transition-colors"
            @click="router.back()"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="submitting || !selectedReach"
            class="inline-flex items-center gap-2 rounded-lg bg-primary-600 hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed px-5 py-2 text-sm font-medium text-white transition-colors"
          >
            <div v-if="submitting" class="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
            Submit report
          </button>
        </div>

      </form>
    </main>

    <!-- Live card preview -->
    <div v-if="form.name || form.content || form.hazard_warning" class="mt-2">
      <p class="text-xs font-semibold uppercase tracking-wide text-neutral-400 dark:text-neutral-500 mb-2">Card preview</p>
      <div class="bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-700 overflow-hidden">
        <div class="px-4 py-3 space-y-1">
          <div v-if="form.hazard_warning.trim()" class="flex items-start gap-2 mb-1.5 rounded-md bg-red-50 dark:bg-red-950/30 border border-red-100 dark:border-red-900 px-2.5 py-1.5">
            <svg class="w-3.5 h-3.5 mt-0.5 shrink-0 text-red-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
            <p class="text-xs text-red-700 dark:text-red-400">{{ form.hazard_warning }}</p>
          </div>
          <div class="flex items-start justify-between gap-2">
            <span class="text-sm font-medium text-neutral-800 dark:text-neutral-100">{{ form.name || 'Your name' }}</span>
            <span class="text-xs text-neutral-400 shrink-0">{{ formatPreviewDate(form.report_date) }}</span>
          </div>
          <p class="text-sm text-neutral-600 dark:text-neutral-400 line-clamp-3 leading-relaxed min-h-5">
            {{ extractPreview(form.content) || 'Your report will appear here…' }}
          </p>
          <div class="flex items-center gap-2 pt-0.5">
            <span class="text-xs text-neutral-400 italic">flow recorded at submit</span>
            <span v-if="form.paddled" class="text-xs text-primary-500 dark:text-primary-400">• paddled</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Hazard confirm dialog -->
    <Teleport to="body">
      <div
        v-if="hazardConfirmOpen"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-sm px-4"
      >
        <div class="w-full max-w-sm bg-white dark:bg-neutral-900 rounded-2xl border border-neutral-200 dark:border-neutral-700 shadow-2xl p-6 space-y-4">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-full bg-amber-100 dark:bg-amber-950/50 flex items-center justify-center shrink-0">
              <svg class="w-5 h-5 text-amber-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
            </div>
            <h3 class="text-base font-semibold text-neutral-900 dark:text-white">Hazard notification</h3>
          </div>
          <p class="text-sm text-neutral-600 dark:text-neutral-400">
            Your hazard warning will appear on the dashboards of all H2OFlows users watching this reach. Please make sure the description is accurate and specific.
          </p>
          <div class="rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 px-3 py-2 text-sm text-amber-800 dark:text-amber-300 italic">
            "{{ form.hazard_warning }}"
          </div>
          <div class="flex items-center justify-end gap-3 pt-1">
            <button
              class="text-sm text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300 transition-colors"
              @click="hazardConfirmOpen = false"
            >Edit warning</button>
            <button
              class="inline-flex items-center gap-2 rounded-lg bg-amber-500 hover:bg-amber-600 px-4 py-2 text-sm font-medium text-white transition-colors"
              @click="hazardConfirmOpen = false; doSubmit()"
            >Submit report</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ ssr: false })

const router = useRouter()
const route = useRoute()
const config = useRuntimeConfig()
const { isAuthenticated, getToken } = useAuth()

const authReady = ref(false)
onMounted(() => { authReady.value = true })

const today = new Date().toISOString().slice(0, 10)

const prefillSlug = computed(() => route.query.reach as string | undefined)

const form = ref({
  report_date: today,
  report_time: '',
  name: '',
  content: '',
  hazard_warning: '',
  paddled: false,
})

interface ReachItem {
  slug: string
  river_name?: string | null
  common_name?: string | null
  put_in_name?: string | null
  take_out_name?: string | null
}

const allReaches = ref<ReachItem[]>([])
const reachQuery = ref('')
const selectedReach = ref<ReachItem | null>(null)
const showReachDropdown = ref(false)

function reachDisplayName(r: ReachItem): string {
  if (r.common_name) return r.common_name
  if (r.put_in_name && r.take_out_name) return `${r.put_in_name} to ${r.take_out_name}`
  return r.slug
}

const filteredReaches = computed(() => {
  const q = reachQuery.value.toLowerCase().trim()
  if (!q) return allReaches.value.slice(0, 12)
  return allReaches.value.filter(r => {
    const name = reachDisplayName(r).toLowerCase()
    const river = (r.river_name ?? '').toLowerCase()
    return name.includes(q) || river.includes(q) || r.slug.includes(q)
  })
})

function selectReach(r: ReachItem) {
  selectedReach.value = r
  reachQuery.value = reachDisplayName(r)
  showReachDropdown.value = false
}

function onReachBlur() {
  setTimeout(() => { showReachDropdown.value = false }, 150)
}

onMounted(async () => {
  const data = await $fetch<ReachItem[]>(`${config.public.apiBase}/api/v1/reaches`).catch(() => [])
  allReaches.value = data ?? []

  if (prefillSlug.value) {
    const found = allReaches.value.find(r => r.slug === prefillSlug.value)
    if (found) selectReach(found)
  }
})

const submitting = ref(false)
const error = ref('')
const hazardConfirmOpen = ref(false)

function submit() {
  if (!selectedReach.value) return
  if (form.value.hazard_warning.trim() && !hazardConfirmOpen.value) {
    hazardConfirmOpen.value = true
    return
  }
  doSubmit()
}

async function doSubmit() {
  if (!selectedReach.value) return
  error.value = ''
  submitting.value = true
  try {
    const token = await getToken()
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (token) headers['Authorization'] = `Bearer ${token}`

    const body: Record<string, unknown> = {
      name: form.value.name,
      report_date: form.value.report_date,
      content: form.value.content,
      paddled: form.value.paddled,
    }
    if (form.value.report_time) body.report_time = form.value.report_time
    if (form.value.hazard_warning.trim()) body.hazard_warning = form.value.hazard_warning.trim()

    const res = await fetch(
      `${config.public.apiBase}/api/v1/reaches/${selectedReach.value.slug}/reports`,
      { method: 'POST', headers, body: JSON.stringify(body) }
    )
    const data = await res.json()
    if (!res.ok) {
      error.value = data.error ?? 'Failed to submit report'
      return
    }
    router.push(`/reports/${data.id}`)
  } catch (e: any) {
    error.value = e?.message ?? 'Network error'
  } finally {
    submitting.value = false
  }
}

function extractPreview(content: string): string {
  const firstPara = content.split(/\n\n+/)[0].trim()
  return firstPara
    .replace(/#{1,6}\s+/g, '')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/\*([^*]+)\*/g, '$1')
    .replace(/~~([^~]+)~~/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/^\s*[-*+]\s+/gm, '')
    .replace(/^\s*\d+\.\s+/gm, '')
    .replace(/^\s*>\s*/gm, '')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/\n/g, ' ')
    .trim()
}

function formatPreviewDate(d: string): string {
  if (!d) return ''
  const [y, m, day] = d.split('-').map(Number)
  return new Date(y, m - 1, day).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}
</script>
