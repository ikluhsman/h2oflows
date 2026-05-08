<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-neutral-950">
    <AppHeader>
      <span class="text-neutral-300 dark:text-neutral-700 shrink-0">/</span>
      <NuxtLink to="/my/reports" class="text-sm text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 transition-colors">My Reports</NuxtLink>
      <span class="text-neutral-300 dark:text-neutral-700 shrink-0">/</span>
      <span class="text-sm font-medium text-neutral-700 dark:text-neutral-200 truncate">Edit</span>
    </AppHeader>

    <!-- Loading -->
    <div v-if="!authReady || loading" class="max-w-2xl mx-auto px-4 py-20 flex justify-center">
      <div class="w-6 h-6 rounded-full border-2 border-primary-500 border-t-transparent animate-spin" />
    </div>

    <!-- Not authed -->
    <div v-else-if="!isAuthenticated" class="max-w-2xl mx-auto px-4 py-20 flex flex-col items-center gap-3 text-center">
      <svg class="w-10 h-10 text-neutral-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
      </svg>
      <h2 class="text-lg font-semibold">Sign in to edit reports</h2>
      <NuxtLink to="/login" class="text-sm text-primary-600 dark:text-primary-400 hover:underline">Sign in</NuxtLink>
    </div>

    <!-- Not found -->
    <div v-else-if="!report" class="max-w-2xl mx-auto px-4 py-20 text-center text-neutral-400 text-sm">
      Report not found.
    </div>

    <!-- Locked (>24h) -->
    <div v-else-if="locked" class="max-w-2xl mx-auto px-4 py-20 flex flex-col items-center gap-3 text-center">
      <svg class="w-10 h-10 text-neutral-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
      <p class="text-sm text-neutral-500 dark:text-neutral-400">Reports are locked for editing after 24 hours.</p>
      <NuxtLink to="/my/reports" class="text-sm text-primary-600 dark:text-primary-400 hover:underline">Back to my reports</NuxtLink>
    </div>

    <main v-else class="max-w-2xl mx-auto px-4 py-8 pb-24 sm:pb-8 space-y-6">
      <h1 class="text-xl font-bold text-neutral-900 dark:text-white">Edit Report</h1>

      <!-- Reach (read-only) -->
      <div class="flex items-center gap-2 rounded-lg bg-neutral-100 dark:bg-neutral-800 px-4 py-3 text-sm">
        <svg class="w-4 h-4 text-neutral-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 14c3-6 6-9 8-9s5 9 8 9"/></svg>
        <span class="text-neutral-500 dark:text-neutral-400 shrink-0">Reach:</span>
        <NuxtLink
          :to="`/reaches/${report.reach_slug}`"
          class="font-medium text-primary-600 dark:text-primary-400 hover:underline truncate"
        >{{ report.reach_name || report.reach_slug }}</NuxtLink>
      </div>

      <form class="space-y-5" @submit.prevent="submit">

        <!-- Date (read-only) -->
        <div>
          <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">Date</label>
          <div class="w-full rounded-lg border border-neutral-200 dark:border-neutral-700 bg-neutral-50 dark:bg-neutral-900/50 px-3 py-2 text-sm text-neutral-500 dark:text-neutral-400">
            {{ formatDate(report.report_date) }}
          </div>
        </div>

        <!-- Name + Time -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">Your name <span class="text-red-500">*</span></label>
            <input
              v-model="form.name"
              type="text"
              maxlength="80"
              required
              class="w-full rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm text-neutral-900 dark:text-white placeholder-neutral-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
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

        <!-- Error -->
        <div v-if="error" class="rounded-lg bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 px-4 py-3 text-sm text-red-700 dark:text-red-400">
          {{ error }}
        </div>

        <!-- Actions -->
        <div class="flex items-center justify-between gap-3 pt-2">
          <NuxtLink
            to="/my/reports"
            class="text-sm text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300 transition-colors"
          >Cancel</NuxtLink>
          <div class="flex items-center gap-3">
            <span v-if="saved" class="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
              <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M20 6 9 17l-5-5"/></svg>
              Saved
            </span>
            <button
              type="submit"
              :disabled="submitting"
              class="inline-flex items-center gap-2 rounded-lg bg-primary-600 hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed px-5 py-2 text-sm font-medium text-white transition-colors"
            >
              <div v-if="submitting" class="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
              Save changes
            </button>
          </div>
        </div>

      </form>
    </main>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ ssr: false })

const route = useRoute()
const router = useRouter()
const config = useRuntimeConfig()
const { isAuthenticated, getToken } = useAuth()

const slug = route.params.slug as string

const authReady = ref(false)
onMounted(() => { authReady.value = true })

interface MyReport {
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
  created_at: string
  reach_name: string
  reach_slug: string
}

const report = ref<MyReport | null>(null)
const loading = ref(false)
const locked = ref(false)

const form = ref({
  name: '',
  report_time: '',
  content: '',
  hazard_warning: '',
  paddled: false,
})

async function load() {
  loading.value = true
  try {
    const token = await getToken()
    const headers: Record<string, string> = {}
    if (token) headers['Authorization'] = `Bearer ${token}`
    const list = await fetch(`${config.public.apiBase}/api/v1/me/reports`, { headers })
    if (!list.ok) return
    const data: MyReport[] = await list.json()
    const found = data.find(r => r.slug === slug)
    if (!found) return
    report.value = found
    locked.value = Date.now() - new Date(found.created_at).getTime() > 24 * 60 * 60 * 1000
    form.value = {
      name: found.name,
      report_time: found.report_time ?? '',
      content: found.content,
      hazard_warning: found.hazard_warning ?? '',
      paddled: found.paddled,
    }
  } finally {
    loading.value = false
  }
}

watch(isAuthenticated, (v) => { if (v) load() }, { immediate: true })

const submitting = ref(false)
const error = ref('')
const saved = ref(false)

async function submit() {
  if (!report.value) return
  submitting.value = true
  error.value = ''
  saved.value = false
  try {
    const token = await getToken()
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (token) headers['Authorization'] = `Bearer ${token}`

    const body: Record<string, unknown> = {
      name: form.value.name,
      content: form.value.content,
      hazard_warning: form.value.hazard_warning.trim(),
      paddled: form.value.paddled,
    }
    if (form.value.report_time) body.report_time = form.value.report_time

    const res = await fetch(
      `${config.public.apiBase}/api/v1/me/reports/${slug}`,
      { method: 'PATCH', headers, body: JSON.stringify(body) }
    )
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      error.value = d.error ?? `HTTP ${res.status}`
      return
    }
    saved.value = true
    setTimeout(() => router.push('/my/reports'), 800)
  } catch (e: any) {
    error.value = e?.message ?? 'Network error'
  } finally {
    submitting.value = false
  }
}

function formatDate(d: string): string {
  const [y, m, day] = d.split('-').map(Number)
  return new Date(y, m - 1, day).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
}
</script>
