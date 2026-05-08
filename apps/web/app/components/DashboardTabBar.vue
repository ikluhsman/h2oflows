<template>
  <div class="sticky top-[55px] z-20 bg-white/95 dark:bg-neutral-950/95 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800">
    <div class="max-w-5xl mx-auto px-4">
      <div ref="scrollerRef" class="flex items-center gap-0.5 overflow-x-auto scrollbar-none py-0.5">

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

          <!-- Three-dot menu trigger per tab. Visible always on mobile (no hover),
               and hover-revealed on desktop. -->
          <button
            v-if="dashboards.length > 1 || db.id === activeDashboardId"
            :ref="(el) => setMenuButtonRef(db.id, el as HTMLElement | null)"
            class="p-1 rounded transition-opacity text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200 opacity-100 sm:opacity-0 sm:group-hover:opacity-100"
            :class="{ '!opacity-100': menuOpenId === db.id }"
            @click.stop="toggleMenu(db.id)"
          >
            <svg class="w-3 h-3" viewBox="0 0 16 16" fill="currentColor">
              <circle cx="8" cy="3" r="1.2"/><circle cx="8" cy="8" r="1.2"/><circle cx="8" cy="13" r="1.2"/>
            </svg>
          </button>
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

  <!-- Floating menu — teleported to body so it escapes the overflow-x-auto container -->
  <Teleport to="body">
    <div
      v-if="menuOpenId && menuPos"
      :style="{ position: 'fixed', top: `${menuPos.top}px`, left: `${menuPos.left}px` }"
      class="z-50 w-36 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 shadow-lg overflow-hidden"
      @click.stop
    >
      <button
        class="w-full text-left px-3 py-2 text-sm hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
        @click="startRenameById(menuOpenId)"
      >Rename</button>
      <button
        v-if="dashboards.length > 1"
        class="w-full text-left px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
        @click="confirmDeleteById(menuOpenId)"
      >Delete</button>
    </div>
  </Teleport>

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

  <!-- Delete confirmation modal -->
  <Teleport to="body">
    <div v-if="deleting" class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="deleting = null">
      <div class="absolute inset-0 bg-black/40" @click="deleting = null" />
      <div class="relative w-full max-w-xs bg-white dark:bg-neutral-900 rounded-xl border border-neutral-200 dark:border-neutral-700 shadow-xl p-5 space-y-3">
        <h3 class="text-sm font-semibold">Delete this dashboard?</h3>
        <p class="text-sm text-neutral-500 dark:text-neutral-400">
          "{{ deleting.name }}" and all its watchlisted gauges will be removed. This cannot be undone.
        </p>
        <div class="flex gap-2 justify-end pt-1">
          <button class="px-3 py-1.5 text-sm rounded-lg border border-neutral-200 dark:border-neutral-700" @click="deleting = null">Cancel</button>
          <button class="px-3 py-1.5 text-sm rounded-lg bg-red-600 text-white hover:bg-red-700" @click="confirmDelete">Delete</button>
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
const menuPos = ref<{ top: number; left: number } | null>(null)
const menuButtonRefs = new Map<string, HTMLElement>()
const renaming = ref<Dashboard | null>(null)
const renameName = ref('')
const renameInput = ref<HTMLInputElement | null>(null)
const deleting = ref<Dashboard | null>(null)
const scrollerRef = ref<HTMLElement | null>(null)

function setMenuButtonRef(id: string, el: HTMLElement | null) {
  if (el) menuButtonRefs.set(id, el)
  else menuButtonRefs.delete(id)
}

function toggleMenu(id: string) {
  if (menuOpenId.value === id) {
    menuOpenId.value = null
    menuPos.value = null
    return
  }
  const btn = menuButtonRefs.get(id)
  if (!btn) return
  const rect = btn.getBoundingClientRect()
  menuPos.value = { top: rect.bottom + 4, left: rect.left }
  menuOpenId.value = id
}

function closeMenu() {
  menuOpenId.value = null
  menuPos.value = null
}

function startRenameById(id: string) {
  const db = props.dashboards.find(d => d.id === id)
  if (!db) return
  renaming.value = db
  renameName.value = db.name
  closeMenu()
  nextTick(() => renameInput.value?.focus())
}

function submitRename() {
  if (!renaming.value || !renameName.value.trim()) return
  emit('rename', renaming.value.slug, renameName.value.trim())
  renaming.value = null
}

function confirmDeleteById(id: string) {
  const db = props.dashboards.find(d => d.id === id)
  if (!db) return
  deleting.value = db
  closeMenu()
}

function confirmDelete() {
  if (!deleting.value) return
  emit('delete', deleting.value.slug)
  deleting.value = null
}

// Close menu on outside click or scroll
if (import.meta.client) {
  document.addEventListener('click', closeMenu)
  // Reposition closed if user scrolls the tab strip — simpler than tracking scroll
  onMounted(() => {
    scrollerRef.value?.addEventListener('scroll', closeMenu)
  })
  onUnmounted(() => {
    document.removeEventListener('click', closeMenu)
    scrollerRef.value?.removeEventListener('scroll', closeMenu)
  })
}
</script>
