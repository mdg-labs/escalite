import type { ReactElement, ReactNode } from 'react'

import { cn } from '../lib/utils'

export type MetricProps = {
  children?: ReactNode
  className?: string
}

export function Metric({ children, className }: MetricProps): ReactElement {
  return (
    <p className={cn('text-3xl font-semibold tracking-tight text-foreground', className)}>
      {children}
    </p>
  )
}
