<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader>
      <span class="text-gray-300 dark:text-gray-700 shrink-0">/</span>
      <span class="text-sm font-medium text-gray-700 dark:text-gray-200">New Report</span>
    </AppHeader>

    <div v-if="!authReady" class="max-w-2xl mx-auto px-4 py-20 flex justify-center">
      <div class="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
    </div>

    <div v-else-if="!isAuthenticated" class="max-w-2xl mx-auto px-4 py-20 flex flex-col items-center gap-3 text-center">
      <svg class="w-10 h-10 text-gray-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
      </svg>
      <h2 class="text-lg font-semibold">Sign in to submit a report</h2>
      <NuxtLink to="/login" class="text-sm text-blue-600 dark:text-blue-400 hover:underline">Sign in</NuxtLink>
    </div>

    <main v-else class="max-w-2xl mx-auto px-4 py-8 space-y-6">
      <h1 class="text-xl font-bold text-gray-900 dark:text-white">New Reach Report</h1>

      <!-- Public notice -->
      <div class="flex items-start gap-2 rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 px-4 py-3 text-sm text-amber-800 dark:text-amber-300">
        <svg class="w-4 h-4 mt-0.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
        All reach reports are public on this site. Please be courteous.
      </div>

      <form class="space-y-5" @submit.prevent="submit">

        <!-- Reach picker -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Reach <span class="text-red-500">*</span></label>
          <div class="relative">
            <input
              v-model="reachQuery"
              type="text"
              placeholder="Search for a reach…"
              class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              autocomplete="off"
              @input="selectedReach = null"
              @focus="showReachDropdown = true"
              @blur="onReachBlur"
            />
            <div
              v-if="showReachDropdown && filteredReaches.length > 0"
              class="absolute z-20 left-0 right-0 top-full mt-1 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg max-h-56 overflow-y-auto"
            >
              <button
                v-for="r in filteredReaches.slice(0, 12)"
                :key="r.slug"
                type="button"
                class="w-full text-left px-3 py-2 text-sm hover:bg-blue-50 dark:hover:bg-blue-950/30 flex flex-col gap-0.5"
                @mousedown.prevent="selectReach(r)"
              >
                <span class="font-medium text-gray-800 dark:text-gray-100">{{ reachDisplayName(r) }}</span>
                <span v-if="r.river_name" class="text-xs text-gray-400">{{ r.river_name }}</span>
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
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Date <span class="text-red-500">*</span></label>
            <input
              v-model="form.report_date"
              type="date"
              :max="today"
              required
              class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Time <span class="text-gray-400 font-normal">(optional)</span></label>
            <input
              v-model="form.report_time"
              type="time"
              class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        <!-- Name -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Your name <span class="text-red-500">*</span></label>
          <input
            v-model="form.name"
            type="text"
            placeholder="e.g. Jane Paddler"
            maxlength="80"
            required
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <!-- Content -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Report <span class="text-red-500">*</span></label>
          <textarea
            v-model="form.content"
            rows="5"
            placeholder="Describe conditions, flow, any notable observations…"
            required
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-y"
          />
        </div>

        <!-- Hazard warning -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Hazard warning
            <span class="text-gray-400 font-normal">(optional — shown prominently)</span>
          </label>
          <textarea
            v-model="form.hazard_warning"
            rows="2"
            placeholder="e.g. Strainer at the bottom of Gorge rapid, river left"
            class="w-full rounded-lg border border-red-200 dark:border-red-800 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-red-400 resize-y"
          />
        </div>

        <!-- Paddled toggle -->
        <div class="flex items-center gap-3">
          <button
            type="button"
            class="relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
            :class="form.paddled ? 'bg-blue-500' : 'bg-gray-200 dark:bg-gray-700'"
            role="switch"
            :aria-checked="form.paddled"
            @click="form.paddled = !form.paddled"
          >
            <span
              class="inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform"
              :class="form.paddled ? 'translate-x-4' : 'translate-x-0'"
            />
          </button>
          <span class="text-sm text-gray-700 dark:text-gray-300">I paddled this reach</span>
        </div>

        <!-- Photo stub -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Photos <span class="text-gray-400 font-normal">(coming soon)</span></label>
          <div class="flex items-center gap-2 rounded-lg border border-dashed border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50 px-4 py-3 text-sm text-gray-400">
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
            class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
          >
            Cancel
          </NuxtLink>
          <button
            v-else
            type="button"
            class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
            @click="router.back()"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="submitting || !selectedReach"
            class="inline-flex items-center gap-2 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed px-5 py-2 text-sm font-medium text-white transition-colors"
          >
            <div v-if="submitting" class="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
            Submit report
          </button>
        </div>

      </form>
    </main>
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

async function submit() {
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
    router.push(`/reports/${data.handle}/${data.slug}`)
  } catch (e: any) {
    error.value = e?.message ?? 'Network error'
  } finally {
    submitting.value = false
  }
}
</script>
