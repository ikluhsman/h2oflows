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

  // isDataAdmin is loaded from the API and reflects the user_roles table.
  // Resolves to true for site_admins too (server-side check is authoritative).
  const _dataAdminLoaded = ref(false)
  const _isDataAdmin = ref(false)

  async function loadAdminRoles() {
    if (!user.value) return
    const token = await getToken()
    if (!token) return
    const { apiBase } = useRuntimeConfig().public
    try {
      const res = await fetch(`${apiBase}/api/v1/admin/me/roles`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (res.ok) {
        const data = await res.json()
        _isDataAdmin.value = !!data.is_data_admin
      }
    } catch { /* non-fatal */ } finally {
      _dataAdminLoaded.value = true
    }
  }

  const isDataAdmin = computed(() => isAdmin.value || _isDataAdmin.value)

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

  return { user, session, isAuthenticated, isAdmin, isDataAdmin, loadAdminRoles, getToken, signOut }
}
