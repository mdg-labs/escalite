import type { APIRequestContext } from '@playwright/test'

export const e2eAdminEmail = 'admin@e2e.test'
export const e2eAdminPassword = 'correct-horse-battery-staple'
export const e2eOrganizationName = 'E2E Test Org'

export const apiBaseUrl = process.env.ESCALITE_API_URL ?? 'http://localhost:8080'

type GraphQLResponse<T> = {
  data?: T
  errors?: Array<{ message: string }>
}

export async function ensureE2EAdminSession(request: APIRequestContext): Promise<void> {
  const loginResponse = await request.post(`${apiBaseUrl}/api/v1/login`, {
    data: {
      email: e2eAdminEmail,
      password: e2eAdminPassword,
    },
  })

  if (loginResponse.ok()) {
    return
  }

  const setupResponse = await request.post(`${apiBaseUrl}/api/v1/setup`, {
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

export async function postGraphQL<T>(
  request: APIRequestContext,
  query: string,
  variables?: Record<string, unknown>,
): Promise<T> {
  const response = await request.post(`${apiBaseUrl}/graphql`, {
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

export type SeededPublicStatusPage = {
  slug: string
  title: string
  componentName: string
}

export async function seedPublicStatusPage(
  request: APIRequestContext,
): Promise<SeededPublicStatusPage> {
  await ensureE2EAdminSession(request)

  const slug = `e2e-public-${Date.now()}`
  const title = 'E2E Public Status'
  const componentName = 'API Gateway'

  await postGraphQL(
    request,
    `mutation SaveStatusPage($input: SaveStatusPageInput!) {
      saveStatusPage(input: $input) {
        slug
      }
    }`,
    {
      input: {
        slug,
        title,
        enabled: true,
      },
    },
  )

  await postGraphQL(
    request,
    `mutation CreateStatusPageComponent($input: CreateStatusPageComponentInput!) {
      createStatusPageComponent(input: $input) {
        id
      }
    }`,
    {
      input: {
        name: componentName,
        description: 'Primary API entrypoint',
        status: 'OPERATIONAL',
        position: 0,
      },
    },
  )

  return { slug, title, componentName }
}
