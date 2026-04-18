<template>
  <nav class="fixed bottom-0 inset-x-0 z-30 sm:hidden bg-white dark:bg-gray-950 border-t border-gray-200 dark:border-gray-800 flex safe-area-bottom">
    <NuxtLink
      to="/dashboard"
      class="flex-1 flex flex-col items-center justify-center py-2 gap-0.5 text-xs font-medium transition-colors"
      :class="route.path === '/dashboard'
        ? 'text-blue-600 dark:text-blue-400'
        : 'text-gray-500 dark:text-gray-400'"
    >
      <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="4" rx="1"/><rect x="14" y="10" width="7" height="11" rx="1"/><rect x="3" y="13" width="7" height="8" rx="1"/>
      </svg>
      Dashboard
    </NuxtLink>
    <NuxtLink
      to="/explore"
      class="flex-1 flex flex-col items-center justify-center py-2 gap-0.5 text-xs font-medium transition-colors"
      :class="route.path === '/explore'
        ? 'text-blue-600 dark:text-blue-400'
        : 'text-gray-500 dark:text-gray-400'"
    >
      <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M4 14c3-6 6-9 8-9s5 9 8 9 5-9 8-9"/>
      </svg>
      Explore
    </NuxtLink>
  </nav>
</template>

<script setup lang="ts">
const route = useRoute()
</script>
