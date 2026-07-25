export const ALERT_TRIGGERED_TYPE = 'alert.triggered' as const

export type AlertTriggeredPushData = {
  type: typeof ALERT_TRIGGERED_TYPE
  alertId: string
  serviceId: string
  priority: string
  title: string
  body: string
  actions?: string[]
  critical?: boolean
}

function readString(value: unknown): string | null {
  if (typeof value === 'string' && value.length > 0) {
    return value
  }
  return null
}

export function parseAlertTriggeredPushData(data: unknown): AlertTriggeredPushData | null {
  if (!data || typeof data !== 'object') {
    return null
  }

  const record = data as Record<string, unknown>
  if (record.type !== ALERT_TRIGGERED_TYPE) {
    return null
  }

  const alertId = readString(record.alertId)
  const serviceId = readString(record.serviceId)
  const priority = readString(record.priority)
  const title = readString(record.title)
  const body = readString(record.body)

  if (!alertId || !serviceId || !priority || !title || !body) {
    return null
  }

  const actions = Array.isArray(record.actions)
    ? record.actions.filter((action): action is string => typeof action === 'string')
    : undefined

  return {
    type: ALERT_TRIGGERED_TYPE,
    alertId,
    serviceId,
    priority,
    title,
    body,
    actions,
    critical: record.critical === true,
  }
}
