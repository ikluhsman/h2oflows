<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-neutral-950">
    <AppHeader />

    <main class="max-w-3xl mx-auto px-4 py-6 space-y-5">

      <!-- Auth gate -->
      <div v-if="!isAuthenticated && authReady" class="mt-20 flex flex-col items-center gap-3 text-center">
        <svg class="w-10 h-10 text-neutral-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
        </svg>
        <h2 class="text-lg font-semibold">Sign in to view your reaches</h2>
        <NuxtLink to="/login" class="text-sm text-primary-600 dark:text-primary-400 hover:underline">Sign in</NuxtLink>
      </div>

      <div v-else-if="!authReady" class="mt-20 flex justify-center">
        <div class="w-6 h-6 rounded-full border-2 border-primary-500 border-t-transparent animate-spin" />
      </div>

      <template v-else>
        <div class="flex items-center justify-between">
          <h1 class="text-xl font-bold text-neutral-900 dark:text-white">My Reaches</h1>
          <UButton icon="i-heroicons-plus" size="sm" @click="router.push('/my/reaches/new')">New reach</UButton>
        </div>

        <!-- Loading -->
        <div v-if="loading" class="space-y-2">
          <div v-for="i in 3" :key="i" class="h-16 rounded-lg bg-neutral-100 dark:bg-neutral-800 animate-pulse" />
        </div>

        <!-- Error -->
        <div v-else-if="error" class="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 p-4 text-sm text-red-600 dark:text-red-400">
          {{ error }}
        </div>

        <!-- Empty state -->
        <div v-else-if="reaches.length === 0" class="mt-10 flex flex-col items-center gap-3 text-center text-neutral-400">
          <svg class="w-10 h-10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M3 6h18M3 12h18M3 18h18" stroke-linecap="round"/>
          </svg>
          <p class="text-sm">No reaches yet. Create your first custom reach.</p>
        </div>

        <!-- Reach list -->
        <div v-else class="space-y-2">
          <NuxtLink
            v-for="reach in reaches"
            :key="reach.id"
            :to="`/my/reaches/${reach.slug}`"
            class="block rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 hover:border-primary-300 dark:hover:border-primary-700 transition-colors p-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-semibold text-neutral-900 dark:text-white truncate">{{ reach.name }}</p>
                <p v-if="reach.river_name" class="text-xs text-neutral-500 dark:text-neutral-400 truncate mt-0.5">{{ reach.river_name }}</p>
              </div>
              <div class="shrink-0 flex items-center gap-2">
                <span
                  v-if="reach.flow_band"
                  class="text-xs font-medium px-2 py-0.5 rounded-full"
                  :class="bandBadgeClass(reach.flow_band)"
                >{{ flowBandLabel(reach.flow_band) }}</span>
                <span v-if="reach.cfs != null" class="text-sm font-mono text-neutral-700 dark:text-neutral-300">
                  {{ reach.cfs.toLocaleString() }} cfs
                </span>
              </div>
            </div>
            <p v-if="reach.note" class="text-xs text-neutral-400 mt-2 line-clamp-1">{{ reach.note }}</p>
          </NuxtLink>
        </div>
      </template>
    </main>

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { flowBandLabel } from '~/utils/flowBand'

const { bandBadgeClass } = useFlowBandPalette()

const { isAuthenticated, getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public
const router = useRouter()

const authReady = ref(false)
const loading   = ref(false)
const error     = ref('')

interface UserReach {
  id:          string
  slug:        string
  name:        string
  river_name:  string | null
  cfs:         number | null
  flow_band:   string | null
  note:        string | null
}

const reaches = ref<UserReach[]>([])

async function load() {
  loading.value = true
  error.value   = ''
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/me/reaches`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (res.status === 401) { error.value = 'Sign in to view your reaches.'; return }
    if (!res.ok) throw new Error(`${res.status}`)
    reaches.value = await res.json()
  } catch (e: any) {
    error.value = e.message ?? 'Failed to load reaches.'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  authReady.value = true
  if (isAuthenticated.value) await load()
})

</script>
