<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-950">
    <p class="text-sm text-gray-400">{{ message }}</p>
  </div>
</template>

<script setup lang="ts">
// Exchange the Supabase auth code from the URL for a session, then redirect.
// This page is the OAuth / magic-link / email-confirm callback target.
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
      await navigateTo('/dashboard')
    }
  } catch (e: any) {
    message.value = `Sign-in failed: ${e?.message ?? 'unknown error'}`
  }
} else {
  await navigateTo('/')
}
</script>
