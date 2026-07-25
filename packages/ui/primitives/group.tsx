'use client'

import { cva, type VariantProps } from 'class-variance-authority'
import type * as React from 'react'
import { cn } from '../lib/utils'

export const groupVariants = cva(
  'flex w-fit *:focus-visible:z-1 has-[>[data-slot=group]]:gap-2 *:has-focus-visible:z-1',
  {
    defaultVariants: {
      orientation: 'horizontal',
    },
    variants: {
      orientation: {
        horizontal:
          '*:pointer-coarse:after:min-w-auto *:data-slot:has-[~[data-slot]]:rounded-e-none *:data-slot:has-[~[data-slot]]:border-e-0 *:[[data-slot]~[data-slot]:not([data-slot=separator])]:rounded-s-none *:[[data-slot]~[data-slot]]:border-s-0',
        vertical:
          'flex-col *:pointer-coarse:after:min-h-auto *:data-slot:has-[~[data-slot]]:rounded-b-none *:data-slot:has-[~[data-slot]]:border-b-0 *:[[data-slot]~[data-slot]]:rounded-t-none *:[[data-slot]~[data-slot]]:border-t-0',
      },
    },
  },
)

export function Group({
  className,
  orientation,
  children,
  ...props
}: {
  className?: string
  orientation?: VariantProps<typeof groupVariants>['orientation']
  children: React.ReactNode
} & React.ComponentProps<'div'>): React.ReactElement {
  return (
    <div
      className={cn(groupVariants({ orientation }), className)}
      data-orientation={orientation}
      data-slot="group"
      role="group"
      {...props}
    >
      {children}
    </div>
  )
}

export function GroupSeparator({
  className,
  orientation = 'vertical',
  ...props
}: React.ComponentProps<'div'> & { orientation?: 'horizontal' | 'vertical' }): React.ReactElement {
  return (
    <div
      aria-orientation={orientation}
      className={cn(
        'shrink-0 bg-border',
        orientation === 'vertical' ? 'mx-0 w-px self-stretch' : 'my-0 h-px w-full',
        className,
      )}
      data-orientation={orientation}
      data-slot="separator"
      role="separator"
      {...props}
    />
  )
}
