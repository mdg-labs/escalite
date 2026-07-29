export type ChartValueFormatter = (value: number) => string

export type BaseChartProps = {
  data: Record<string, string | number>[]
  index: string
  categories: string[]
  colors?: readonly string[]
  className?: string
  valueFormatter?: ChartValueFormatter
  showAnimation?: boolean
  showLegend?: boolean
  showGridLines?: boolean
  yAxisWidth?: number
}

export type BarChartProps = BaseChartProps

export type AreaChartProps = BaseChartProps
