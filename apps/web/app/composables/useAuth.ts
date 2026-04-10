/**
 * useAuth — thin wrapper around @nuxtjs/supabase primitives.
 *
 * Provides:
 *   - user / session reactive refs
 *   - signOut helper
 *   - getToken() for injecting Authorization: Bearer into API calls
 *   - isAuthenticated convenience computed
 */
export function useAuth() {
  const client  = useSupabaseClient()
  const user    = useSupabaseUser()
  const session = useSupabaseSession()

  const isAuthenticated = computed(() => !!user.value)

  // True when the user's Supabase app_metadata.role === "admin".
  // Set this in the Supabase dashboard: Authentication → Users → Edit → app_metadata: {"role":"admin"}
  const isAdmin = computed(() => (user.value?.app_metadata as any)?.role === 'admin')

  /**
   * Returns the current access token string, or null when unauthenticated.
   * Uses getSession() directly so the token is always fresh — the reactive
   * session ref may not yet be populated when called right after page load.
   */
  async function getToken(): Promise<string | null> {
    const { data } = await client.auth.getSession()
    return data.session?.access_token ?? null
  }

  async function signOut() {
    await client.auth.signOut()
  }

  return { user, session, isAuthenticated, isAdmin, getToken, signOut }
}
