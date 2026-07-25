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
  'schedule.pageTitle': 'Schedule',
  'schedule.pageDescription':
    'View on-call layers, rotation windows, and create overrides in your local timezone.',
  'schedule.notFound': 'Schedule not found.',
  'schedule.loading': 'Loading schedule…',
  'schedule.calendar.title': 'Schedule calendar',
  'schedule.timezone.label': 'Times shown in',
  'schedule.onCallNow': 'On call now',
  'schedule.layer': 'Layer {layer}',
  'schedule.overrides.title': 'Overrides',
  'schedule.overrides.empty': 'No overrides in this range.',
  'schedule.override.create': 'Create override',
  'schedule.override.forbidden':
    'Override creation requires org admin or team membership.',
  'schedule.override.rotation': 'Rotation layer',
  'schedule.override.user': 'Replacement user',
  'schedule.override.dateRange': 'Override window',
  'schedule.action.cancel': 'Cancel',
  'schedule.action.save': 'Save override',
  'schedule.action.delete': 'Delete override',
  'schedule.computedAt': 'Computed at',
} as const

export type MessageKey = keyof typeof messages

type InterpolationValues = Record<string, string>

export function t(key: MessageKey, values?: InterpolationValues): string {
  let message: string = messages[key]
  if (values) {
    for (const [token, value] of Object.entries(values)) {
      message = message.replace(`{${token}}`, value)
    }
  }
  return message
}
