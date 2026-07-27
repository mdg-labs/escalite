'use client'

import type { ReactElement } from 'react'

import { Badge } from '../../primitives/badge'
import { TableCell, TableRow } from '../../primitives/table'
import { cn } from '../../lib/utils'
import type { AlertCardAlert, AlertCardLabels } from './types'
import {
  alertPriorityBadgeVariant,
  alertStatusBadgeVariant,
  alertStatusIndicatorClass,
  formatAlertCardTimestamp,
} from './utils'

export type AlertCardProps = {
  alert: AlertCardAlert
  labels: AlertCardLabels
  selected?: boolean
  onSelect?: () => void
  className?: string
}

export function AlertCard({
  alert,
  className,
  labels,
  onSelect,
  selected = false,
}: AlertCardProps): ReactElement {
  return (
    <TableRow
      className={cn(onSelect ? 'cursor-pointer' : undefined, className)}
      data-state={selected ? 'selected' : undefined}
      onClick={onSelect}
    >
      <TableCell className="max-w-xs truncate font-medium text-foreground">{alert.summary}</TableCell>
      <TableCell>
        <Badge variant={alertStatusBadgeVariant(alert.status)}>
          <span
            aria-hidden="true"
            className={cn('size-1.5 rounded-full', alertStatusIndicatorClass(alert.status))}
          />
          {labels.statusLabel(alert.status)}
        </Badge>
      </TableCell>
      <TableCell>
        <Badge variant={alertPriorityBadgeVariant(alert.priority)}>
          {labels.priorityLabel(alert.priority)}
        </Badge>
      </TableCell>
      <TableCell>
        {alert.eventCount > 1 ? (
          <Badge variant="outline">{alert.eventCount}</Badge>
        ) : (
          <span className="text-muted-foreground">1</span>
        )}
      </TableCell>
      <TableCell className="text-muted-foreground">
        {formatAlertCardTimestamp(alert.updatedAt)}
      </TableCell>
    </TableRow>
  )
}
