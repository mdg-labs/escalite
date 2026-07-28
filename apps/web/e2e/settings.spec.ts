import { expect, test } from './fixtures'
import { loginAsE2EAdmin } from './helpers'

test.describe('settings', () => {
  test('navigate settings sections', async ({ page }) => {
    await loginAsE2EAdmin(page)

    await page.goto('/settings')
    await expect(page).toHaveURL('/settings/enterprise')
    await expect(page.getByRole('heading', { name: 'SAML single sign-on', level: 2 })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'SCIM provisioning', level: 2 })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Slack workspace', level: 2 })).toBeVisible()

    await page.getByRole('link', { name: 'Notifications' }).click()
    await expect(page).toHaveURL('/settings/notifications')
    await expect(
      page.getByRole('heading', { name: 'Notification contact methods', level: 2 }),
    ).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Notification rules', level: 2 })).toBeVisible()

    await page.goto('/settings/devices')
    await expect(page.getByRole('heading', { name: 'Mobile devices', level: 1 })).toBeVisible()

    await page.getByRole('link', { name: 'Roles' }).click()
    await expect(page).toHaveURL('/settings/roles')
    await expect(
      page.getByRole('heading', { name: 'Incident role definitions', level: 2 }),
    ).toBeVisible()

    await page.goto('/settings/enterprise')
    await expect(page.getByRole('link', { name: 'Enterprise' })).toHaveAttribute(
      'aria-current',
      'page',
    )
  })
})
