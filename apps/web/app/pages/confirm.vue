<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-950">
    <p class="text-sm text-gray-400">Signing in…</p>
  </div>
</template>

<script setup lang="ts">
// With implicit flow the Supabase client processes the #access_token fragment
// automatically on initialization. Just wait for the session then redirect.
definePageMeta({ ssr: false })

const client = useSupabaseClient()

onMounted(async () => {
  // Give the client a tick to process the hash fragment
  const { data } = await client.auth.getSession()
  if (data.session) {
    window.location.href = '/dashboard'
  } else {
    // Listen for the auth state change triggered by hash processing
    const { data: { subscription } } = client.auth.onAuthStateChange((event, session) => {
      if (session) {
        subscription.unsubscribe()
        window.location.href = '/dashboard'
      }
    })
    // Timeout fallback
    setTimeout(() => { window.location.href = '/' }, 5000)
  }
})
</script>
