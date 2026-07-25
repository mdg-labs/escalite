/**
 * Escalite semantic tokens for Tamagui (mobile).
 *
 * Values align with `packages/ui/tokens/severity.css` dark theme and the shared
 * `packages/tokens` contract referenced in doc 03 / doc 05.
 */
export const DEFAULT_THEME = 'dark' as const

export type EscaliteTheme = typeof DEFAULT_THEME | 'light'

/** Okabe–Ito inspired severity palette — dark theme (matches web .dark). */
export const severityColors = {
  severityCritical: '#f0a070',
  severityCriticalForeground: '#ffd8c2',
  severityHigh: '#f5c842',
  severityHighForeground: '#fff0b8',
  severityMedium: '#6eb8e8',
  severityMediumForeground: '#d4ecff',
  severityLow: '#9ad4f5',
  severityLowForeground: '#e4f6ff',
  severityResolved: '#5fd4a4',
  severityResolvedForeground: '#d2f5e6',
  severityInfo: '#6eb8e8',
  severityInfoForeground: '#d4ecff',
} as const

/** Core surface tokens — dark mode default for on-call tooling. */
export const surfaceColors = {
  background: '#0a0a0b',
  backgroundHover: '#141416',
  color: '#f5f5f5',
  colorMuted: '#a3a3a3',
  borderColor: 'rgba(255, 255, 255, 0.08)',
  shadowColor: 'rgba(0, 0, 0, 0.4)',
} as const

export const escaliteTokens = {
  ...severityColors,
  ...surfaceColors,
} as const
