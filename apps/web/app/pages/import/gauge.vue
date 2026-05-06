<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <div class="flex justify-center px-4 mt-16">
      <div class="w-full max-w-md space-y-5">
        <div class="rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 space-y-4">
          <h1 class="text-lg font-bold text-gray-900 dark:text-white">Import Custom Gauge</h1>

          <!-- Auth not ready -->
          <div v-if="!authReady" class="flex justify-center py-4">
            <div class="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
          </div>

          <!-- Not authenticated -->
          <div v-else-if="!isAuthenticated" class="space-y-3">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              Sign in to import this custom gauge formula to your account.
            </p>
            <UButton block @click="router.push('/login')">Sign in to import</UButton>
          </div>

          <!-- No token -->
          <div v-else-if="!token" class="text-sm text-red-500">
            No share token in URL. Check the link and try again.
          </div>

          <!-- Success -->
          <div v-else-if="importedSlug" class="space-y-3">
            <p class="text-sm text-green-600 dark:text-green-400 font-medium">Gauge imported successfully.</p>
            <UButton block @click="router.push(`/my/gauges/${importedSlug}`)">View gauge</UButton>
          </div>

          <!-- Ready -->
          <div v-else class="space-y-4">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              Import a shared custom gauge formula into your account.
            </p>
            <div v-if="importError" class="rounded border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 p-3 text-sm text-red-600 dark:text-red-400">
              {{ importError }}
            </div>
            <UButton block :loading="importing" @click="doImport">Import gauge</UButton>
            <UButton block variant="ghost" color="neutral" @click="router.push('/')">Cancel</UButton>
          </div>

        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

const route = useRoute()
const router = useRouter()
const { isAuthenticated, getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public

const token = computed(() => route.query.token as string | undefined)
const authReady = ref(false)
const importing = ref(false)
const importError = ref<string | null>(null)
const importedSlug = ref<string | null>(null)

onMounted(() => { authReady.value = true })

async function doImport() {
  if (!token.value) return
  importing.value = true
  importError.value = null
  try {
    const jwt = await getToken()
    const res = await fetch(`${apiBase}/api/v1/me/custom-gauges/import`, {
      method:  'POST',
      headers: { 'Content-Type': 'application/json', ...(jwt ? { Authorization: `Bearer ${jwt}` } : {}) },
      body: JSON.stringify({ token: token.value }),
    })
    const data = await res.json()
    if (!res.ok) { importError.value = data.error ?? 'Import failed'; return }
    importedSlug.value = data.slug
  } catch (e: any) {
    importError.value = e.message
  } finally {
    importing.value = false
  }
}
</script>
