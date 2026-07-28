import type { APIRequestContext, Page } from '@playwright/test'

export const e2eAdminEmail = 'admin@e2e.test'
export const e2eAdminPassword = 'correct-horse-battery-staple'
export const e2eOrganizationName = 'E2E Test Org'
export const e2eDefaultTeamName = 'Default'

export const appOrigin = process.env.ESCALITE_APP_ORIGIN ?? 'http://localhost:5173'
export const apiBaseUrl = process.env.ESCALITE_API_URL ?? 'http://localhost:8080'

export async function ensureE2EAdminSession(request: APIRequestContext): Promise<void> {
  const loginResponse = await request.post(`${appOrigin.replace(/\/$/, '')}/api/v1/login`, {
    data: {
      email: e2eAdminEmail,
      password: e2eAdminPassword,
    },
  })

  if (loginResponse.ok()) {
    return
  }

  const setupResponse = await request.post(`${appOrigin.replace(/\/$/, '')}/api/v1/setup`, {
    data: {
      organizationName: e2eOrganizationName,
      email: e2eAdminEmail,
      password: e2eAdminPassword,
    },
  })

  if (!setupResponse.ok()) {
    throw new Error(
      `Failed to bootstrap E2E organization (login ${loginResponse.status()}, setup ${setupResponse.status()})`,
    )
  }
}

export async function loginAsE2EAdmin(page: Page): Promise<void> {
  await ensureE2EAdminSession(page.request)
  await page.goto('/dashboard')
  await page.waitForURL('/dashboard')
}

type GraphQLResponse<T> = {
  data?: T
  errors?: Array<{ message: string }>
}

export async function postGraphQL<T>(
  request: APIRequestContext,
  query: string,
  variables?: Record<string, unknown>,
): Promise<T> {
  const response = await request.post('/graphql', {
    data: {
      query,
      variables,
    },
  })

  if (!response.ok()) {
    throw new Error(`GraphQL request failed with status ${response.status()}`)
  }

  const body = (await response.json()) as GraphQLResponse<T>
  if (body.errors?.length) {
    throw new Error(body.errors.map((error) => error.message).join('; '))
  }

  if (!body.data) {
    throw new Error('GraphQL response missing data')
  }

  return body.data
}
