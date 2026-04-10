<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-950">
    <p class="text-sm text-gray-400">{{ message }}</p>
  </div>
</template>

<script setup lang="ts">
// Must be client-only — PKCE code verifier lives in sessionStorage (no SSR access).
definePageMeta({ ssr: false })

const client  = useSupabaseClient()
const route   = useRoute()
const message = ref('Signing in…')

const code = route.query.code as string | undefined

if (code) {
  try {
    const { error } = await client.auth.exchangeCodeForSession(code)
    if (error) {
      message.value = `Sign-in failed: ${error.message}`
    } else {
      // Hard redirect so the server receives the new session cookie on the next request.
      window.location.href = '/dashboard'
    }
  } catch (e: any) {
    message.value = `Sign-in failed: ${e?.message ?? 'unknown error'}`
  }
} else {
  window.location.href = '/'
}
</script>
