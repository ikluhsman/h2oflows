export interface Dashboard {
  id: string
  slug: string
  name: string
  position: number
  created_at: string
}

// Module-level state — shared across all composable calls in the same page lifecycle.
const dashboards = ref<Dashboard[]>([])
const activeDashboardId = ref<string | null>(null)
const loaded = ref(false)

const ACTIVE_DB_KEY = 'h2oflow_active_dashboard_id'

export function useDashboards() {
  const { apiBase } = useRuntimeConfig().public
  const { getToken } = useAuth()

  const activeDashboard = computed(() =>
    dashboards.value.find(d => d.id === activeDashboardId.value) ?? dashboards.value[0] ?? null
  )

  async function load() {
    const token = await getToken()
    if (!token) return
    const res = await fetch(`${apiBase}/api/v1/me/dashboards`, {
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => null)
    if (!res?.ok) return
    const data = await res.json()
    dashboards.value = data.dashboards ?? []
    if (dashboards.value.length) {
      // Restore last-selected dashboard; fall back to first if it no longer exists.
      const saved = import.meta.client ? localStorage.getItem(ACTIVE_DB_KEY) : null
      if (saved && dashboards.value.some(d => d.id === saved)) {
        activeDashboardId.value = saved
      } else {
        activeDashboardId.value = dashboards.value[0].id
      }
    }
    loaded.value = true
  }

  function setActive(id: string) {
    activeDashboardId.value = id
    if (import.meta.client) localStorage.setItem(ACTIVE_DB_KEY, id)
  }

  async function create(name: string): Promise<Dashboard | null> {
    const token = await getToken()
    if (!token) return null
    const res = await fetch(`${apiBase}/api/v1/me/dashboards`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    }).catch(() => null)
    if (!res?.ok) return null
    const d: Dashboard = await res.json()
    dashboards.value = [...dashboards.value, d]
    return d
  }

  async function rename(slug: string, name: string) {
    const token = await getToken()
    if (!token) return
    await fetch(`${apiBase}/api/v1/me/dashboards/${slug}`, {
      method: 'PATCH',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    }).catch(() => {})
    dashboards.value = dashboards.value.map(d => d.slug === slug ? { ...d, name } : d)
  }

  async function remove(slug: string) {
    const token = await getToken()
    if (!token) return
    await fetch(`${apiBase}/api/v1/me/dashboards/${slug}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => {})
    const removed = dashboards.value.find(d => d.slug === slug)
    dashboards.value = dashboards.value.filter(d => d.slug !== slug)
    // If active dashboard deleted, switch to first remaining
    if (removed && activeDashboardId.value === removed.id) {
      const newId = dashboards.value[0]?.id ?? null
      activeDashboardId.value = newId
      if (import.meta.client) {
        if (newId) localStorage.setItem(ACTIVE_DB_KEY, newId)
        else localStorage.removeItem(ACTIVE_DB_KEY)
      }
    }
  }

  async function reorder(ids: string[]) {
    const token = await getToken()
    if (!token) return
    // Optimistic: reorder local list immediately
    const idSet = new Map(ids.map((id, i) => [id, i]))
    dashboards.value = [...dashboards.value].sort((a, b) => {
      const ai = idSet.get(a.id) ?? 999
      const bi = idSet.get(b.id) ?? 999
      return ai - bi
    })
    await fetch(`${apiBase}/api/v1/me/dashboards-reorder`, {
      method: 'PATCH',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids }),
    }).catch(() => {})
  }

  return { dashboards, activeDashboard, activeDashboardId, loaded, load, setActive, create, rename, remove, reorder }
}
