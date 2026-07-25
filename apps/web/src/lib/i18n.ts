const messages = {
  'alerts.title': 'Alerts',
  'alerts.empty': 'No alerts match this filter.',
  'alerts.filter.all': 'All',
  'alerts.filter.triggered': 'Triggered',
  'alerts.filter.acknowledged': 'Acknowledged',
  'alerts.filter.closed': 'Closed',
  'alerts.column.summary': 'Summary',
  'alerts.column.status': 'Status',
  'alerts.column.priority': 'Priority',
  'alerts.column.events': 'Events',
  'alerts.column.updated': 'Updated',
  'alerts.detail.select': 'Select an alert to view details.',
  'alerts.detail.timeline': 'Timeline',
  'alerts.detail.created': 'Created',
  'alerts.detail.acknowledged': 'Acknowledged',
  'alerts.detail.closed': 'Closed',
  'alerts.detail.dedupKey': 'Dedup key',
  'alerts.detail.serviceId': 'Service',
  'alerts.action.acknowledge': 'Acknowledge',
  'alerts.action.close': 'Close',
  'alerts.action.acknowledge.loading': 'Acknowledging…',
  'alerts.action.close.loading': 'Closing…',
  'alerts.status.triggered': 'Triggered',
  'alerts.status.acknowledged': 'Acknowledged',
  'alerts.status.closed': 'Closed',
  'alerts.priority.high': 'High',
  'alerts.priority.low': 'Low',
  'alerts.error.action': 'Action failed. Try again.',
  'nav.dashboard': 'Dashboard',
  'nav.alerts': 'Alerts',
  'nav.integrations': 'Integrations',
} as const

export type MessageKey = keyof typeof messages

export function t(key: MessageKey): string {
  return messages[key]
}
