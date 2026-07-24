import { expect, test } from './fixtures'

test.describe('compose dev stack', () => {
  test('login page loads after API /healthz is ready', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
  })
})
