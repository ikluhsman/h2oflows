<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-3xl mx-auto px-4 py-16 flex flex-col items-center gap-4 text-center">
      <div class="text-5xl">🛶</div>
      <h1 class="text-xl font-bold">Trip Reports</h1>
      <p class="text-sm text-gray-500 max-w-xs leading-relaxed">
        Share your run, confirm flow conditions, and help build the data that makes every reach smarter.
        Trip reports are coming soon.
      </p>
      <NuxtLink
        to="/dashboard"
        class="mt-2 inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold transition-colors"
      >Go to dashboard</NuxtLink>
    </main>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ ssr: false })
</script>
