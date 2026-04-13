<template>
  <UModal v-model:open="open" :ui="{ width: 'max-w-lg' }">
    <template #header>
      <div class="flex items-center justify-between gap-2 w-full">
        <div>
          <h2 class="font-semibold text-base">Share your experience</h2>
          <p class="text-xs text-gray-400 mt-0.5">{{ reachName }}</p>
        </div>
        <button
          v-if="step === 'form'"
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          @click="step = 'pick'"
        >← Back</button>
      </div>
    </template>

    <template #body>

      <!-- Step 1: Destination picker -->
      <div v-if="step === 'pick'" class="space-y-3">

        <!-- Primary destinations -->
        <div class="grid grid-cols-3 gap-2">
          <button
            v-for="dest in primaryDests"
            :key="dest.id"
            class="flex flex-col items-center gap-1.5 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:bg-blue-50 dark:hover:bg-blue-950/30 hover:border-blue-300 dark:hover:border-blue-700 px-3 py-3 text-center transition-colors"
            @click="selectDest(dest.id)"
          >
            <span class="text-2xl">{{ dest.icon }}</span>
            <span class="text-xs font-medium text-gray-700 dark:text-gray-200 leading-tight">{{ dest.label }}</span>
            <span class="text-[10px] text-gray-400 leading-tight">{{ dest.sublabel }}</span>
          </button>
        </div>

        <!-- Coming-soon external destinations -->
        <div>
          <p class="text-[10px] text-gray-400 uppercase tracking-wide mb-2">Also share to (coming soon)</p>
          <div class="flex flex-wrap gap-2">
            <div
              v-for="ext in externalDests"
              :key="ext.id"
              class="flex items-center gap-1.5 rounded-lg border border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 px-3 py-1.5 opacity-50 cursor-not-allowed"
            >
              <span class="text-base">{{ ext.icon }}</span>
              <span class="text-xs text-gray-500">{{ ext.label }}</span>
            </div>
          </div>
        </div>

        <!-- h2oflows opt-in -->
        <label class="flex items-start gap-2.5 cursor-pointer pt-1">
          <input
            v-model="shareConsent"
            type="checkbox"
            class="mt-0.5 rounded accent-blue-600"
          />
          <div>
            <span class="text-xs font-medium text-gray-700 dark:text-gray-200">Share anonymized data with h2oflows</span>
            <p class="text-[11px] text-gray-400 mt-0.5">Helps improve flow recommendations · No personal data shared</p>
          </div>
        </label>

      </div>

      <!-- Step 2: Trip report form -->
      <div v-else-if="step === 'form' && selectedDest === 'trip_report'" class="space-y-4">

        <div>
          <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Title <span class="text-gray-400">(optional)</span></label>
          <input
            v-model="formTitle"
            type="text"
            placeholder="e.g. Spring run on high water"
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/40"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Notes</label>
          <textarea
            v-model="formBody"
            rows="4"
            placeholder="How was the run? Any hazards, lines, or beta worth sharing?"
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/40 resize-none"
          />
        </div>

        <div class="flex gap-3">
          <div class="flex-1">
            <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Flow impression</label>
            <div class="flex gap-1.5">
              <button
                v-for="opt in impressionOpts"
                :key="opt.value"
                :class="[
                  'flex-1 rounded-lg border py-1.5 text-xs font-medium transition-colors',
                  formImpression === opt.value
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/40 text-blue-600 dark:text-blue-400'
                    : 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:border-gray-300'
                ]"
                @click="formImpression = formImpression === opt.value ? null : opt.value"
              >{{ opt.label }}</button>
            </div>
          </div>

          <div class="w-40">
            <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Date</label>
            <input
              v-model="formDate"
              type="date"
              class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/40"
            />
          </div>
        </div>

        <!-- Current CFS -->
        <div v-if="currentCfs != null" class="flex items-center gap-2 rounded-lg bg-gray-50 dark:bg-gray-900 border border-gray-100 dark:border-gray-800 px-3 py-2">
          <svg class="w-3.5 h-3.5 text-gray-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
          <span class="text-xs text-gray-500">Will record current flow: <strong class="text-gray-700 dark:text-gray-200">{{ currentCfs.toLocaleString() }} cfs</strong>
            <span v-if="flowStatus" class="ml-1 text-gray-400">({{ flowStatusLabel(flowStatus) }})</span>
          </span>
        </div>

        <div class="flex items-center gap-2 pt-1">
          <label class="flex items-center gap-2 cursor-pointer flex-1">
            <input v-model="formPublish" type="checkbox" class="rounded accent-blue-600" />
            <span class="text-xs text-gray-600 dark:text-gray-400">Publish publicly</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="shareConsent" type="checkbox" class="rounded accent-blue-600" />
            <span class="text-xs text-gray-500">Share with h2oflows</span>
          </label>
        </div>

        <div v-if="submitError" class="rounded-lg bg-red-50 dark:bg-red-950/40 border border-red-200 dark:border-red-800 px-3 py-2 text-xs text-red-600 dark:text-red-400">
          {{ submitError }}
        </div>

        <div class="flex gap-2 pt-1">
          <UButton
            variant="outline" color="neutral" size="sm" class="flex-1"
            @click="open = false"
          >Cancel</UButton>
          <UButton
            size="sm" class="flex-1"
            :loading="submitting"
            :disabled="submitting"
            @click="submitTripReport"
          >Submit trip report</UButton>
        </div>
      </div>

      <!-- Step 2: Quick form (flow update / hazard alert) -->
      <div v-else-if="step === 'form' && selectedDest !== 'trip_report'" class="space-y-4">

        <div>
          <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
            {{ selectedDest === 'hazard_alert' ? 'Describe the hazard' : 'Note' }} <span class="text-gray-400">(optional)</span>
          </label>
          <textarea
            v-model="formBody"
            rows="3"
            :placeholder="selectedDest === 'hazard_alert' ? 'What hazard did you observe? Location, severity…' : 'Any notes about current conditions?'"
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/40 resize-none"
          />
        </div>

        <div v-if="selectedDest === 'flow_update'" class="flex gap-3">
          <div class="flex-1">
            <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Flow impression</label>
            <div class="flex gap-1.5">
              <button
                v-for="opt in impressionOpts"
                :key="opt.value"
                :class="[
                  'flex-1 rounded-lg border py-1.5 text-xs font-medium transition-colors',
                  formImpression === opt.value
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/40 text-blue-600 dark:text-blue-400'
                    : 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:border-gray-300'
                ]"
                @click="formImpression = formImpression === opt.value ? null : opt.value"
              >{{ opt.label }}</button>
            </div>
          </div>
          <div class="w-40">
            <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Date</label>
            <input v-model="formDate" type="date" class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/40" />
          </div>
        </div>

        <div v-if="submitError" class="rounded-lg bg-red-50 dark:bg-red-950/40 border border-red-200 dark:border-red-800 px-3 py-2 text-xs text-red-600 dark:text-red-400">
          {{ submitError }}
        </div>

        <div class="flex gap-2 pt-1">
          <UButton variant="outline" color="neutral" size="sm" class="flex-1" @click="open = false">Cancel</UButton>
          <UButton size="sm" class="flex-1" :loading="submitting" :disabled="submitting" @click="submitContribution">
            {{ selectedDest === 'hazard_alert' ? 'Report hazard' : 'Submit update' }}
          </UButton>
        </div>
      </div>

    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{
  reachSlug:  string
  reachName:  string
  currentCfs?: number | null
  flowStatus?: string | null
}>()

