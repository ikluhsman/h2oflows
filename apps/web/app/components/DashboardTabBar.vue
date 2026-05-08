<template>
  <div class="sticky top-[51px] z-20 bg-white/95 dark:bg-neutral-950/95 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800">
    <div class="max-w-5xl mx-auto px-4">
      <div class="flex items-center gap-0.5 overflow-x-auto scrollbar-none py-0.5">

        <!-- Tabs -->
        <div
          v-for="db in dashboards"
          :key="db.id"
          class="group relative flex items-center shrink-0"
        >
          <button
            class="flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-t transition-colors whitespace-nowrap"
            :class="db.id === activeDashboardId
              ? 'text-primary-600 dark:text-primary-400 border-b-2 border-primary-500 -mb-px'
              : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200'"
            @click="$emit('select', db.id)"
          >
            {{ db.name }}
          </button>

          <!-- Three-dot menu per tab -->
          <div class="relative" v-if="dashboards.length > 1 || db.id === activeDashboardId">
            <button
              class="p-1 rounded opacity-0 group-hover:opacity-100 transition-opacity text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200"
              :class="{ '!opacity-100': menuOpenId === db.id }"
              @click.stop="toggleMenu(db.id)"
            >
              <svg class="w-3 h-3" viewBox="0 0 16 16" fill="currentColor">
                <circle cx="8" cy="3" r="1.2"/><circle cx="8" cy="8" r="1.2"/><circle cx="8" cy="13" r="1.2"/>
              </svg>
            </button>
            <div
              v-if="menuOpenId === db.id"
              class="absolute left-0 top-full mt-1 z-50 w-36 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 shadow-lg overflow-hidden"
            >
              <button
                class="w-full text-left px-3 py-2 text-sm hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
                @click="startRename(db)"
              >Rename</button>
              <button
                v-if="dashboards.length > 1"
                class="w-full text-left px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
                @click="$emit('delete', db.slug); menuOpenId = null"
              >Delete</button>
            </div>
          </div>
        </div>

        <!-- New dashboard button -->
        <button
          class="shrink-0 flex items-center gap-1 px-2 py-2 text-sm text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
          title="New dashboard"
          @click="$emit('new')"
        >
          <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd"/>
          </svg>
        </button>
      </div>
    </div>
  </div>

  <!-- Rename modal -->
  <Teleport to="body">
    <div v-if="renaming" class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="renaming = null">
      <div class="absolute inset-0 bg-black/40" @click="renaming = null" />
      <div class="relative w-full max-w-xs bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-700 shadow-xl p-5 space-y-4">
        <h3 class="text-sm font-semibold">Rename dashboard</h3>
        <input
          ref="renameInput"
          v-model="renameName"
          type="text"
          class="w-full rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 px-3 py-2 text-sm"
          @keydown.enter="submitRename"
          @keydown.esc="renaming = null"
        />
        <div class="flex gap-2 justify-end">
          <button class="px-3 py-1.5 text-sm rounded-lg border border-neutral-200 dark:border-neutral-700" @click="renaming = null">Cancel</button>
          <button class="px-3 py-1.5 text-sm rounded-lg bg-primary-600 text-white hover:bg-primary-700" @click="submitRename">Save</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { Dashboard } from '~/composables/useDashboards'

const props = defineProps<{
  dashboards: Dashboard[]
  activeDashboardId: string | null
}>()

const emit = defineEmits<{
  select: [id: string]
  new: []
  delete: [slug: string]
  rename: [slug: string, name: string]
}>()

const menuOpenId = ref<string | null>(null)
const renaming = ref<Dashboard | null>(null)
const renameName = ref('')
const renameInput = ref<HTMLInputElement | null>(null)

function toggleMenu(id: string) {
  menuOpenId.value = menuOpenId.value === id ? null : id
}

function startRename(db: Dashboard) {
  renaming.value = db
  renameName.value = db.name
  menuOpenId.value = null
  nextTick(() => renameInput.value?.focus())
}

function submitRename() {
  if (!renaming.value || !renameName.value.trim()) return
  emit('rename', renaming.value.slug, renameName.value.trim())
  renaming.value = null
}

// Close menu on outside click
if (import.meta.client) {
  document.addEventListener('click', () => { menuOpenId.value = null })
}
</script>
