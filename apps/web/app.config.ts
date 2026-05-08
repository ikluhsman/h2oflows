// Palette names with paddling/water themes:
//   H2OFlows (blue)   — brand default
//   Ocean    (sky)    — open ocean blue
//   River    (teal)   — clear river water
//   Forest   (emerald)— canyon walls and pines
//   Indigo   (indigo) — deep canyon shadows
//   Sunset   (orange) — sunset paddle / safety helmets
//   Coral    (fuchsia)— vivid drysuit / drytop accents
//   Dawn     (rose)   — dawn patrol pre-launch sky
//   Moss     (lime)   — riverbank moss & algae
export const PALETTES = [
  // Slate row (cool neutral)
  { id: 'h2oflows-slate', label: 'H2OFlows / Slate', primary: 'blue',    neutral: 'slate', primarySwatch: '#3b82f6', neutralSwatch: '#64748b' },
  { id: 'ocean-slate',    label: 'Ocean / Slate',    primary: 'sky',     neutral: 'slate', primarySwatch: '#0ea5e9', neutralSwatch: '#64748b' },
  { id: 'river-slate',    label: 'River / Slate',    primary: 'teal',    neutral: 'slate', primarySwatch: '#14b8a6', neutralSwatch: '#64748b' },
  { id: 'forest-slate',   label: 'Forest / Slate',   primary: 'emerald', neutral: 'slate', primarySwatch: '#10b981', neutralSwatch: '#64748b' },
  { id: 'indigo-slate',   label: 'Indigo / Slate',   primary: 'indigo',  neutral: 'slate', primarySwatch: '#6366f1', neutralSwatch: '#64748b' },
  { id: 'sunset-slate',   label: 'Sunset / Slate',   primary: 'orange',  neutral: 'slate', primarySwatch: '#f97316', neutralSwatch: '#64748b' },
  { id: 'coral-slate',    label: 'Coral / Slate',    primary: 'fuchsia', neutral: 'slate', primarySwatch: '#d946ef', neutralSwatch: '#64748b' },
  { id: 'dawn-slate',     label: 'Dawn / Slate',     primary: 'rose',    neutral: 'slate', primarySwatch: '#f43f5e', neutralSwatch: '#64748b' },
  { id: 'moss-slate',     label: 'Moss / Slate',     primary: 'lime',    neutral: 'slate', primarySwatch: '#84cc16', neutralSwatch: '#64748b' },
  // Stone row (warm neutral)
  { id: 'h2oflows-stone', label: 'H2OFlows / Stone', primary: 'blue',    neutral: 'stone', primarySwatch: '#3b82f6', neutralSwatch: '#78716c' },
  { id: 'ocean-stone',    label: 'Ocean / Stone',    primary: 'sky',     neutral: 'stone', primarySwatch: '#0ea5e9', neutralSwatch: '#78716c' },
  { id: 'river-stone',    label: 'River / Stone',    primary: 'teal',    neutral: 'stone', primarySwatch: '#14b8a6', neutralSwatch: '#78716c' },
  { id: 'forest-stone',   label: 'Forest / Stone',   primary: 'emerald', neutral: 'stone', primarySwatch: '#10b981', neutralSwatch: '#78716c' },
  { id: 'indigo-stone',   label: 'Indigo / Stone',   primary: 'indigo',  neutral: 'stone', primarySwatch: '#6366f1', neutralSwatch: '#78716c' },
  { id: 'sunset-stone',   label: 'Sunset / Stone',   primary: 'orange',  neutral: 'stone', primarySwatch: '#f97316', neutralSwatch: '#78716c' },
  { id: 'coral-stone',    label: 'Coral / Stone',    primary: 'fuchsia', neutral: 'stone', primarySwatch: '#d946ef', neutralSwatch: '#78716c' },
  { id: 'dawn-stone',     label: 'Dawn / Stone',     primary: 'rose',    neutral: 'stone', primarySwatch: '#f43f5e', neutralSwatch: '#78716c' },
  { id: 'moss-stone',     label: 'Moss / Stone',     primary: 'lime',    neutral: 'stone', primarySwatch: '#84cc16', neutralSwatch: '#78716c' },
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
