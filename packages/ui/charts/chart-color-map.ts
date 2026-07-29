import type { ChartColor } from './chart-colors'

const chartColorValues: Record<ChartColor, string> = {
  blue: 'var(--chart-1)',
  emerald: 'var(--chart-2)',
  amber: 'var(--chart-3)',
  violet: 'var(--chart-4)',
  rose: 'var(--chart-5)',
}

export function resolveChartColor(color: string, index: number): string {
  if (color in chartColorValues) {
    return chartColorValues[color as ChartColor]
  }

  const palette = Object.values(chartColorValues)
  return palette[index % palette.length] ?? 'var(--chart-1)'
}
