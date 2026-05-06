<template>
  <div class="space-y-3">
    <!-- Input rows -->
    <div v-if="modelValue.length" class="space-y-2">
      <div
        v-for="(inp, i) in modelValue"
        :key="inp.gauge_id"
        class="flex items-center gap-2 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2"
      >
        <!-- Sign toggle -->
        <button
          class="shrink-0 w-7 h-7 rounded font-bold text-sm flex items-center justify-center transition-colors"
          :class="inp.sign === 1
            ? 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-400'
            : 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-400'"
          type="button"
          :title="inp.sign === 1 ? 'Click to subtract' : 'Click to add'"
          @click="toggleSign(i)"
        >
          {{ inp.sign === 1 ? '+' : '−' }}
        </button>

        <!-- Gauge info -->
        <div class="flex-1 min-w-0">
          <p class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ inp.gauge_name }}</p>
          <p class="text-xs text-gray-400 truncate">{{ inp.external_id }} · {{ inp.source }}</p>
        </div>

        <!-- Remove -->
        <button
          class="shrink-0 p-1 rounded text-gray-300 dark:text-gray-600 hover:text-red-500 dark:hover:text-red-400 transition-colors"
          type="button"
          @click="removeAt(i)"
        >
          <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Formula preview -->
    <p v-if="modelValue.length" class="text-xs text-gray-400 font-mono px-1">
      {{ formulaPreview }}
    </p>

    <!-- Search -->
    <div class="relative" ref="searchWrap">
      <div class="flex gap-2">
        <UInput
          v-model="query"
          placeholder="Search gauges to add…"
          icon="i-heroicons-magnifying-glass"
          class="flex-1"
          @input="onInput"
          @focus="showDropdown = results.length > 0"
          @keydown.escape="showDropdown = false"
        />
      </div>

      <!-- Results dropdown -->
      <div
        v-if="showDropdown && results.length"
        class="absolute z-50 left-0 right-0 mt-1 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-lg max-h-56 overflow-y-auto"
      >
        <button
          v-for="g in results"
          :key="g.id"
          type="button"
          class="w-full text-left px-3 py-2 hover:bg-blue-50 dark:hover:bg-blue-950/30 transition-colors"
          @click="addGauge(g)"
        >
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ g.name }}</p>
          <p class="text-xs text-gray-400">{{ g.external_id }} · {{ g.source }}</p>
        </button>
      </div>
    </div>

    <p v-if="!modelValue.length" class="text-xs text-gray-400">
      Search and add at least one gauge. Use the +/− toggle to add or subtract each reading.
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onClickOutside } from '@vueuse/core'

export interface FormulaInput {
  gauge_id: string
  gauge_name: string
  external_id: string
  source: string
  sign: 1 | -1
}

const props = defineProps<{ modelValue: FormulaInput[] }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: FormulaInput[]): void }>()

const { apiBase } = useRuntimeConfig().public
const query = ref('')
const loading = ref(false)
const results = ref<{ id: string; name: string; external_id: string; source: string }[]>([])
const showDropdown = ref(false)
const searchWrap = ref<HTMLElement | null>(null)

onClickOutside(searchWrap, () => { showDropdown.value = false })

let debounceTimer: ReturnType<typeof setTimeout>

function onInput() {
  clearTimeout(debounceTimer)
  if (query.value.length < 2) {
    results.value = []
    showDropdown.value = false
    return
  }
  debounceTimer = setTimeout(search, 300)
}

async function search() {
  loading.value = true
  try {
    const res = await fetch(`${apiBase}/api/v1/gauges/search?q=${encodeURIComponent(query.value)}&limit=15`)
    if (!res.ok) return
    const data = await res.json()
    results.value = (data.features ?? []).map((f: any) => ({
      id:          f.properties.id,
      name:        f.properties.name,
      external_id: f.properties.external_id,
      source:      f.properties.source,
    }))
    showDropdown.value = results.value.length > 0
  } catch {
    results.value = []
  } finally {
    loading.value = false
  }
}

function addGauge(g: { id: string; name: string; external_id: string; source: string }) {
  if (props.modelValue.some(inp => inp.gauge_id === g.id)) return
  emit('update:modelValue', [
    ...props.modelValue,
    { gauge_id: g.id, gauge_name: g.name, external_id: g.external_id, source: g.source, sign: 1 },
  ])
  query.value = ''
  results.value = []
  showDropdown.value = false
}

function removeAt(i: number) {
  const next = [...props.modelValue]
  next.splice(i, 1)
  emit('update:modelValue', next)
}

function toggleSign(i: number) {
  const next = [...props.modelValue]
  next[i] = { ...next[i], sign: next[i].sign === 1 ? -1 : 1 }
  emit('update:modelValue', next)
}

const formulaPreview = computed(() => {
  return props.modelValue.map((inp, i) => {
    const prefix = i === 0 ? (inp.sign === -1 ? '−' : '') : (inp.sign === 1 ? ' + ' : ' − ')
    return prefix + inp.gauge_name
  }).join('')
})
</script>
