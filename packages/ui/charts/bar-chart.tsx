import type { ReactElement } from 'react'
import {
  Bar,
  BarChart as RechartsBarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import { cn } from '../lib/utils'
import { defaultChartColors } from './chart-colors'
import { resolveChartColor } from './chart-color-map'
import type { BarChartProps } from './chart-types'

export type { BarChartProps }

export function BarChart({
  categories,
  className = 'h-72',
  colors = [...defaultChartColors],
  data,
  index,
  showAnimation = true,
  showGridLines = true,
  showLegend = false,
  valueFormatter,
  yAxisWidth = 56,
}: BarChartProps): ReactElement {
  return (
    <div className={cn('w-full text-muted-foreground', className)}>
      <ResponsiveContainer height="100%" width="100%">
        <RechartsBarChart
          data={data}
          margin={{ bottom: 4, left: 0, right: 8, top: 8 }}
        >
          {showGridLines ? (
            <CartesianGrid className="stroke-border/60" strokeDasharray="3 3" vertical={false} />
          ) : null}
          <XAxis
            axisLine={false}
            dataKey={index}
            tick={{ fill: 'currentColor', fontSize: 12 }}
            tickLine={false}
          />
          <YAxis
            axisLine={false}
            tick={{ fill: 'currentColor', fontSize: 12 }}
            tickFormatter={valueFormatter}
            tickLine={false}
            width={yAxisWidth}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'var(--popover)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-md)',
              color: 'var(--popover-foreground)',
            }}
            formatter={(value) =>
              typeof value === 'number' && valueFormatter ? valueFormatter(value) : value
            }
            labelStyle={{ color: 'var(--popover-foreground)' }}
          />
          {showLegend ? (
            <Legend
              formatter={(value) => <span className="text-foreground">{value}</span>}
              iconType="circle"
              wrapperStyle={{ fontSize: 12, paddingTop: 12 }}
            />
          ) : null}
          {categories.map((category, categoryIndex) => (
            <Bar
              dataKey={category}
              fill={resolveChartColor(colors[categoryIndex] ?? defaultChartColors[categoryIndex], categoryIndex)}
              isAnimationActive={showAnimation}
              key={category}
              maxBarSize={48}
              radius={[4, 4, 0, 0]}
            />
          ))}
        </RechartsBarChart>
      </ResponsiveContainer>
    </div>
  )
}
