import { reactive } from 'vue'
import { flowStatusForBand } from '~/utils/flowBand'

interface FlowRange {
  label: string
  min_value: number | null
  max_value: number | null
}

export function useReachFlowBand() {
  const { apiBase } = useRuntimeConfig().public
  const rangesCache = reactive<Record<string, FlowRange[]>>({})

  async function prefetch(slug: string) {
    if (slug in rangesCache) return
    rangesCache[slug] = []
    try {
      const res = await fetch(`${apiBase}/api/v1/reaches/${slug}/flow-ranges`)
      if (res.ok) rangesCache[slug] = await res.json()
    } catch {}
  }

  function bandForCfs(slug: string | null | undefined, cfs: number | null | undefined): string | null {
    if (!slug || cfs == null) return null
    const ranges = rangesCache[slug]
    if (!ranges || ranges.length === 0) return null
    return ranges.find(fr =>
      (fr.min_value == null || cfs >= fr.min_value) &&
      (fr.max_value == null || cfs < fr.max_value)
    )?.label ?? null
  }

  function statusForBand(band: string | null): string {
    return band ? flowStatusForBand(band) : 'unknown'
  }

  return { prefetch, bandForCfs, statusForBand }
}
