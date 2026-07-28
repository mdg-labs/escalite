import { expect, test } from './fixtures'
import { e2eAdminEmail, e2eAdminPassword } from './helpers'

test.describe('first-admin bootstrap', () => {
  test('logout, login, and reach dashboard', async ({ page }) => {
    await page.goto('/login')
    await page.getByLabel('Email').fill(e2eAdminEmail)
    await page.getByLabel('Password').fill(e2eAdminPassword)
    await page.getByRole('button', { name: 'Sign in' }).click()

    await expect(page).toHaveURL('/dashboard')
    await expect(page.getByRole('heading', { name: 'Dashboard', level: 1 })).toBeVisible()
    await expect(page.getByText(`Signed in as ${e2eAdminEmail}`)).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Open alerts', level: 2 })).toBeVisible()

    const logoutResponse = await page.request.post('/api/v1/logout')
    expect(logoutResponse.status()).toBe(204)

    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()

    await page.getByLabel('Email').fill(e2eAdminEmail)
    await page.getByLabel('Password').fill(e2eAdminPassword)
    await page.getByRole('button', { name: 'Sign in' }).click()

    await expect(page).toHaveURL('/dashboard')
    await expect(page.getByRole('heading', { name: 'Dashboard', level: 1 })).toBeVisible()
    await expect(page.getByText('On call now').first()).toBeVisible()
  })
})
