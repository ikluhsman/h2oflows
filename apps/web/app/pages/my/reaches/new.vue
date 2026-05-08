<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-neutral-950 flex flex-col">
    <!-- Header bar -->
    <div class="sticky top-0 z-10 flex items-center justify-between px-4 py-3 bg-white/90 dark:bg-neutral-950/90 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800">
      <div class="flex items-center gap-2 min-w-0">
        <NuxtLink to="/my/reaches" class="text-xs text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300 shrink-0">My Reaches</NuxtLink>
        <span class="text-neutral-300 dark:text-neutral-700 shrink-0">/</span>
        <h2 class="text-sm font-semibold text-neutral-800 dark:text-neutral-100 truncate">New Reach</h2>
      </div>
    </div>

    <!-- Auth gate -->
    <div v-if="!isAuthenticated && authReady" class="flex-1 flex flex-col items-center justify-center gap-3 text-center">
      <h2 class="text-lg font-semibold">Sign in to create a reach</h2>
      <NuxtLink to="/login" class="text-sm text-primary-600 dark:text-primary-400 hover:underline">Sign in</NuxtLink>
    </div>

    <div v-else-if="!authReady" class="flex-1 flex items-center justify-center">
      <div class="w-6 h-6 rounded-full border-2 border-primary-500 border-t-transparent animate-spin" />
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
