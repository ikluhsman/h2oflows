<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-neutral-950">
    <AppHeader>
      <span class="text-neutral-300 dark:text-neutral-700 shrink-0">/</span>
      <span class="text-sm font-medium text-neutral-700 dark:text-neutral-200">My Reports</span>
    </AppHeader>

    <div v-if="!authReady" class="max-w-3xl mx-auto px-4 py-20 flex justify-center">
      <div class="w-6 h-6 rounded-full border-2 border-primary-500 border-t-transparent animate-spin" />
    </div>

    <div v-else-if="!isAuthenticated" class="max-w-3xl mx-auto px-4 py-20 flex flex-col items-center gap-3 text-center">
      <svg class="w-10 h-10 text-neutral-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
      </svg>
      <h2 class="text-lg font-semibold">Sign in to view your reports</h2>
      <NuxtLink to="/login" class="text-sm text-primary-600 dark:text-primary-400 hover:underline">Sign in</NuxtLink>
    </div>

    <main v-else class="max-w-3xl mx-auto px-4 py-6 pb-24 sm:pb-6 space-y-5">
      <div class="flex items-center justify-between">
        <h1 class="text-xl font-bold text-neutral-900 dark:text-white">My Reports</h1>
        <NuxtLink
          to="/reports/new"
          class="inline-flex items-center gap-1.5 rounded-lg bg-primary-600 hover:bg-primary-700 px-3 py-1.5 text-sm font-medium text-white transition-colors"
        >
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
          New report
        </NuxtLink>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="space-y-3">
        <div v-for="i in 3" :key="i" class="h-24 rounded-xl bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
      </div>

      <!-- Error -->
      <div v-else-if="fetchError" class="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 p-4 text-sm text-red-700 dark:text-red-400">
        {{ fetchError }}
      </div>

      <!-- Empty -->
      <div v-else-if="reports.length === 0" class="mt-16 flex flex-col items-center gap-3 text-center text-neutral-400">
        <svg class="w-10 h-10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/>
        </svg>
        <p class="text-sm">No reports yet.</p>
        <NuxtLink to="/reports/new" class="text-sm text-primary-600 dark:text-primary-400 hover:underline">File your first report</NuxtLink>
      </div>

      <!-- Report cards -->
      <div v-else class="space-y-3">
        <div
          v-for="rep in reports"
          :key="rep.id"
          class="bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-700 px-4 py-3 space-y-2"
        >
          <!-- Top row: name + date -->
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span
                  v-if="rep.hazard_warning"
                  class="inline-flex items-center gap-1 rounded bg-red-50 dark:bg-red-950 px-1.5 py-0.5 text-xs font-medium text-red-700 dark:text-red-400"
                >
                  <svg class="w-2.5 h-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
                  Hazard
                </span>
                <span class="text-sm font-medium text-neutral-800 dark:text-neutral-100">{{ rep.name }}</span>
              </div>
              <NuxtLink
                v-if="rep.reach_slug"
                :to="`/reaches/${rep.reach_slug}`"
                class="text-xs text-primary-500 dark:text-primary-400 hover:underline"
              >{{ rep.reach_name ?? rep.reach_slug }}</NuxtLink>
            </div>
            <span class="text-xs text-neutral-400 shrink-0">{{ formatDate(rep.report_date) }}</span>
          </div>

          <!-- Content snippet -->
          <p class="text-sm text-neutral-600 dark:text-neutral-400 line-clamp-2">{{ extractPreview(rep.content) }}</p>

          <!-- Flow badge -->
          <div v-if="rep.flow_cfs != null" class="flex items-center gap-1.5">
            <span class="text-xs text-neutral-500 dark:text-neutral-400">{{ Math.round(rep.flow_cfs).toLocaleString() }} cfs</span>
            <span v-if="rep.flow_band" :class="bandChipClass(rep.flow_band)" class="text-xs font-medium capitalize">{{ rep.flow_band }}</span>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-3 pt-1">
            <NuxtLink
              v-if="rep.url"
              :to="rep.url"
              class="text-xs text-primary-600 dark:text-primary-400 hover:underline"
            >View</NuxtLink>
            <button
              v-if="canEdit(rep.created_at)"
              class="text-xs text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300 transition-colors"
              @click="editReport(rep)"
            >Edit</button>
            <button
              class="text-xs text-red-500 hover:text-red-600 dark:hover:text-red-400 transition-colors"
              @click="confirmDelete(rep)"
            >Delete</button>
          </div>
        </div>
      </div>

      <!-- Delete confirm dialog -->
      <Teleport to="body">
        <div
          v-if="deleteTarget"
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-sm px-4"
        >
          <div class="w-full max-w-sm bg-white dark:bg-neutral-900 rounded-2xl border border-neutral-200 dark:border-neutral-700 shadow-2xl p-6 space-y-4">
            <h3 class="text-base font-semibold text-neutral-900 dark:text-white">Delete report?</h3>
            <p class="text-sm text-neutral-500 dark:text-neutral-400">
              "{{ deleteTarget.name }}" will be permanently removed and cannot be recovered.
            </p>
            <div v-if="deleteError" class="text-sm text-red-600 dark:text-red-400">{{ deleteError }}</div>
            <div class="flex items-center justify-end gap-3">
              <button
                class="text-sm text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300 transition-colors"
                @click="deleteTarget = null; deleteError = ''"
              >Cancel</button>
              <button
                :disabled="deleting"
                class="inline-flex items-center gap-2 rounded-lg bg-red-600 hover:bg-red-700 disabled:opacity-50 px-4 py-2 text-sm font-medium text-white transition-colors"
                @click="doDelete"
              >
                <div v-if="deleting" class="w-3.5 h-3.5 rounded-full border-2 border-white/30 border-t-white animate-spin" />
                Delete
              </button>
            </div>
          </div>
        </div>
      </Teleport>

    </main>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ ssr: false })

