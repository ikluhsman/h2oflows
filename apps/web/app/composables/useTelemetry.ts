/**
 * Lightweight telemetry consent + fire-and-forget event helper.
 *
 * Consent is stored in localStorage (`h2oflow_telemetry_consent`, default true).
 * All pings are fire-and-forget — errors are silently swallowed.
 */

const CONSENT_KEY = 'h2oflow_telemetry_consent'

export function useTelemetryConsent() {
  const enabled = ref<boolean>(true)

  onMounted(() => {
    const stored = localStorage.getItem(CONSENT_KEY)
    enabled.value = stored === null ? true : stored === 'true'
  })

  function setConsent(value: boolean) {
    enabled.value = value
    localStorage.setItem(CONSENT_KEY, String(value))
  }

  return { enabled, setConsent }
}

/**
 * Submit a run note / flow contribution.
 * Saves locally and, when the backend exists, POSTs fire-and-forget.
 */
export interface RunNote {
  reach_slug:       string
  note_type:        'trip_report' | 'flow_update' | 'hazard_alert' | 'general'
  flow_impression:  'too_low' | 'good' | 'high'
  note_text:        string
  observed_at:      string
  timestamp:        string
}

const NOTES_KEY = 'h2oflow_run_notes'

export function useRunNotes(reachSlug: string) {
  const { apiBase } = useRuntimeConfig().public
  const consent = useTelemetryConsent()

  function submit(note: Omit<RunNote, 'reach_slug' | 'timestamp'>) {
    if (!consent.enabled.value) return

    const entry: RunNote = {
      reach_slug: reachSlug,
      timestamp:  new Date().toISOString(),
      ...note,
    }

    // Persist locally
    try {
      const existing: RunNote[] = JSON.parse(localStorage.getItem(NOTES_KEY) ?? '[]')
      existing.push(entry)
      localStorage.setItem(NOTES_KEY, JSON.stringify(existing.slice(-100)))
    } catch { /* ignore */ }

    // Fire-and-forget POST (backend deferred — fails silently)
    fetch(`${apiBase}/api/v1/reaches/${reachSlug}/contributions`, {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify(entry),
    }).catch(() => { /* backend not yet deployed */ })
  }

  return { submit }
}
