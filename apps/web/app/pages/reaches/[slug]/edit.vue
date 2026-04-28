<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader>
      <template #default>
        <span class="text-gray-300 dark:text-gray-700 shrink-0">/</span>
        <NuxtLink
          :to="`/reaches/${slug}`"
          class="text-sm font-medium truncate text-gray-500 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
        >{{ slug }}</NuxtLink>
        <span class="text-gray-300 dark:text-gray-700 shrink-0">/</span>
        <span class="text-sm font-medium text-gray-700 dark:text-gray-200">edit</span>
      </template>
    </AppHeader>

    <!-- Not authorized -->
    <div v-if="!isDataAdmin && authReady" class="mt-20 flex flex-col items-center gap-3 text-center">
      <svg class="w-10 h-10 text-gray-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
      <h2 class="text-lg font-semibold">Access restricted</h2>
      <p class="text-sm text-gray-500">You need data admin permissions to edit reaches.</p>
      <NuxtLink :to="`/reaches/${slug}`" class="text-sm text-blue-500 hover:text-blue-400">Back to reach</NuxtLink>
    </div>

    <!-- Loading auth -->
    <div v-else-if="!authReady" class="mt-20 flex justify-center">
      <div class="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
    </div>

    <main v-else class="max-w-5xl mx-auto px-4 py-6 space-y-6">
      <div class="flex items-center justify-between flex-wrap gap-2">
        <div>
          <h1 class="text-xl font-bold text-gray-900 dark:text-white">Edit reach</h1>
          <p class="text-xs text-gray-400 font-mono mt-0.5">{{ slug }}</p>
        </div>
        <NuxtLink :to="`/reaches/${slug}`" class="text-sm text-blue-500 hover:text-blue-400 transition-colors">
          View reach
        </NuxtLink>
      </div>

      <ReachEditor
        :slug="slug"
        :rivers="rivers"
        @slug-changed="onSlugChanged"
        @rivers-updated="loadRivers"
      />

      <div class="border-t border-gray-200 dark:border-gray-800 pt-6">
        <KmlImportPanel />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'

definePageMeta({ ssr: false })

const route  = useRoute()
const router = useRouter()
const { isDataAdmin, loadAdminRoles, getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public

const slug      = computed(() => route.params.slug as string)
const authReady = ref(false)

onMounted(async () => {
  await loadAdminRoles()
  authReady.value = true
  if (isDataAdmin.value) loadRivers()
})

watch(isDataAdmin, (val) => {
  if (val && authReady.value && rivers.value.length === 0) loadRivers()
})

// ── Rivers (needed by ReachEditor river selector) ────────────────────────────
interface River { id: string; slug: string; name: string; gnis_id: string | null; basin: string | null; basin_locked: boolean; state_abbr: string | null; reach_count: number }

const rivers = ref<River[]>([])

async function loadRivers() {
  const token = await getToken()
  if (!token) return
  try {
    const res = await fetch(`${apiBase}/api/v1/admin/rivers`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (res.ok) rivers.value = await res.json()
  } catch { /* non-fatal */ }
}

// ── Slug rename handler ───────────────────────────────────────────────────────
async function onSlugChanged(newSlug: string) {
  await router.replace(`/reaches/${newSlug}/edit`)
}
</script>
