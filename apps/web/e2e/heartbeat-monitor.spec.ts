import { expect, test } from './fixtures'
import { e2eDefaultTeamName, loginAsE2EAdmin, postGraphQL } from './helpers'

test.describe('heartbeat monitors', () => {
  test('create heartbeat monitor on service detail', async ({ page }) => {
    const serviceName = `E2E Heartbeat Service ${Date.now()}`
    const monitorName = `E2E Heartbeat ${Date.now()}`

    await loginAsE2EAdmin(page)

    await page.goto('/services')
    await page.getByRole('button', { name: 'Create service' }).click()
    await page.getByLabel('Name').fill(serviceName)
    await page.getByLabel('Team').click()
    await page.getByRole('option', { name: e2eDefaultTeamName }).click()
    await page.getByRole('button', { name: 'Create service' }).click()

    const serviceLink = page.getByRole('link', { name: serviceName })
    await expect(serviceLink).toBeVisible()
    const serviceHref = await serviceLink.getAttribute('href')
    const serviceId = serviceHref?.split('/').pop() ?? ''
    expect(serviceId).toMatch(/^[0-9a-f-]{36}$/i)

    await page.goto(`/services/${serviceId}`)
    await page.getByRole('tab', { name: 'Heartbeats' }).click()
    await expect(page.getByRole('heading', { name: 'Heartbeat monitors', level: 2 })).toBeVisible()

    await page.getByRole('button', { name: 'Add monitor' }).click()
    await page.getByLabel('Name').fill(monitorName)
    await page.getByRole('button', { name: 'Create monitor' }).click()

    await expect(
      page.getByRole('dialog').getByRole('heading', { name: 'Heartbeat monitor created', exact: true }),
    ).toBeVisible()
    await expect(page.getByText(/\/heartbeat\//)).toBeVisible()
    await page.getByRole('button', { name: 'I saved the URL' }).click()

    await expect(page.getByRole('cell', { name: monitorName })).toBeVisible()
    await expect(page.getByText('Healthy')).toBeVisible()

    const monitorsData = await postGraphQL<{
      heartbeatMonitors: Array<{ id: string; name: string; token?: string | null }>
    }>(
      page.request,
      `query HeartbeatMonitors($serviceId: ID!) {
        heartbeatMonitors(serviceId: $serviceId) {
          id
          name
          token
        }
      }`,
      { serviceId },
    )

    const createdMonitor = monitorsData.heartbeatMonitors.find((monitor) => monitor.name === monitorName)
    expect(createdMonitor).toBeTruthy()
    expect(createdMonitor?.token).toBeFalsy()
  })
})