const open = defineModel<boolean>('open', { default: false })

const config = useRuntimeConfig()
const toast  = useToast()

// ---- State ------------------------------------------------------------------

const step         = ref<'pick' | 'form'>('pick')
const selectedDest = ref<string | null>(null)

const shareConsent  = ref(false)
const formTitle     = ref('')
const formBody      = ref('')
const formImpression = ref<string | null>(null)
const formPublish   = ref(true)
const formDate      = ref(new Date().toISOString().slice(0, 10))

const submitting  = ref(false)
const submitError = ref<string | null>(null)

// ---- Destination options ----------------------------------------------------

const primaryDests = [
  { id: 'trip_report',  icon: '🚣', label: 'Trip Report',   sublabel: 'Full run write-up' },
  { id: 'flow_update',  icon: '💧', label: 'Flow Update',   sublabel: 'Quick conditions' },
  { id: 'hazard_alert', icon: '⚠️', label: 'Hazard Alert',  sublabel: 'Safety issue' },
]

const externalDests = [
  { id: 'aw',        icon: '🌊', label: 'AW' },
  { id: 'instagram', icon: '📸', label: 'Instagram' },
  { id: 'youtube',   icon: '▶️', label: 'YouTube' },
]

const impressionOpts = [
  { value: 'too_low', label: 'Too Low' },
  { value: 'good',    label: 'Good' },
  { value: 'high',    label: 'High' },
]

