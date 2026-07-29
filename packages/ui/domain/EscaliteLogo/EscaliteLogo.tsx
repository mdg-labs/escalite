import type { ReactElement } from 'react'

import { cn } from '../../lib/utils'

export type EscaliteLogoVariant = 'mark' | 'mark-lg' | 'app-icon'

const VARIANT_SRC: Record<EscaliteLogoVariant, string> = {
  mark: '/branding/logo.svg',
  'mark-lg': '/branding/logo-big.svg',
  'app-icon': '/branding/app-icon.svg',
}

const VARIANT_SIZE: Record<EscaliteLogoVariant, string> = {
  mark: 'h-7 w-7',
  'mark-lg': 'h-16 w-16',
  'app-icon': 'h-10 w-10',
}

type EscaliteLogoProps = {
  variant?: EscaliteLogoVariant
  className?: string
  /** Defaults to decorative when used beside visible “Escalite” text. */
  alt?: string
}

export function EscaliteLogo({
  variant = 'mark',
  className,
  alt = '',
}: EscaliteLogoProps): ReactElement {
  return (
    <img
      alt={alt}
      className={cn('shrink-0', VARIANT_SIZE[variant], className)}
      src={VARIANT_SRC[variant]}
    />
  )
}
