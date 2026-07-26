import type { ReactElement } from 'react'
import { BarChart as TremorBarChart, type BarChartProps } from '@tremor/react'

import { defaultChartColors } from './chart-colors'

export type { BarChartProps }

export function BarChart({
  colors = [...defaultChartColors],
  className = 'h-72',
  ...props
}: BarChartProps): ReactElement {
  return <TremorBarChart className={className} colors={colors} {...props} />
}
