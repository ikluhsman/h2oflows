<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-2xl mx-auto px-4 py-6 space-y-5">

      <NuxtLink to="/my/gauges" class="text-sm text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
        ← Custom Gauges
      </NuxtLink>

      <div v-if="loading" class="space-y-4">
        <div class="h-8 w-48 rounded bg-gray-200 dark:bg-gray-700 animate-pulse" />
        <div class="h-40 rounded-xl bg-gray-100 dark:bg-gray-800 animate-pulse" />
      </div>

      <template v-else-if="gauge">
        <!-- Hero -->
        <div class="flex items-start justify-between gap-4">
          <div>
            <h1 class="text-xl font-bold text-gray-900 dark:text-white">{{ gauge.name }}</h1>
            <p v-if="gauge.description" class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">{{ gauge.description }}</p>
          </div>
          <div class="text-right shrink-0">
            <p v-if="gauge.last_value_cfs != null" class="text-2xl font-bold tabular-nums text-gray-900 dark:text-white">
              {{ gauge.last_value_cfs.toLocaleString() }}
              <span class="text-sm font-normal text-gray-400">{{ gauge.unit }}</span>
            </p>
            <p v-else class="text-gray-400">—</p>
            <p v-if="gauge.last_value_at" class="text-xs text-gray-400 mt-0.5">
              Updated {{ new Date(gauge.last_value_at).toLocaleTimeString() }}
            </p>
          </div>
        </div>

        <!-- Edit form -->
        <form class="space-y-5" @submit.prevent="save">
          <div class="rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-5 space-y-4">
            <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-300">Details</h2>

            <UFormField label="Name" required>
              <UInput v-model="form.name" class="w-full" required />
            </UFormField>

            <UFormField label="Description">
              <UInput v-model="form.description" class="w-full" />
            </UFormField>

            <UFormField label="Unit">
              <USelect v-model="form.unit" :items="['cfs', 'ft', 'acre-ft']" class="w-40" />
            </UFormField>
          </div>

          <div class="rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-5 space-y-3">
            <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-300">Formula</h2>
            <CustomGaugeFormulaBuilder v-model="form.inputs" />
          </div>

          <div v-if="saveError" class="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 p-3 text-sm text-red-600 dark:text-red-400">
            {{ saveError }}
          </div>

          <div class="flex items-center justify-between gap-3">
            <UButton variant="ghost" color="error" :loading="deleting" type="button" @click="confirmDelete">
              Delete
            </UButton>
            <div class="flex gap-3">
              <UButton variant="outline" color="neutral" type="button" @click="copyShareLink">Share</UButton>
              <UButton type="submit" :loading="saving" :disabled="!form.name || form.inputs.length === 0">Save</UButton>
            </div>
          </div>
        </form>
      </template>

      <div v-else class="mt-20 text-center text-gray-400">Gauge not found.</div>

      <!-- Share toast -->
      <div
        v-if="shareToast"
        class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 rounded-lg border border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-950/80 px-4 py-2.5 text-sm text-green-700 dark:text-green-300 shadow-lg"
      >
        Share link copied to clipboard
      </div>

    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import type { FormulaInput } from '~/components/gauge/CustomGaugeFormulaBuilder.vue'

const route = useRoute()
const router = useRouter()
const { getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public

const slug = computed(() => route.params.slug as string)

interface CustomGaugeDetail {
  id: string
  slug: string
  name: string
  description: string | null
  unit: string
  last_value_cfs: number | null
  last_value_at: string | null
  inputs: Array<{
    gauge_id: string
    gauge_name: string
    external_id: string
    source: string
    sign: 1 | -1
  }>
}

const gauge = ref<CustomGaugeDetail | null>(null)
const loading = ref(true)
const saving = ref(false)
const deleting = ref(false)
const saveError = ref<string | null>(null)
const shareToast = ref(false)

const form = ref({
  name: '',
  description: '',
  unit: 'cfs',
  inputs: [] as FormulaInput[],
})

async function load() {
  loading.value = true
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/me/custom-gauges/${slug.value}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) { gauge.value = null; return }
    const data: CustomGaugeDetail = await res.json()
    gauge.value = data
    form.value = {
      name:        data.name,
      description: data.description ?? '',
      unit:        data.unit,
      inputs:      (data.inputs ?? []).map(inp => ({
        gauge_id:    inp.gauge_id,
        gauge_name:  inp.gauge_name,
        external_id: inp.external_id,
        source:      inp.source,
        sign:        inp.sign,
      })),
    }
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!form.value.name || form.value.inputs.length === 0) return
  saving.value = true
  saveError.value = null
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/me/custom-gauges/${slug.value}`, {
      method:  'PATCH',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({
        name:        form.value.name,
        description: form.value.description || null,
        unit:        form.value.unit,
        inputs:      form.value.inputs.map(inp => ({ gauge_id: inp.gauge_id, sign: inp.sign })),
      }),
    })
    const data = await res.json()
    if (!res.ok) { saveError.value = data.error ?? 'Save failed'; return }
    if (data.slug && data.slug !== slug.value) {
      router.replace(`/my/gauges/${data.slug}`)
    }
    await load()
  } catch (e: any) {
    saveError.value = e.message
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!confirm(`Delete "${gauge.value?.name}"? This cannot be undone.`)) return
  deleting.value = true
  try {
    const token = await getToken()
    await fetch(`${apiBase}/api/v1/me/custom-gauges/${slug.value}`, {
      method:  'DELETE',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    router.push('/my/gauges')
  } finally {
    deleting.value = false
  }
}

async function copyShareLink() {
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/me/custom-gauges/${slug.value}/share`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) return
    const { token: shareToken } = await res.json()
    const url = `${window.location.origin}/import/gauge?token=${shareToken}`
    await navigator.clipboard.writeText(url)
    shareToast.value = true
    setTimeout(() => { shareToast.value = false }, 2500)
  } catch {}
}

onMounted(load)
</script>
