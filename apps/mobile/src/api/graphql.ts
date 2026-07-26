import { getStoredRefreshToken } from '@/auth/secure-store'
import { getServerEndpoints } from '@/server/endpoints'

type GraphQLResponse<T> = {
  data?: T
  errors?: Array<{ message: string }>
}

export async function graphqlRequest<T>(
  query: string,
  variables?: Record<string, unknown>,
): Promise<T> {
  const refreshToken = await getStoredRefreshToken()
  if (!refreshToken) {
    throw new Error('not authenticated')
  }

  const { graphqlUrl } = getServerEndpoints()
  const response = await fetch(graphqlUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${refreshToken}`,
    },
    body: JSON.stringify({ query, variables }),
  })

  if (!response.ok) {
    throw new Error('graphql request failed')
  }

  const body = (await response.json()) as GraphQLResponse<T>
  if (body.errors?.length) {
    throw new Error(body.errors[0]?.message ?? 'graphql request failed')
  }

  if (!body.data) {
    throw new Error('graphql request failed')
  }

  return body.data
}
