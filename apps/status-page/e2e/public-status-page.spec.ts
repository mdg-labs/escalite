import { expect, test } from './fixtures'
import { seedPublicStatusPage } from './helpers'

test.describe('public status page', () => {
  test('loads slug, shows components, and subscribes email', async ({ page }) => {
    const { slug, title, componentName } = await seedPublicStatusPage(page.request)
    const subscriberEmail = `status-page-e2e-${Date.now()}@e2e.test`

    await page.goto(`/${slug}`)

    await expect(page.getByRole('heading', { level: 1, name: title })).toBeVisible()
    await expect(page.getByRole('heading', { level: 2, name: 'Current status' })).toBeVisible()
    await expect(page.getByText('All systems operational')).toBeVisible()
    await expect(page.getByRole('heading', { level: 2, name: 'Components' })).toBeVisible()
    await expect(page.getByText(componentName)).toBeVisible()
    await expect(page.getByRole('heading', { level: 2, name: 'Active incidents' })).toBeVisible()
    await expect(page.getByText('No active incidents.')).toBeVisible()

    await expect(page.getByRole('heading', { level: 2, name: 'Subscribe to updates' })).toBeVisible()
    await page.getByLabel('Email address').fill(subscriberEmail)
    await page.getByRole('button', { name: 'Subscribe' }).click()

    await expect(
      page.getByText('You are subscribed to incident updates.'),
    ).toBeVisible()
  })
})
