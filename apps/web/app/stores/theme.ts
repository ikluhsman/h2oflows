import { defineStore } from 'pinia'
import type { PaletteId } from '../../app.config'

export const useThemeStore = defineStore('theme', {
  state: () => ({
    paletteId: 'h2oflows-slate' as PaletteId,
  }),
  persist: true,
})
