import { config as defaultConfig } from '@tamagui/config/v3'
import { createTamagui } from 'tamagui'

import {
  brandColors,
  DEFAULT_THEME,
  escaliteTokens,
  severityColors,
  surfaceColors,
} from '@escalite/tokens'

export const tamaguiConfig = createTamagui({
  ...defaultConfig,
  tokens: {
    ...defaultConfig.tokens,
    color: {
      ...defaultConfig.tokens.color,
      ...escaliteTokens,
    },
  },
  themes: {
    ...defaultConfig.themes,
    dark: {
      ...defaultConfig.themes.dark,
      background: surfaceColors.background,
      backgroundHover: surfaceColors.backgroundHover,
      color: surfaceColors.color,
      colorMuted: surfaceColors.colorMuted,
      borderColor: surfaceColors.borderColor,
      shadowColor: surfaceColors.shadowColor,
      ...severityColors,
      ...brandColors,
    },
    light: {
      ...defaultConfig.themes.light,
      ...severityColors,
      ...brandColors,
    },
  },
  defaultTheme: DEFAULT_THEME,
})

export default tamaguiConfig

export type AppConfig = typeof tamaguiConfig

declare module 'tamagui' {
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface TamaguiCustomConfig extends AppConfig {}
}

export { DEFAULT_THEME }
