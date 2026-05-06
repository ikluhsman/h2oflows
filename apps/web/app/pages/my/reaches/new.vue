<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950 flex flex-col">
    <!-- Header bar -->
    <div class="sticky top-0 z-10 flex items-center justify-between px-4 py-3 bg-white/90 dark:bg-gray-950/90 backdrop-blur-sm border-b border-gray-200 dark:border-gray-800">
      <div class="flex items-center gap-2 min-w-0">
        <NuxtLink to="/my/reaches" class="text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 shrink-0">My Reaches</NuxtLink>
        <span class="text-gray-300 dark:text-gray-700 shrink-0">/</span>
        <h2 class="text-sm font-semibold text-gray-800 dark:text-gray-100 truncate">New Reach</h2>
      </div>
    </div>

    <!-- Auth gate -->
    <div v-if="!isAuthenticated && authReady" class="flex-1 flex flex-col items-center justify-center gap-3 text-center">
      <h2 class="text-lg font-semibold">Sign in to create a reach</h2>
      <NuxtLink to="/login" class="text-sm text-blue-600 dark:text-blue-400 hover:underline">Sign in</NuxtLink>
    </div>

    <div v-else-if="!authReady" class="flex-1 flex items-center justify-center">
      <div class="w-6 h-6 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
    </div>

    <div v-else class="flex-1 max-w-5xl w-full mx-auto px-4 py-6">
      <UserReachAuthor @created="onCreated" @cancel="router.push('/my/reaches')" />
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ ssr: false })

const router = useRouter()
const { isAuthenticated } = useAuth()
const authReady = ref(false)

onMounted(() => { authReady.value = true })

function onCreated(slug: string) {
  router.push(`/my/reaches/${slug}`)
}
</script>
