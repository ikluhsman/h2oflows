import { PALETTES } from '../../app.config'
import { useThemeStore } from '~/stores/theme'

export default defineNuxtPlugin(() => {
  const store = useThemeStore()
  const appConfig = useAppConfig()

  const palette = PALETTES.find(p => p.id === store.paletteId)
  if (palette) {
    appConfig.ui.colors.primary = palette.primary
    appConfig.ui.colors.neutral = palette.neutral
  }
})
