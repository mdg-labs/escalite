/** Escalite chart palette mapped to design tokens (see tokens/globals.css). */
export const defaultChartColors = [
  'blue',
  'emerald',
  'amber',
  'violet',
  'rose',
] as const

export type ChartColor = (typeof defaultChartColors)[number]
