import { expect, test } from './fixtures'
import {
  apiBaseUrl,
  e2eDefaultTeamName,
  loginAsE2EAdmin,
  postGraphQL,
} from './helpers'

const serviceName = 'E2E Payments API'
const alertSummary = 'E2E disk usage high'

test.describe('alert happy path', () => {
  test('create service in UI, fire generic-rest alert, acknowledge in UI', async ({ page }) => {
    await loginAsE2EAdmin(page)

    await page.goto('/services')
    await expect(page.getByRole('heading', { name: 'Services' })).toBeVisible()

    await page.getByRole('button', { name: 'Create service' }).click()
    await page.getByLabel('Name').fill(serviceName)
    await page.getByLabel('Team').click()
    await page.getByRole('option', { name: e2eDefaultTeamName }).click()
    await page.getByRole('button', { name: 'Create service' }).click()

    await expect(page.getByRole('link', { name: serviceName })).toBeVisible()

    const serviceLink = page.getByRole('link', { name: serviceName })
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

    await page.goto(`/alerts/${alertBody.id}`)
    await expect(page.getByRole('heading', { level: 2, name: alertSummary })).toBeVisible()
    await expect(page.getByText('Triggered', { exact: true }).first()).toBeVisible()

    await page.getByRole('button', { name: 'Acknowledge' }).first().click()
    await expect(page.getByText('Acknowledged', { exact: true }).first()).toBeVisible()
    await expect(page.getByRole('button', { name: 'Acknowledge' }).first()).toBeDisabled()
  })
})
