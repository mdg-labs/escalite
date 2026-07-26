import type { ReactElement } from 'react'
import { AreaChart as TremorAreaChart, type AreaChartProps } from '@tremor/react'

import { defaultChartColors } from './chart-colors'

export type { AreaChartProps }

export function AreaChart({
  colors = [...defaultChartColors],
  className = 'h-72',
  ...props
}: AreaChartProps): ReactElement {
  return <TremorAreaChart className={className} colors={colors} {...props} />
}
