import { expect, test } from './fixtures'
import { loginAsE2EAdmin } from './helpers'

test.describe('not found page', () => {
  test('unknown path shows 404 copy inside app shell', async ({ page }) => {
    await loginAsE2EAdmin(page)

    await page.goto('/does-not-exist')

    await expect(page).toHaveURL('/does-not-exist')
    await expect(page.getByRole('heading', { name: 'Page not found' })).toBeVisible()
    await expect(
      page.getByText("The page you're looking for doesn't exist or was moved."),
    ).toBeVisible()
    await expect(page.getByRole('link', { name: 'Go to dashboard' })).toBeVisible()
    await expect(page.locator('[data-slot="app-sidebar"]')).toBeVisible()
  })
})
