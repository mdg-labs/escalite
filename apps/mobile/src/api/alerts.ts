import { mobileAuthConfig } from '@/auth/config'
import { getStoredRefreshToken } from '@/auth/secure-store'

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

type AlertQueryResponse = {
  data?: {
    alert?: MobileAlert | null
  }
  errors?: Array<{ message: string }>
}

export async function fetchAlert(alertId: string): Promise<MobileAlert | null> {
  const refreshToken = await getStoredRefreshToken()
  if (!refreshToken) {
    return null
  }

  const response = await fetch(`${mobileAuthConfig.apiBaseUrl}/graphql`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${refreshToken}`,
    },
    body: JSON.stringify({
      query: `query Alert($id: ID!) {
        alert(id: $id) {
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
        }
      }`,
      variables: { id: alertId },
    }),
  })

  if (!response.ok) {
    throw new Error('alert request failed')
  }

  const body = (await response.json()) as AlertQueryResponse
  if (body.errors?.length) {
    throw new Error(body.errors[0]?.message ?? 'alert request failed')
  }

  return body.data?.alert ?? null
}
