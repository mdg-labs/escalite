import { AlertPriority, AlertStatus, type AlertFieldsFragment } from '@escalite/ts-types'

import { t } from './i18n'

export function alertStatusLabel(status: AlertStatus): string {
  switch (status) {
    case AlertStatus.Acknowledged:
      return t('alerts.status.acknowledged')
    case AlertStatus.Closed:
      return t('alerts.status.closed')
    default:
      return t('alerts.status.triggered')
  }
}

export function alertPriorityLabel(priority: AlertPriority): string {
  return priority === AlertPriority.Low ? t('alerts.priority.low') : t('alerts.priority.high')
}

export function alertStatusBadgeVariant(
  status: AlertStatus,
): 'severity-high' | 'severity-info' | 'severity-resolved' {
  switch (status) {
    case AlertStatus.Acknowledged:
      return 'severity-info'
    case AlertStatus.Closed:
      return 'severity-resolved'
    default:
      return 'severity-high'
  }
}

export function alertPriorityBadgeVariant(
  priority: AlertPriority,
): 'severity-critical' | 'severity-low' {
  return priority === AlertPriority.High ? 'severity-critical' : 'severity-low'
}

export function formatAlertTimestamp(value: string | null | undefined): string {
  if (!value) {
    return '—'
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

export function mergeAlertUpdate(
  alerts: AlertFieldsFragment[],
  updated: AlertFieldsFragment,
): AlertFieldsFragment[] {
  const index = alerts.findIndex((alert) => alert.id === updated.id)
  if (index === -1) {
    return [updated, ...alerts]
  }
  const next = [...alerts]
  next[index] = updated
  return next
}

export function sortAlertsByUpdatedAt(alerts: AlertFieldsFragment[]): AlertFieldsFragment[] {
  return [...alerts].sort(
    (left, right) => new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime(),
  )
}
