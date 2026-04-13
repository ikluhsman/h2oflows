<template>
  <header class="sticky top-0 z-20 border-b border-gray-200 dark:border-gray-800 bg-white/90 dark:bg-gray-950/90 backdrop-blur-sm">
    <div class="max-w-5xl mx-auto px-4 py-2.5 flex items-center gap-2">

      <!-- Logo -->
      <NuxtLink to="/" class="flex items-center gap-1.5 shrink-0 mr-1">
        <svg class="w-5 h-5 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M2 12c2-4 4-6 6-6s4 6 6 6 4-6 6-6" stroke-linecap="round"/>
        </svg>
        <span class="text-sm font-bold tracking-tight hidden sm:inline">H2OFlows</span>
      </NuxtLink>

      <!-- Breadcrumb / page-level content injected by each page -->
      <div class="flex items-center gap-2 min-w-0 flex-1">
        <slot />
      </div>

      <!-- Page-level action buttons -->
      <slot name="actions" />

      <!-- Global Ask button -->
      <button
        class="shrink-0 hidden sm:flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-gray-200 dark:border-gray-700 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors"
        title="Ask anything"
        @click="askOpen = true"
      >
        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
        </svg>
        <span class="text-xs">Ask anything…</span>
      </button>

      <!-- Dashboard shortcut — always visible -->
      <NuxtLink
        to="/dashboard"
        class="shrink-0 flex items-center gap-1 p-1.5 rounded-md transition-colors"
        :class="route.path === '/dashboard'
          ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950/50'
          : 'text-gray-500 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 hover:bg-gray-50 dark:hover:bg-gray-900'"
        title="Flow Dashboard"
      >
        <svg class="w-4.5 h-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="4" rx="1"/><rect x="14" y="10" width="7" height="11" rx="1"/><rect x="3" y="13" width="7" height="8" rx="1"/>
        </svg>
        <span class="hidden sm:inline text-xs font-medium">Dashboard</span>
      </NuxtLink>

      <!-- Map shortcut — always visible -->
      <NuxtLink
        to="/map"
        class="shrink-0 flex items-center gap-1 p-1.5 rounded-md transition-colors"
        :class="route.path === '/map'
          ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950/50'
          : 'text-gray-500 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 hover:bg-gray-50 dark:hover:bg-gray-900'"
        title="Interactive Map"
      >
        <svg class="w-4.5 h-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="3 6 9 3 15 6 21 3 21 18 15 21 9 18 3 21"/><line x1="9" y1="3" x2="9" y2="18"/><line x1="15" y1="6" x2="15" y2="21"/>
        </svg>
        <span class="hidden sm:inline text-xs font-medium">Map</span>
      </NuxtLink>

      <!-- Hamburger — mobile only -->
      <button
        class="sm:hidden shrink-0 p-1.5 rounded-md text-gray-500 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
        aria-label="Open menu"
        @click="menuOpen = !menuOpen"
      >
        <svg v-if="!menuOpen" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/>
        </svg>
        <svg v-else class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>

      <!-- User avatar — far right -->
      <ClientOnly>
        <div class="relative shrink-0" data-user-menu>
          <button
            class="flex items-center justify-center w-7 h-7 rounded-full transition-colors"
            :class="isAuthenticated
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-600 dark:text-blue-400 hover:bg-blue-200 dark:hover:bg-blue-900'
              : 'bg-gray-100 dark:bg-gray-800 text-gray-400 dark:text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-700'"
            :title="isAuthenticated ? `Signed in as ${user?.email ?? user?.user_metadata?.user_name ?? 'you'}` : 'Sign in'"
            @click="userMenuOpen = !userMenuOpen"
          >
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="8" r="4"/><path d="M20 21a8 8 0 0 0-16 0"/>
            </svg>
          </button>
          <div
            v-if="userMenuOpen"
            class="absolute right-0 top-full mt-1 w-44 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg py-1 z-30"
          >
            <template v-if="isAuthenticated">
              <p class="px-3 py-1.5 text-xs text-gray-400 truncate">{{ user?.email ?? user?.user_metadata?.user_name }}</p>
              <div class="border-t border-gray-100 dark:border-gray-800" />
            </template>
            <button
              class="w-full text-left px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors flex items-center gap-2"
              @click="toggleColorMode"
            >
              <svg v-if="colorMode.value === 'dark'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
              <svg v-else class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
              {{ colorMode.value === 'dark' ? 'Light mode' : 'Dark mode' }}
            </button>
            <div class="border-t border-gray-100 dark:border-gray-800" />
            <template v-if="isAuthenticated">
              <button
                class="w-full text-left px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                @click="userMenuOpen = false; handleSignOut()"
              >Sign out</button>
            </template>
            <template v-else>
              <NuxtLink
                to="/login"
                class="block px-3 py-1.5 text-sm font-medium text-blue-600 dark:text-blue-400 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                @click="userMenuOpen = false"
              >Sign in</NuxtLink>
            </template>
          </div>
        </div>
      </ClientOnly>
    </div>

    <!-- Mobile menu dropdown -->
    <div v-if="menuOpen" class="sm:hidden border-t border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-950 px-4 py-3 flex flex-col gap-1">
      <!-- Ask — mobile -->
      <button
        class="text-left px-3 py-2 rounded-md text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors flex items-center gap-2"
        @click="menuOpen = false; askOpen = true"
      >
        <svg class="w-4 h-4 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
        </svg>
        Ask anything
      </button>
      <div class="border-t border-gray-100 dark:border-gray-800 mt-1 pt-2">
        <ClientOnly>
          <button
            v-if="isAuthenticated"
            class="w-full text-left px-3 py-2 rounded-md text-sm text-gray-500 hover:text-gray-900 dark:hover:text-white hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors"
            @click="handleSignOut"
          >Sign out</button>
          <NuxtLink
            v-else
            to="/login"
            class="block px-3 py-2 rounded-md text-sm font-medium text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-950 transition-colors"
            @click="menuOpen = false"
          >Sign in</NuxtLink>
        </ClientOnly>
      </div>
    </div>
  </header>

  <!-- Global Ask modal (Teleport so it's above everything) -->
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="askOpen"
        class="fixed inset-0 z-50 flex items-start justify-center pt-[10vh] px-4 bg-black/20 backdrop-blur-sm"
        @click.self="closeAsk"
      >
        <div class="w-full max-w-xl bg-white dark:bg-gray-900 rounded-2xl shadow-2xl border border-gray-200 dark:border-gray-800 overflow-hidden">
          <form class="flex items-center gap-2 px-4 py-3 border-b border-gray-100 dark:border-gray-800" @submit.prevent="askQuestion">
            <svg class="w-4 h-4 text-gray-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
            </svg>
            <input
              ref="askInputRef"
              v-model="askQuery"
              type="text"
              placeholder='Ask anything — e.g. "Browns Canyon at 800 cfs?"'
              class="flex-1 bg-transparent text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none"
              :disabled="asking"
            />
            <button
              v-if="askQuery"
              type="button"
              class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              @click="askQuery = ''; askResult = null"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18M6 6l12 12"/></svg>
            </button>
            <button
              type="submit"
              :disabled="asking || !askQuery.trim()"
              class="shrink-0 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-40 text-white text-xs font-semibold transition-colors"
            >
              <span v-if="asking" class="flex items-center gap-1">
                <span class="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin"/>
              </span>
              <span v-else>Ask</span>
            </button>
          </form>

          <div v-if="askResult" class="px-4 py-4 space-y-3 max-h-96 overflow-y-auto">
            <div
              v-for="result in (askResult.results ?? [])"
              :key="result.reach_slug"
              class="rounded-lg border border-gray-100 dark:border-gray-800 p-3 space-y-1"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs font-semibold uppercase tracking-wide text-blue-500">{{ result.reach_name }}</span>
                <NuxtLink
                  :to="`/reaches/${result.reach_slug}`"
                  class="text-xs text-blue-600 dark:text-blue-400 hover:underline font-medium shrink-0"
                  @click="closeAsk"
                >View reach →</NuxtLink>
              </div>
              <p class="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{{ result.answer }}</p>
            </div>
            <p v-if="!askResult.results?.length && askResult.answer" class="text-sm text-gray-500 leading-relaxed">{{ askResult.answer }}</p>
          </div>
          <p v-else-if="!asking && !askResult" class="px-4 py-3 text-xs text-gray-400">
            Try: "What's Foxton like at 300 cfs?" or "Best beginner runs near Denver"
          </p>
          <p v-if="askError" class="px-4 py-3 text-sm text-red-400">{{ askError }}</p>

          <div class="px-4 py-2.5 border-t border-gray-100 dark:border-gray-800 flex justify-end">
            <button class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" @click="closeAsk">Close</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, nextTick, watch, onMounted, onUnmounted } from 'vue'

const { user, isAuthenticated, signOut } = useAuth()
const router = useRouter()
const route = useRoute()
const colorMode = useColorMode()
const menuOpen = ref(false)
const userMenuOpen = ref(false)

function toggleColorMode() {
  colorMode.preference = colorMode.value === 'dark' ? 'light' : 'dark'
}

const { apiBase } = useRuntimeConfig().public

// Close menus on route change
watch(() => route.path, () => { menuOpen.value = false; userMenuOpen.value = false })

function onDocClick(e: MouseEvent) {
  if (userMenuOpen.value && !(e.target as HTMLElement).closest('[data-user-menu]')) {
    userMenuOpen.value = false
  }
}
onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))

async function handleSignOut() {
  menuOpen.value = false
  await signOut()
  router.push('/')
}

// ── Global Ask ────────────────────────────────────────────────────────────────
const askOpen     = ref(false)
const askInputRef = ref<HTMLInputElement>()
const askQuery    = ref('')
const asking      = ref(false)
const askError    = ref('')
const askResult   = ref<{ results?: { answer: string; reach_slug: string; reach_name: string }[]; answer?: string } | null>(null)

watch(askOpen, async (open) => {
  if (open) {
    askQuery.value  = ''
    askResult.value = null
    askError.value  = ''
    await nextTick()
    askInputRef.value?.focus()
  }
})

function closeAsk() { askOpen.value = false }

async function askQuestion() {
  const q = askQuery.value.trim()
  if (!q) return
  asking.value    = true
  askError.value  = ''
  askResult.value = null
  try {
    const res = await fetch(`${apiBase}/api/v1/ask`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question: q }),
    })
    if (!res.ok) throw new Error(`${res.status}`)
    askResult.value = await res.json()
  } catch {
    askError.value = 'Something went wrong. Try again.'
  } finally {
    asking.value = false
  }
}
</script>
