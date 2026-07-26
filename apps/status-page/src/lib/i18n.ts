const messages = {
  'statusPage.loading': 'Loading status…',
  'statusPage.notFound': 'Status page not found.',
  'statusPage.error.load': 'Unable to load status page. Try again later.',
  'statusPage.overall.title': 'Current status',
  'statusPage.overall.operational': 'All systems operational',
  'statusPage.overall.degraded': 'Degraded performance',
  'statusPage.overall.partial_outage': 'Partial outage',
  'statusPage.overall.major_outage': 'Major outage',
  'statusPage.components.title': 'Components',
  'statusPage.components.empty': 'No components configured.',
  'statusPage.incidents.title': 'Active incidents',
  'statusPage.incidents.empty': 'No active incidents.',
  'statusPage.incidents.updates': 'Updates',
  'statusPage.incidents.affected': 'Affected components',
  'statusPage.component.status.operational': 'Operational',
  'statusPage.component.status.degraded': 'Degraded',
  'statusPage.component.status.partial_outage': 'Partial outage',
  'statusPage.component.status.major_outage': 'Major outage',
  'statusPage.incident.status.investigating': 'Investigating',
  'statusPage.incident.status.identified': 'Identified',
  'statusPage.incident.status.monitoring': 'Monitoring',
  'statusPage.incident.status.resolved': 'Resolved',
  'statusPage.subscribe.title': 'Subscribe to updates',
  'statusPage.subscribe.description':
    'Receive email notifications when incidents are posted or updated.',
  'statusPage.subscribe.email': 'Email address',
  'statusPage.subscribe.emailPlaceholder': 'you@example.com',
  'statusPage.subscribe.submit': 'Subscribe',
  'statusPage.subscribe.submitting': 'Subscribing…',
  'statusPage.subscribe.success': 'You are subscribed to incident updates.',
  'statusPage.subscribe.error': 'Subscription failed. Check the email and try again.',
  'statusPage.landing.title': 'Escalite status page',
  'statusPage.landing.description':
    'Open a status page by slug, for example /acme-status.',
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
