import type { AlertCardPriority, AlertCardStatus } from './types'

export function alertStatusIndicatorClass(status: AlertCardStatus): string {
  switch (status) {
    case 'Acknowledged':
      return 'bg-severity-info'
    case 'Closed':
      return 'bg-severity-resolved'
    case 'Triggered':
      return 'bg-severity-high'
    default:
      return 'bg-muted-foreground/64'
  }
}

export function alertStatusBadgeVariant(
  status: AlertCardStatus,
): 'warning' | 'info' | 'success' {
  switch (status) {
    case 'Acknowledged':
      return 'info'
    case 'Closed':
      return 'success'
    case 'Triggered':
      return 'warning'
    default:
      return 'warning'
  }
}

export function alertPriorityBadgeVariant(priority: AlertCardPriority): 'error' | 'secondary' {
  return priority === 'High' ? 'error' : 'secondary'
}

export function formatAlertCardTimestamp(value: string | null | undefined): string {
  if (!value) {
    return '—'
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}
