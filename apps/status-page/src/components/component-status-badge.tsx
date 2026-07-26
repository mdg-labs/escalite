import type { ReactElement } from 'react'
import { Badge } from '@escalite/ui'

import { t } from '../lib/i18n'
import {
  componentStatusBadgeVariant,
  componentStatusIcon,
  componentStatusLabel,
  type ComponentStatus,
} from '../lib/status-page'

type ComponentStatusBadgeProps = {
  status: ComponentStatus
}

export function ComponentStatusBadge({ status }: ComponentStatusBadgeProps): ReactElement {
  const Icon = componentStatusIcon(status)

  return (
    <Badge className="gap-1" variant={componentStatusBadgeVariant(status)}>
      <Icon aria-hidden className="size-3.5" />
      {t(componentStatusLabel(status))}
    </Badge>
  )
}
