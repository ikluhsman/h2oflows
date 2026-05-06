<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-2xl mx-auto px-4 py-6 space-y-5">
      <div class="flex items-center gap-3">
        <NuxtLink to="/my/gauges" class="text-sm text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
          ← Custom Gauges
        </NuxtLink>
      </div>

      <h1 class="text-xl font-bold text-gray-900 dark:text-white">New Custom Gauge</h1>

      <form class="space-y-5" @submit.prevent="submit">

        <div class="rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-5 space-y-4">
          <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-300">Details</h2>

          <UFormField label="Name" required>
            <UInput v-model="form.name" placeholder="e.g. Upper Animas Total" class="w-full" required />
          </UFormField>

          <UFormField label="Description">
            <UInput v-model="form.description" placeholder="Optional description" class="w-full" />
          </UFormField>

          <UFormField label="Unit">
            <USelect v-model="form.unit" :items="['cfs', 'ft', 'acre-ft']" class="w-40" />
          </UFormField>
        </div>

        <div class="rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-5 space-y-3">
          <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-300">Formula</h2>
          <p class="text-xs text-gray-400">Add gauges and set each one to + (add) or − (subtract).</p>
          <CustomGaugeFormulaBuilder v-model="form.inputs" />
        </div>

        <div v-if="saveError" class="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 p-3 text-sm text-red-600 dark:text-red-400">
          {{ saveError }}
        </div>

        <div class="flex justify-end gap-3">
          <UButton variant="ghost" color="neutral" @click="router.push('/my/gauges')">Cancel</UButton>
          <UButton type="submit" :loading="saving" :disabled="!form.name || form.inputs.length === 0">
            Create gauge
          </UButton>
        </div>

      </form>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { FormulaInput } from '~/components/gauge/CustomGaugeFormulaBuilder.vue'

const { getToken } = useAuth()
const { apiBase } = useRuntimeConfig().public
const router = useRouter()

const form = ref({
  name: '',
  description: '',
  unit: 'cfs',
  inputs: [] as FormulaInput[],
})

const saving = ref(false)
const saveError = ref<string | null>(null)

async function submit() {
  if (!form.value.name || form.value.inputs.length === 0) return
  saving.value = true
  saveError.value = null
  try {
    const token = await getToken()
    const res = await fetch(`${apiBase}/api/v1/me/custom-gauges`, {
      method:  'POST',
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
    router.push(`/my/gauges/${data.slug}`)
  } catch (e: any) {
    saveError.value = e.message
  } finally {
    saving.value = false
  }
}
</script>
