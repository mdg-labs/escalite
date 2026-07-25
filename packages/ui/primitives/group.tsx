import type * as React from 'react'

import { cn } from '../lib/utils'

export function Group({
  className,
  ...props
}: React.ComponentProps<'div'>): React.ReactElement {
  return (
    <div
      className={cn('flex flex-wrap items-center gap-2', className)}
      data-slot="group"
      {...props}
    />
  )
}
