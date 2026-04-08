<template>
  <header class="sticky top-0 z-20 border-b border-gray-200 dark:border-gray-800 bg-white/90 dark:bg-gray-950/90 backdrop-blur-sm">
    <div class="max-w-full px-4 py-3 flex items-center gap-3">
      <NuxtLink to="/" class="flex items-center gap-2 shrink-0">
        <span class="text-xl font-bold tracking-tight">H2OFlows</span>
      </NuxtLink>

      <!-- Breadcrumb / page-level nav injected by each page -->
      <slot />

      <div class="flex-1" />

      <!-- Page-level action buttons (trip banner, Add gauge, etc.) -->
      <slot name="actions" />

      <!-- Auth state -->
      <template v-if="isAuthenticated">
        <button
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          :title="`Signed in as ${user?.email ?? user?.user_metadata?.user_name ?? 'you'}`"
          @click="handleSignOut"
        >Sign out</button>
      </template>
      <template v-else>
        <NuxtLink
          to="/login"
          class="text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
        >Sign in</NuxtLink>
      </template>

      <UColorModeButton size="sm" color="neutral" variant="ghost" />
    </div>
  </header>
</template>

<script setup lang="ts">
const { user, isAuthenticated, signOut } = useAuth()
const router = useRouter()

async function handleSignOut() {
  await signOut()
  router.push('/')
}
</script>
