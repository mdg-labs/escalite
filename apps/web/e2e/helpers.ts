import type { APIRequestContext, Page } from '@playwright/test'

export const e2eAdminEmail = 'admin@e2e.test'
export const e2eAdminPassword = 'correct-horse-battery-staple'
export const e2eOrganizationName = 'E2E Test Org'
export const e2eDefaultTeamName = 'Default'

export const apiBaseUrl = process.env.ESCALITE_API_URL ?? 'http://localhost:8080'

export async function loginAsE2EAdmin(page: Page): Promise<void> {
  await page.goto('/login')
  await page.getByLabel('Email').fill(e2eAdminEmail)
  await page.getByLabel('Password').fill(e2eAdminPassword)
  await page.getByRole('button', { name: 'Sign in' }).click()
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
