export const PALETTES = [
  // Slate row (cool neutral)
  { id: 'h2oflows-slate', label: 'H2OFlows / Slate', primary: 'blue',    neutral: 'slate', primarySwatch: '#3b82f6', neutralSwatch: '#64748b' },
  { id: 'ocean-slate',    label: 'Ocean / Slate',    primary: 'sky',     neutral: 'slate', primarySwatch: '#0ea5e9', neutralSwatch: '#64748b' },
  { id: 'river-slate',    label: 'River / Slate',    primary: 'teal',    neutral: 'slate', primarySwatch: '#14b8a6', neutralSwatch: '#64748b' },
  { id: 'forest-slate',   label: 'Forest / Slate',   primary: 'emerald', neutral: 'slate', primarySwatch: '#10b981', neutralSwatch: '#64748b' },
  { id: 'indigo-slate',   label: 'Indigo / Slate',   primary: 'indigo',  neutral: 'slate', primarySwatch: '#6366f1', neutralSwatch: '#64748b' },
  // Stone row (warm neutral)
  { id: 'h2oflows-stone', label: 'H2OFlows / Stone', primary: 'blue',    neutral: 'stone', primarySwatch: '#3b82f6', neutralSwatch: '#78716c' },
  { id: 'ocean-stone',    label: 'Ocean / Stone',    primary: 'sky',     neutral: 'stone', primarySwatch: '#0ea5e9', neutralSwatch: '#78716c' },
  { id: 'river-stone',    label: 'River / Stone',    primary: 'teal',    neutral: 'stone', primarySwatch: '#14b8a6', neutralSwatch: '#78716c' },
  { id: 'forest-stone',   label: 'Forest / Stone',   primary: 'emerald', neutral: 'stone', primarySwatch: '#10b981', neutralSwatch: '#78716c' },
  { id: 'indigo-stone',   label: 'Indigo / Stone',   primary: 'indigo',  neutral: 'stone', primarySwatch: '#6366f1', neutralSwatch: '#78716c' },
] as const

export type PaletteId = typeof PALETTES[number]['id']

export default defineAppConfig({
  ui: {
    colors: {
      primary: 'blue',
      neutral: 'slate',
    },
  },
})
