import { graphqlRequest } from '@/api/graphql'

export type MobileAlert = {
  id: string
  organizationId: string
  serviceId: string
  status: string
  summary: string
  description: string
  priority: string
  createdAt: string
  acknowledgedAt: string | null
  closedAt: string | null
}

const alertFields = `
  id
  organizationId
  serviceId
  status
  summary
  description
  priority
  createdAt
  acknowledgedAt
  closedAt
`

export async function fetchAlert(alertId: string): Promise<MobileAlert | null> {
  const data = await graphqlRequest<{ alert?: MobileAlert | null }>(
    `query Alert($id: ID!) {
      alert(id: $id) {
        ${alertFields}
      }
    }`,
    { id: alertId },
  )

  return data.alert ?? null
}

export async function acknowledgeAlert(alertId: string): Promise<MobileAlert> {
  const data = await graphqlRequest<{ acknowledgeAlert: MobileAlert }>(
    `mutation AcknowledgeAlert($id: ID!) {
      acknowledgeAlert(id: $id) {
        ${alertFields}
      }
    }`,
    { id: alertId },
  )

  return data.acknowledgeAlert
}

export async function reEscalateAlert(alertId: string): Promise<MobileAlert> {
  const data = await graphqlRequest<{ reEscalateAlert: MobileAlert }>(
    `mutation ReEscalateAlert($id: ID!) {
      reEscalateAlert(id: $id) {
        ${alertFields}
      }
    }`,
    { id: alertId },
  )

  return data.reEscalateAlert
}

export async function snoozeAlert(alertId: string, durationMinutes: number): Promise<MobileAlert> {
  const data = await graphqlRequest<{ snoozeAlert: MobileAlert }>(
    `mutation SnoozeAlert($id: ID!, $durationMinutes: Int!) {
      snoozeAlert(id: $id, durationMinutes: $durationMinutes) {
        ${alertFields}
      }
    }`,
    { id: alertId, durationMinutes },
  )

  return data.snoozeAlert
}

export function canAcknowledgeAlert(alert: MobileAlert): boolean {
  return alert.status.toLowerCase() === 'triggered'
}

export function canEscalateAlert(alert: MobileAlert): boolean {
  const status = alert.status.toLowerCase()
  return status === 'triggered' || status === 'acknowledged'
}
