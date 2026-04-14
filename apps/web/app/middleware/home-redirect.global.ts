// Redirect signed-in users away from the homepage on fresh loads only.
// Internal navigation (e.g. clicking the logo) is allowed through.
export default defineNuxtRouteMiddleware((to, from) => {
  if (to.path !== '/') return
  // from.name is undefined on initial page load (no prior route in the SPA)
  if (from.name !== undefined) return
  const { isAuthenticated } = useAuth()
  if (isAuthenticated.value) {
    return navigateTo('/dashboard')
  }
})
