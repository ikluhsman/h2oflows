<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-3xl mx-auto px-4 py-6 space-y-5">

      <div v-if="!isAuthenticated && authReady" class="mt-20 flex flex-col items-center gap-3 text-center">
        <h2 class="text-lg font-semibold">Sign in to view your custom gauges</h2>
        <NuxtLink to="/login" class="text-sm text-blue-600 dark:text-blue-400 hover:underline">Sign in</NuxtLink>
      </div>

      <div v-else-if="!authReady" class="mt-20 flex justify-center">
        <div class="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
      </div>

      <template v-else>
        <div class="flex items-center justify-between">
          <div>
            <h1 class="text-xl font-bold text-gray-900 dark:text-white">My Gauges</h1>
            <p class="text-xs text-gray-400 mt-0.5">{{ items.length }}/25 used</p>
          </div>
          <UButton icon="i-heroicons-plus" size="sm" :disabled="items.length >= 25" @click="router.push('/my/gauges/new')">
            New gauge
          </UButton>
        </div>

        <div v-if="loading" class="space-y-2">
          <div v-for="i in 3" :key="i" class="h-16 rounded-lg bg-gray-100 dark:bg-gray-800 animate-pulse" />
        </div>

        <div v-else-if="error" class="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 p-4 text-sm text-red-600 dark:text-red-400">
          {{ error }}
        </div>

        <div v-else-if="items.length === 0" class="mt-10 flex flex-col items-center gap-3 text-center text-gray-400">
          <svg class="w-10 h-10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M9 7h6m0 10v-3m-3 3v-6m-3 6v-1M3 6h18M3 12h18M3 18h18" stroke-linecap="round"/>
          </svg>
          <p class="text-sm">No custom gauges yet. Create a sum or difference of real gauges.</p>
        </div>

        <div v-else class="space-y-2">
          <NuxtLink
            v-for="g in items"
            :key="g.id"
            :to="`/my/gauges/${g.slug}`"
            class="block rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 hover:border-blue-300 dark:hover:border-blue-700 transition-colors p-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-semibold text-gray-900 dark:text-white truncate">{{ g.name }}</p>
                <p v-if="g.description" class="text-xs text-gray-500 dark:text-gray-400 truncate mt-0.5">{{ g.description }}</p>
              </div>
              <div class="shrink-0 text-right">
                <span v-if="g.last_value_cfs != null" class="text-base font-bold tabular-nums text-gray-900 dark:text-white">
                  {{ g.last_value_cfs.toLocaleString() }}
                  <span class="text-xs font-normal text-gray-400">{{ g.unit }}</span>
                </span>
                <span v-else class="text-sm text-gray-400">—</span>
              </div>
            </div>
          </NuxtLink>
        </div>
      </template>

    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const { isAuthenticated, getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public
const router = useRouter()

const authReady = ref(false)
const loading = ref(false)
const error = ref('')

interface CustomGaugeSummary {
  id: string
  slug: string
  name: string
  description: string | null
  unit: string
  last_value_cfs: number | null
}

const items = ref<CustomGaugeSummary[]>([])

async function load() {
  loading.value = true
  error.value = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/me/custom-gauges`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) throw new Error(`${res.status}`)
    const data = await res.json()
    items.value = data.items ?? []
  } catch (e: any) {
    error.value = e.message ?? 'Failed to load.'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  authReady.value = true
  if (isAuthenticated.value) await load()
})
</script>
