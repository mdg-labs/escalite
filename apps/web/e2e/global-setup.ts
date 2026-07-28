import {
  apiBaseUrl,
  e2eAdminEmail,
  e2eAdminPassword,
  e2eOrganizationName,
} from './helpers'

async function waitForHealthz(): Promise<void> {
  const healthzUrl = `${apiBaseUrl.replace(/\/$/, '')}/healthz`
  const deadline = Date.now() + 60_000

  while (Date.now() < deadline) {
    try {
      const response = await fetch(healthzUrl)
      if (response.ok) {
        const body = (await response.json()) as { status?: string }
        if (body.status === 'ok') {
          return
        }
      }
    } catch {
      // API not ready yet.
    }

    await new Promise((resolve) => setTimeout(resolve, 1_000))
  }

  throw new Error(`API health check timed out: ${healthzUrl}`)
}

async function ensureE2EBootstrap(): Promise<void> {
  const loginResponse = await fetch(`${apiBaseUrl}/api/v1/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: e2eAdminEmail,
      password: e2eAdminPassword,
    }),
  })

  if (loginResponse.ok) {
    return
  }

  const setupResponse = await fetch(`${apiBaseUrl}/api/v1/setup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      organizationName: e2eOrganizationName,
      email: e2eAdminEmail,
      password: e2eAdminPassword,
    }),
  })

  if (!setupResponse.ok) {
    throw new Error(
      `Failed to bootstrap E2E organization (login ${loginResponse.status}, setup ${setupResponse.status})`,
    )
  }
}

export default async function globalSetup(): Promise<void> {
  await waitForHealthz()
  await ensureE2EBootstrap()
}
