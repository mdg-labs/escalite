'use client'

import { NumberField as NumberFieldPrimitive } from '@base-ui/react/number-field'
import { MinusIcon, PlusIcon } from 'lucide-react'
import type * as React from 'react'

import { cn } from '../lib/utils'

export function NumberField({
  className,
  size = 'default',
  ...props
}: NumberFieldPrimitive.Root.Props & {
  size?: 'sm' | 'default' | 'lg'
}): React.ReactElement {
  return (
    <NumberFieldPrimitive.Root
      className={cn('flex w-full flex-col items-start gap-2', className)}
      data-size={size}
      data-slot="number-field"
      {...props}
    />
  )
}

export function NumberFieldGroup({
  className,
  ...props
}: NumberFieldPrimitive.Group.Props): React.ReactElement {
  return (
    <NumberFieldPrimitive.Group
      className={cn(
        'relative flex w-full justify-between rounded-lg border border-input bg-background text-base text-foreground shadow-xs/5 ring-ring/24 transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] focus-within:border-ring focus-within:ring-[3px] data-disabled:pointer-events-none data-disabled:opacity-64 sm:text-sm dark:bg-input/32',
        className,
      )}
      data-slot="number-field-group"
      {...props}
    />
  )
}

export function NumberFieldDecrement({
  className,
  ...props
}: NumberFieldPrimitive.Decrement.Props): React.ReactElement {
  return (
    <NumberFieldPrimitive.Decrement
      className={cn(
        'relative flex shrink-0 cursor-pointer items-center justify-center rounded-s-[calc(var(--radius-lg)-1px)] px-[calc(--spacing(3)-1px)] transition-colors hover:bg-accent',
        className,
      )}
      data-slot="number-field-decrement"
      {...props}
    >
      <MinusIcon />
    </NumberFieldPrimitive.Decrement>
  )
}

export function NumberFieldIncrement({
  className,
  ...props
}: NumberFieldPrimitive.Increment.Props): React.ReactElement {
  return (
    <NumberFieldPrimitive.Increment
      className={cn(
        'relative flex shrink-0 cursor-pointer items-center justify-center rounded-e-[calc(var(--radius-lg)-1px)] px-[calc(--spacing(3)-1px)] transition-colors hover:bg-accent',
        className,
      )}
      data-slot="number-field-increment"
      {...props}
    >
      <PlusIcon />
    </NumberFieldPrimitive.Increment>
  )
}

export function NumberFieldInput({
  className,
  ...props
}: NumberFieldPrimitive.Input.Props): React.ReactElement {
  return (
    <NumberFieldPrimitive.Input
      className={cn(
        'h-8.5 w-full min-w-0 grow bg-transparent px-[calc(--spacing(3)-1px)] text-center tabular-nums leading-8.5 outline-none sm:h-7.5 sm:leading-7.5',
        className,
      )}
      data-slot="number-field-input"
      {...props}
    />
  )
}

export { NumberFieldPrimitive }
