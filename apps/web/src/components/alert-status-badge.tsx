import type { AlertStatus } from '@escalite/ts-types'
import { Badge } from '@escalite/ui'
import type { ReactElement } from 'react'

import { alertStatusBadgeVariant, alertStatusLabel } from '../lib/alerts'

type AlertStatusBadgeProps = {
  status: AlertStatus
}

export function AlertStatusBadge({ status }: AlertStatusBadgeProps): ReactElement {
  return <Badge variant={alertStatusBadgeVariant(status)}>{alertStatusLabel(status)}</Badge>
}
