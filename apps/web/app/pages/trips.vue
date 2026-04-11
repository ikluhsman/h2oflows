<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950">
    <AppHeader />

    <main class="max-w-3xl mx-auto px-4 py-8">

      <!-- Sign-in prompt for anonymous users -->
      <div
        v-if="!isAuthenticated"
        class="mb-6 rounded-xl border border-blue-100 dark:border-blue-900 bg-blue-50 dark:bg-blue-950/30 px-4 py-3 flex items-center justify-between gap-4"
      >
        <div class="min-w-0">
          <p class="text-sm font-medium text-blue-700 dark:text-blue-300">Sync trips across devices</p>
          <p class="text-xs text-blue-600/70 dark:text-blue-400/70 mt-0.5">Sign in to save your trips to your account and access them anywhere.</p>
        </div>
        <NuxtLink
          to="/login"
          class="shrink-0 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-xs font-semibold transition-colors"
        >Sign in</NuxtLink>
      </div>

      <div class="flex items-center justify-between mb-6">
        <h1 class="text-xl font-bold">My Trips</h1>
        <span v-if="trips.length" class="text-sm text-gray-400">{{ trips.length }} trip{{ trips.length === 1 ? '' : 's' }}</span>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-24 text-gray-400 text-sm">
        Loading trips…
      </div>

      <!-- Error -->
      <div v-else-if="error" class="py-24 text-center">
        <p class="text-sm text-red-500">{{ error }}</p>
        <button class="mt-3 text-sm text-blue-600 hover:underline" @click="load">Try again</button>
      </div>

      <!-- Empty -->
      <div v-else-if="trips.length === 0" class="py-24 flex flex-col items-center gap-4 text-center">
        <div class="text-5xl">🛶</div>
        <h2 class="text-lg font-semibold">No trips yet</h2>
        <p class="text-sm text-gray-500 max-w-xs leading-relaxed">
          Tap <strong>Track it</strong> on any gauge in your dashboard to start recording a trip. Your trips will appear here.
        </p>
        <NuxtLink
          to="/dashboard"
          class="mt-2 inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold transition-colors"
        >Go to dashboard</NuxtLink>
      </div>

      <!-- Trip list -->
      <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <TripCard
          v-for="trip in trips"
          :key="trip.id"
          :trip="trip"
          @open="openTrip(trip.id)"
        />
      </div>

      <!-- Capacitor note — shown in browser context -->
      <div class="mt-10 rounded-xl border border-amber-100 dark:border-amber-900 bg-amber-50 dark:bg-amber-950/30 px-4 py-3 text-xs text-amber-700 dark:text-amber-400 leading-relaxed">
        <strong>Beta:</strong> GPS recording uses your browser's location API. For full offline support and background tracking, the native mobile app is coming soon.
      </div>
    </main>

    <!-- Trip detail modal -->
    <TripDetailModal v-model:open="detailOpen" :trip-id="selectedTripId" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useTrips, type TripSummary } from '~/composables/useTrips'

// Disable SSR so Supabase session is always available on first render
definePageMeta({ ssr: false })

const { listTrips } = useTrips()
const { isAuthenticated } = useAuth()

const trips          = ref<TripSummary[]>([])
const loading        = ref(true)
const error          = ref('')
const detailOpen     = ref(false)
const selectedTripId = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value   = ''
  try {
    trips.value = await listTrips()
  } catch {
    error.value = "Could not load trips. Make sure you're connected."
  } finally {
    loading.value = false
  }
}

function openTrip(id: string) {
  selectedTripId.value = id
  detailOpen.value     = true
}

onMounted(load)

// Re-load when returning to the page so newly uploaded trips appear.
// Nuxt 4's useRoute().meta is available but onActivated is simpler here.
// Since this page is not keepAlive by default, onMounted handles it.
</script>
