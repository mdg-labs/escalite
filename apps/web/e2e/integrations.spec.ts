import { expect, test } from './fixtures'
import { e2eDefaultTeamName, loginAsE2EAdmin } from './helpers'

test.describe('integrations', () => {
  test('integrations tab on service detail', async ({ page }) => {
    const serviceName = `E2E Integration Service ${Date.now()}`

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

    await page.goto(`/services/${serviceId}?tab=integrations`)
    await expect(page.getByRole('heading', { name: 'Integration keys', level: 2 })).toBeVisible()
    await expect(page.getByText('No integration keys for this service yet.')).toBeVisible()

    await page.getByRole('tab', { name: 'Integrations' }).click()
    await expect(page).toHaveURL(`/services/${serviceId}?tab=integrations`)
  })
})
