/**
 * useTrips — fetch and manage trip records.
 *
 * When authenticated (Supabase session present), requests carry
 * Authorization: Bearer <token> and the API scopes trips by user_id.
 * For anonymous/legacy use, trips fall back to device_id scoping.
 */

export interface TripSummary {
  id:            string
  started_at:    string
  ended_at:      string | null
  duration_min:  number | null
  start_cfs:     number | null
  end_cfs:       number | null
  distance_mi:   number | null
  notes:         string | null
  title:         string | null
  share_consent: boolean | null
  reach_name:    string
  reach_slug:    string
  gauge_name:    string
}

export interface TripDetail extends TripSummary {
  track:       GeoJSONLineString | null
  point_count: number
}

export interface DescribeResult {
  title:       string
  description: string
}

interface GeoJSONLineString {
  type:        'LineString'
  coordinates: [number, number][]
}

export function useTrips() {
  const { apiBase } = useRuntimeConfig().public
  const { getToken } = useAuth()

  function getDeviceId(): string {
    const key = 'h2oflow_device_id'
    let id = localStorage.getItem(key)
    if (!id) {
      id = crypto.randomUUID()
      localStorage.setItem(key, id)
    }
    return id
  }

  /** Build request headers — adds Bearer token when signed in. */
  function authHeaders(extra?: Record<string, string>): HeadersInit {
    const token = getToken()
    const headers: Record<string, string> = { ...extra }
    if (token) headers['Authorization'] = `Bearer ${token}`
    return headers
  }

  async function listTrips(): Promise<TripSummary[]> {
    const res = await fetch(
      `${apiBase}/api/v1/trips?device_id=${getDeviceId()}`,
      { headers: authHeaders() },
    )
    if (!res.ok) throw new Error(`${res.status}`)
    return res.json()
  }

  async function getTrip(id: string): Promise<TripDetail> {
    const res = await fetch(
      `${apiBase}/api/v1/trips/${id}?device_id=${getDeviceId()}`,
      { headers: authHeaders() },
    )
    if (!res.ok) throw new Error(`${res.status}`)
    return res.json()
  }

  async function patchTrip(id: string, patch: { notes?: string; title?: string; share_consent?: boolean }): Promise<void> {
    const res = await fetch(`${apiBase}/api/v1/trips/${id}`, {
      method:  'PATCH',
      headers: authHeaders({ 'Content-Type': 'application/json' }),
      body:    JSON.stringify({ device_id: getDeviceId(), ...patch }),
    })
    if (!res.ok) throw new Error(`${res.status}`)
  }

  async function describeTrip(id: string): Promise<DescribeResult> {
    const res = await fetch(`${apiBase}/api/v1/trips/${id}/describe`, {
      method:  'POST',
      headers: authHeaders({ 'Content-Type': 'application/json' }),
      body:    JSON.stringify({ device_id: getDeviceId() }),
    })
    if (!res.ok) throw new Error(`${res.status}`)
    return res.json()
  }

  return { listTrips, getTrip, patchTrip, describeTrip, getDeviceId }
}
