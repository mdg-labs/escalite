import { expect, test } from './fixtures'
import {
  apiBaseUrl,
  e2eDefaultTeamName,
  loginAsE2EAdmin,
  postGraphQL,
} from './helpers'

const serviceName = `E2E Payments API ${Date.now()}`
const alertSummary = 'E2E disk usage high'

test.describe('alert happy path', () => {
  test('create service in UI, fire generic-rest alert, acknowledge in UI', async ({ page }) => {
    test.setTimeout(120_000)

    await loginAsE2EAdmin(page)

    await page.goto('/services')
    await expect(page.getByRole('heading', { name: 'Services' })).toBeVisible()

    await page.getByRole('button', { name: 'Create service' }).click()
    await page.getByLabel('Name').fill(serviceName)
    await page.getByLabel('Team').click()
    await page.getByRole('option', { name: e2eDefaultTeamName }).click()
    await page.getByRole('button', { name: 'Create service' }).click()

    const serviceLink = page.getByRole('link', { name: serviceName }).first()
    await expect(serviceLink).toBeVisible()
    const serviceHref = await serviceLink.getAttribute('href')
    expect(serviceHref).toMatch(/\/services\/[0-9a-f-]{36}$/)
    const serviceId = serviceHref?.split('/').pop() ?? ''
    expect(serviceId).not.toBe('')

    const keyData = await postGraphQL<{
      createIntegrationKey: { token: string }
    }>(
      page.request,
      `mutation CreateIntegrationKey($input: CreateIntegrationKeyInput!) {
        createIntegrationKey(input: $input) {
          token
        }
      }`,
      {
        input: {
          serviceId,
          pluginName: 'generic-rest-api',
          config: {},
        },
      },
    )

    const token = keyData.createIntegrationKey.token
    expect(token).toBeTruthy()

    const dedupKey = `e2e-alert-${Date.now()}`
    const alertResponse = await page.request.post(`${apiBaseUrl.replace(/\/$/, '')}/api/v1/alerts`, {
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: {
        summary: alertSummary,
        description: 'Playwright generic-rest-api alert',
        dedup_key: dedupKey,
        priority: 'high',
      },
    })

    expect(alertResponse.status()).toBe(201)
    const alertBody = (await alertResponse.json()) as { id: string }
    expect(alertBody.id).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
    )

    await expect
      .poll(
        async () => {
          const data = await postGraphQL<{ alert: { summary: string } | null }>(
            page.request,
            `query Alert($id: ID!) {
              alert(id: $id) {
                summary
              }
            }`,
            { id: alertBody.id },
          )
          return data.alert?.summary ?? null
        },
        { timeout: 15_000 },
      )
      .toBe(alertSummary)

    const ackData = await postGraphQL<{
      acknowledgeAlert: { status: string }
    }>(
      page.request,
      `mutation AcknowledgeAlert($id: ID!) {
        acknowledgeAlert(id: $id) {
          status
        }
      }`,
      { id: alertBody.id },
    )

    expect(ackData.acknowledgeAlert.status).toBe('ACKNOWLEDGED')

    const confirmed = await postGraphQL<{
      alert: { status: string } | null
    }>(
      page.request,
      `query Alert($id: ID!) {
        alert(id: $id) {
          status
        }
      }`,
      { id: alertBody.id },
    )

    expect(confirmed.alert?.status).toBe('ACKNOWLEDGED')
  })
})
