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

      <!-- Primary nav — desktop -->
      <nav class="hidden sm:flex items-center gap-0.5">
        <NuxtLink
          v-for="link in navLinks" :key="link.to" :to="link.to"
          class="px-2.5 py-1.5 rounded-md text-sm font-medium transition-colors"
          :class="$route.path.startsWith(link.to) && link.to !== '/'
            ? 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-white'
            : 'text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-50 dark:hover:bg-gray-900'"
        >{{ link.label }}</NuxtLink>
      </nav>

      <!-- Breadcrumb / page-level content injected by each page -->
      <div class="flex items-center gap-2 min-w-0 flex-1">
        <slot />
      </div>

      <!-- Page-level action buttons -->
      <slot name="actions" />

      <!-- Auth — desktop only, ClientOnly prevents SSR/client mismatch -->
      <ClientOnly>
        <template v-if="isAuthenticated">
          <button
            class="hidden sm:inline shrink-0 text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
            :title="`Signed in as ${user?.email ?? user?.user_metadata?.user_name ?? 'you'}`"
            @click="handleSignOut"
          >Sign out</button>
        </template>
        <template v-else>
          <NuxtLink
            to="/login"
            class="hidden sm:inline shrink-0 text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors"
          >Sign in</NuxtLink>
        </template>
      </ClientOnly>

      <UColorModeButton size="sm" color="neutral" variant="ghost" class="shrink-0" />

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
    </div>

    <!-- Mobile menu dropdown -->
    <div v-if="menuOpen" class="sm:hidden border-t border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-950 px-4 py-3 flex flex-col gap-1">
      <NuxtLink
        v-for="link in navLinks" :key="link.to" :to="link.to"
        class="px-3 py-2 rounded-md text-sm font-medium transition-colors"
        :class="$route.path.startsWith(link.to) && link.to !== '/'
          ? 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-white'
          : 'text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-900'"
        @click="menuOpen = false"
      >{{ link.label }}</NuxtLink>
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
</template>

<script setup lang="ts">
const { user, isAuthenticated, signOut } = useAuth()
const router = useRouter()
const route = useRoute()
const menuOpen = ref(false)

// Close menu on route change
watch(() => route.path, () => { menuOpen.value = false })

const navLinks = [
  { to: '/dashboard', label: 'Dashboard' },
  { to: '/trips',     label: 'My Trips'  },
  { to: '/map',       label: 'Map'       },
]

async function handleSignOut() {
  menuOpen.value = false
  await signOut()
  router.push('/')
}
</script>
