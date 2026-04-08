<template>
  <Transition
    enter-active-class="transition duration-300 ease-out"
    enter-from-class="translate-y-full opacity-0"
    enter-to-class="translate-y-0 opacity-100"
    leave-active-class="transition duration-200 ease-in"
    leave-from-class="translate-y-0 opacity-100"
    leave-to-class="translate-y-full opacity-0"
  >
    <div
      v-if="visible"
      class="fixed bottom-3 left-3 right-3 sm:left-auto sm:right-4 sm:bottom-4 sm:max-w-md z-30 rounded-lg border border-amber-300 dark:border-amber-700 bg-amber-50 dark:bg-amber-950 shadow-lg p-4"
    >
      <div class="flex items-start gap-3">
        <div class="text-amber-600 dark:text-amber-400 text-lg shrink-0" aria-hidden="true">⚠</div>
        <div class="flex-1 text-sm text-amber-900 dark:text-amber-100">
          <p class="font-semibold mb-1">Preview release — verify before you paddle</p>
          <p class="text-xs leading-relaxed">
            H2OFlows is in early development. Reach descriptions, rapid lists, access points, and
            flow ranges are largely AI-generated drafts and may be incomplete or wrong. Always
            cross-check against
            <a
              href="https://www.americanwhitewater.org/"
              target="_blank"
              rel="noopener"
              class="underline hover:no-underline"
            >American Whitewater</a>,
            local guidebooks, and people who have actually run the river. Real-time gauge data
            from USGS and Colorado DWR is unmodified.
          </p>
        </div>
        <button
          type="button"
          class="text-amber-600 dark:text-amber-400 hover:text-amber-800 dark:hover:text-amber-200 text-xs font-medium shrink-0"
          @click="dismiss"
        >Got it</button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const STORAGE_KEY = 'h2oflow:disclaimer-dismissed-v1'
const visible = ref(false)

onMounted(() => {
  // Show on every fresh visit unless explicitly dismissed.
  if (typeof window === 'undefined') return
  try {
    if (window.localStorage.getItem(STORAGE_KEY) !== 'true') {
      visible.value = true
    }
  } catch {
    visible.value = true
  }
})

function dismiss() {
  visible.value = false
  try {
    window.localStorage.setItem(STORAGE_KEY, 'true')
  } catch {
    // localStorage blocked — banner just won't persist its dismissed state.
  }
}
</script>
