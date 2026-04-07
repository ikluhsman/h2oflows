/**
 * useTrips — fetch and manage trip records for the current device.
 *
 * All trips are keyed by device_id (anonymous beta; no auth required).
 * The device_id is a random UUID stored in localStorage, generated once on first install.
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

  function getDeviceId(): string {
    const key = 'h2oflow_device_id'
    let id = localStorage.getItem(key)
    if (!id) {
      id = crypto.randomUUID()
      localStorage.setItem(key, id)
    }
    return id
  }

  async function listTrips(): Promise<TripSummary[]> {
    const res = await fetch(`${apiBase}/api/v1/trips?device_id=${getDeviceId()}`)
    if (!res.ok) throw new Error(`${res.status}`)
    return res.json()
  }

  async function getTrip(id: string): Promise<TripDetail> {
    const res = await fetch(`${apiBase}/api/v1/trips/${id}?device_id=${getDeviceId()}`)
    if (!res.ok) throw new Error(`${res.status}`)
    return res.json()
  }

  async function patchTrip(id: string, patch: { notes?: string; title?: string; share_consent?: boolean }): Promise<void> {
    const res = await fetch(`${apiBase}/api/v1/trips/${id}`, {
      method:  'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify({ device_id: getDeviceId(), ...patch }),
    })
    if (!res.ok) throw new Error(`${res.status}`)
  }

  async function describeTrip(id: string): Promise<DescribeResult> {
    const res = await fetch(`${apiBase}/api/v1/trips/${id}/describe`, {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify({ device_id: getDeviceId() }),
    })
    if (!res.ok) throw new Error(`${res.status}`)
    return res.json()
  }

  return { listTrips, getTrip, patchTrip, describeTrip, getDeviceId }
}
