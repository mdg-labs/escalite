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
): 'severity-high' | 'severity-info' | 'severity-resolved' {
  switch (status) {
    case 'Acknowledged':
      return 'severity-info'
    case 'Closed':
      return 'severity-resolved'
    case 'Triggered':
      return 'severity-high'
    default:
      return 'severity-high'
  }
}

export function alertPriorityBadgeVariant(
  priority: AlertCardPriority,
): 'severity-critical' | 'severity-low' {
  return priority === 'High' ? 'severity-critical' : 'severity-low'
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
