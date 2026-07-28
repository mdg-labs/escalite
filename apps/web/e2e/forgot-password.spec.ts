import { expect, test } from './fixtures'
import { e2eAdminEmail } from './helpers'

test.describe('password reset request', () => {
  test('login links to forgot password and submits reset request', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()

    await page.getByRole('link', { name: 'Forgot password?' }).click()
    await expect(page).toHaveURL('/forgot-password')
    await expect(page.getByRole('heading', { name: 'Reset your password' })).toBeVisible()

    await page.getByLabel('Email').fill(e2eAdminEmail)
    await page.getByRole('button', { name: 'Send reset link' }).click()

    await expect(
      page.getByText(
        'If an account exists for that email, you will receive a password reset link shortly.',
      ),
    ).toBeVisible()
    await expect(page.getByRole('link', { name: 'Back to sign in' })).toBeVisible()
  })
})
