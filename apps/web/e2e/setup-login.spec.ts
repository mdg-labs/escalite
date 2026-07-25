import { expect, test } from './fixtures'
import {
  e2eAdminEmail,
  e2eAdminPassword,
  e2eOrganizationName,
} from './helpers'

test.describe('first-admin bootstrap', () => {
  test('setup, logout, login, and reach dashboard', async ({ page }) => {
    await page.goto('/setup')
    await expect(page.getByRole('heading', { name: 'Set up Escalite' })).toBeVisible()

    await page.getByLabel('Organization name').fill(e2eOrganizationName)
    await page.getByLabel('Admin email').fill(e2eAdminEmail)
    await page.getByLabel('Password').fill(e2eAdminPassword)
    await page.getByRole('button', { name: 'Create organization' }).click()

    await expect(page).toHaveURL('/dashboard')
    await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
    await expect(page.getByText(`Signed in as ${e2eAdminEmail}`)).toBeVisible()

    const logoutResponse = await page.request.post('/api/v1/logout')
    expect(logoutResponse.status()).toBe(204)

    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()

    await page.getByLabel('Email').fill(e2eAdminEmail)
    await page.getByLabel('Password').fill(e2eAdminPassword)
    await page.getByRole('button', { name: 'Sign in' }).click()

    await expect(page).toHaveURL('/dashboard')
    await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
  })
})
