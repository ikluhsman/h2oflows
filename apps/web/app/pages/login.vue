<template>
  <div class="min-h-screen flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-950 px-4">
    <div class="w-full max-w-sm">

      <!-- Logo -->
      <div class="flex flex-col items-center mb-8 gap-3">
        <svg class="w-10 h-10 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M2 12c2-4 4-6 6-6s4 6 6 6 4-6 6-6" stroke-linecap="round"/>
          <path d="M2 18c2-4 4-6 6-6s4 6 6 6 4-6 6-6" stroke-linecap="round" opacity="0.4"/>
        </svg>
        <div class="text-center">
          <h1 class="text-xl font-bold tracking-tight">H2OFlows</h1>
          <p class="text-sm text-gray-500 mt-1">Sign in to sync trips and save your dashboard</p>
        </div>
      </div>

      <!-- Card -->
      <div class="bg-white dark:bg-gray-900 rounded-2xl border border-gray-200 dark:border-gray-800 shadow-sm p-6 space-y-4">

        <!-- Google OAuth -->
        <button
          class="w-full flex items-center justify-center gap-2.5 px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-sm font-medium text-gray-800 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          :disabled="loading"
          @click="signInWithProvider('google')"
        >
          <svg class="w-4 h-4 shrink-0" viewBox="0 0 24 24">
            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
            <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z" fill="#FBBC05"/>
            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
          </svg>
          <span v-if="loading">Connecting…</span>
          <span v-else>Continue with Google</span>
        </button>

        <div class="relative flex items-center gap-3">
          <div class="flex-1 h-px bg-gray-200 dark:bg-gray-700"/>
          <span class="text-xs text-gray-400">or</span>
          <div class="flex-1 h-px bg-gray-200 dark:bg-gray-700"/>
        </div>

        <!-- Email / password -->
        <form class="space-y-3" @submit.prevent="signInWithEmail">
          <div>
            <input
              v-model="email"
              type="email"
              placeholder="Email"
              autocomplete="email"
              required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <input
              v-model="password"
              type="password"
              placeholder="Password"
              autocomplete="current-password"
              required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <p v-if="authError" class="text-xs text-red-500">{{ authError }}</p>

          <button
            type="submit"
            :disabled="loading || !email || !password"
            class="w-full py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-semibold transition-colors"
          >
            <span v-if="loading">Signing in…</span>
            <span v-else>Sign in</span>
          </button>
        </form>

        <!-- Sign up link -->
        <p class="text-xs text-center text-gray-400">
          Don't have an account?
          <button class="text-blue-500 hover:text-blue-600 font-medium" @click="signUpWithEmail">Sign up</button>
        </p>
      </div>

      <!-- Back to map -->
      <div class="mt-6 text-center">
        <NuxtLink to="/map" class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
          ← Back to map
        </NuxtLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

definePageMeta({ layout: false })

const client = useSupabaseClient()
const router = useRouter()
const route  = useRoute()

const email    = ref('')
const password = ref('')
const loading  = ref(false)
const authError = ref('')

// Redirect destination — come from wherever they were trying to go
const redirectTo = computed(() => (route.query.redirect as string) || '/trips')

async function signInWithProvider(provider: 'google') {
  loading.value   = true
  authError.value = ''
  const { error } = await client.auth.signInWithOAuth({
    provider,
    options: { redirectTo: `${window.location.origin}/confirm` },
  })
  if (error) {
    authError.value = error.message
    loading.value   = false
  }
  // On success the browser navigates away — loading stays true
}

async function signInWithEmail() {
  if (!email.value || !password.value) return
  loading.value   = true
  authError.value = ''
  const { error } = await client.auth.signInWithPassword({
    email:    email.value,
    password: password.value,
  })
  loading.value = false
  if (error) {
    authError.value = error.message
  } else {
    router.push(redirectTo.value)
  }
}

async function signUpWithEmail() {
  if (!email.value || !password.value) return
  loading.value   = true
  authError.value = ''
  const { error } = await client.auth.signUp({
    email:    email.value,
    password: password.value,
    options:  { emailRedirectTo: `${window.location.origin}/confirm` },
  })
  loading.value = false
  if (error) {
    authError.value = error.message
  } else {
    authError.value = '' // show a "check your email" message
    alert('Check your email to confirm your account.')
  }
}
</script>