const router = useRouter()
const config = useRuntimeConfig()
const { isAuthenticated, getToken } = useAuth()

const authReady = ref(false)
onMounted(() => { authReady.value = true })

interface MyReport {
  id: string
  slug: string
  name: string
  report_date: string
  content: string
  hazard_warning?: string
  paddled: boolean
  flow_cfs?: number
  flow_band?: string
  created_at: string
  reach_name?: string
  reach_slug?: string
  url?: string
}

const reports = ref<MyReport[]>([])
const loading = ref(false)
const fetchError = ref('')

async function load() {
  loading.value = true
  fetchError.value = ''
  try {
    const token = await getToken()
    const headers: Record<string, string> = {}
    if (token) headers['Authorization'] = `Bearer ${token}`
    const res = await fetch(`${config.public.apiBase}/api/v1/me/reports`, { headers })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    reports.value = await res.json()
  } catch (e: any) {
    fetchError.value = e?.message ?? 'Failed to load reports'
  } finally {
    loading.value = false
  }
}

watch(isAuthenticated, (v) => { if (v) load() }, { immediate: true })

function formatDate(d: string): string {
  const [y, m, day] = d.split('-').map(Number)
  return new Date(y, m - 1, day).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function bandChipClass(band: string): string {
  if (band === 'low') return 'text-sky-600 dark:text-sky-400'
  if (band === 'running') return 'text-emerald-600 dark:text-emerald-400'
  if (band === 'high') return 'text-amber-600 dark:text-amber-400'
  return 'text-neutral-500'
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

function canEdit(createdAt: string): boolean {
  const ms = Date.now() - new Date(createdAt).getTime()
  return ms < 24 * 60 * 60 * 1000
}

function editReport(rep: MyReport) {
  router.push(`/my/reports/${rep.slug}`)
}

const deleteTarget = ref<MyReport | null>(null)
const deleting = ref(false)
const deleteError = ref('')

function confirmDelete(rep: MyReport) {
  deleteTarget.value = rep
  deleteError.value = ''
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  deleteError.value = ''
  try {
    const token = await getToken()
    const headers: Record<string, string> = {}
    if (token) headers['Authorization'] = `Bearer ${token}`
    const res = await fetch(
      `${config.public.apiBase}/api/v1/me/reports/${deleteTarget.value.slug}`,
      { method: 'DELETE', headers }
    )
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      throw new Error(d.error ?? `HTTP ${res.status}`)
    }
    reports.value = reports.value.filter(r => r.id !== deleteTarget.value!.id)
    deleteTarget.value = null
  } catch (e: any) {
    deleteError.value = e?.message ?? 'Delete failed'
  } finally {
    deleting.value = false
  }
}
</script>
