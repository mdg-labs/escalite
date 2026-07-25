import type { AlertPriority } from '@escalite/ts-types'
import { Badge } from '@escalite/ui'
import type { ReactElement } from 'react'

import { alertPriorityBadgeVariant, alertPriorityLabel } from '../lib/alerts'

type AlertPriorityBadgeProps = {
  priority: AlertPriority
}

export function AlertPriorityBadge({ priority }: AlertPriorityBadgeProps): ReactElement {
  return (
    <Badge variant={alertPriorityBadgeVariant(priority)}>{alertPriorityLabel(priority)}</Badge>
  )
}