// ---- Actions ----------------------------------------------------------------

function selectDest(id: string) {
  selectedDest.value = id
  submitError.value  = null
  step.value = 'form'
}

async function submitTripReport() {
  submitting.value  = true
  submitError.value = null
  try {
    const body: Record<string, any> = {
      body:                  formBody.value || undefined,
      title:                 formTitle.value || undefined,
      flow_impression:       formImpression.value ?? undefined,
      observed_at:           formDate.value ? new Date(formDate.value).toISOString() : undefined,
      share_consent_h2oflows: shareConsent.value,
      published:             formPublish.value,
    }
    const res = await fetch(
      `${config.public.apiBase}/api/v1/reaches/${props.reachSlug}/trip-reports`,
      { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
    )
    const json = await res.json()
    if (!res.ok) throw new Error(json.error ?? `Server error ${res.status}`)

    open.value = false
    toast.add({ title: 'Trip report submitted!', description: formPublish.value ? `Public page: /trips/${json.public_slug}` : 'Saved privately.', color: 'success' })
    resetForm()
  } catch (err: any) {
    submitError.value = err?.message ?? 'Something went wrong'
  } finally {
    submitting.value = false
  }
}

async function submitContribution() {
  submitting.value  = true
  submitError.value = null
  try {
    const typeMap: Record<string, string> = {
      flow_update:  'flow_update',
      hazard_alert: 'hazard_alert',
    }
    const body: Record<string, any> = {
      contribution_type:     typeMap[selectedDest.value!] ?? 'general',
      body:                  formBody.value || undefined,
      flow_impression:       formImpression.value ?? undefined,
      observed_at:           formDate.value ? new Date(formDate.value).toISOString() : undefined,
      share_consent_h2oflows: shareConsent.value,
    }
    const res = await fetch(
      `${config.public.apiBase}/api/v1/reaches/${props.reachSlug}/contributions`,
      { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
    )
    const json = await res.json()
    if (!res.ok) throw new Error(json.error ?? `Server error ${res.status}`)

    open.value = false
    toast.add({ title: selectedDest.value === 'hazard_alert' ? 'Hazard reported' : 'Flow update submitted', color: 'success' })
    resetForm()
  } catch (err: any) {
    submitError.value = err?.message ?? 'Something went wrong'
  } finally {
    submitting.value = false
  }
}

function resetForm() {
  step.value          = 'pick'
  selectedDest.value  = null
  formTitle.value     = ''
  formBody.value      = ''
  formImpression.value = null
  formPublish.value   = true
  formDate.value      = new Date().toISOString().slice(0, 10)
  submitError.value   = null
}

function flowStatusLabel(status: string): string {
  switch (status) {
    case 'runnable': return 'Runnable'
    case 'caution':  return 'Below Recommended'
    case 'low':      return 'Too Low'
    case 'flood':    return 'Above Recommended'
    default:         return status
  }
}

// Reset form when modal closes
watch(open, (val) => { if (!val) resetForm() })
</script>
