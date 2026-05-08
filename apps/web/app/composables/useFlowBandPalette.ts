import { computed, watchEffect } from 'vue'
import { useThemeStore } from '~/stores/theme'
import { PALETTES } from '../../app.config'
import { PALETTE_FLOW_SOLID, PALETTE_BADGE_CLASS } from '~/utils/flowBand'

/**
 * Reads the active palette from the theme store and exposes palette-aware
 * flow band colors. Also sets CSS custom properties on the document root so
 * templates can reference `var(--flow-low)` / `var(--flow-running)` / `var(--flow-high)`
 * without needing the composable directly.
 */
export function useFlowBandPalette() {
  const themeStore = useThemeStore()

  const primary = computed(() => {
    const p = PALETTES.find(p => p.id === themeStore.paletteId)
    return p?.primary ?? 'blue'
  })

  const flowSolid = computed(() =>
    PALETTE_FLOW_SOLID[primary.value] ?? PALETTE_FLOW_SOLID.blue
  )

  const badgeClass = computed(() =>
    PALETTE_BADGE_CLASS[primary.value] ?? PALETTE_BADGE_CLASS.blue
  )

  if (import.meta.client) {
    watchEffect(() => {
      const c = flowSolid.value
      const root = document.documentElement
      root.style.setProperty('--flow-low',     c.low)
      root.style.setProperty('--flow-running', c.running)
      root.style.setProperty('--flow-high',    c.high)
    })
  }

  function bandSolid(band?: string | null, status?: string | null): string {
    const c = flowSolid.value
    const b = band ?? (status === 'caution' ? 'low' : status === 'runnable' ? 'running' : status === 'flood' ? 'high' : null)
    if (b === 'low')     return c.low
    if (b === 'running') return c.running
    if (b === 'high')    return c.high
    return '#9ca3af'
  }

  function bandBadgeClass(band?: string | null, status?: string | null): string {
    const table = badgeClass.value
    const b = band ?? (status === 'caution' ? 'low' : status === 'runnable' ? 'running' : status === 'flood' ? 'high' : null)
    return b ? (table[b as keyof typeof table] ?? '') : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
  }

  function bandFill(band?: string | null, status?: string | null): string {
    const solid = bandSolid(band, status)
    if (solid === '#9ca3af') return 'rgba(156,163,175,0.10)'
    // Convert hex to rgba with opacity
    const r = parseInt(solid.slice(1, 3), 16)
    const g = parseInt(solid.slice(3, 5), 16)
    const b2 = parseInt(solid.slice(5, 7), 16)
    return `rgba(${r},${g},${b2},0.22)`
  }

  return { primary, flowSolid, bandSolid, bandBadgeClass, bandFill }
}
