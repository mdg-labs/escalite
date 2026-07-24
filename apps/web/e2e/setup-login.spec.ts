import { expect, test } from './fixtures'

const adminEmail = 'admin@e2e.test'
const adminPassword = 'correct-horse-battery-staple'
const organizationName = 'E2E Test Org'

test.describe('first-admin bootstrap', () => {
  test('setup, logout, login, and reach dashboard', async ({ page }) => {
    await page.goto('/setup')
    await expect(page.getByRole('heading', { name: 'Set up Escalite' })).toBeVisible()

    await page.getByLabel('Organization name').fill(organizationName)
    await page.getByLabel('Admin email').fill(adminEmail)
    await page.getByLabel('Password').fill(adminPassword)
    await page.getByRole('button', { name: 'Create organization' }).click()

    await expect(page).toHaveURL('/dashboard')
    await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
    await expect(page.getByText(`Signed in as ${adminEmail}`)).toBeVisible()

    const logoutResponse = await page.request.post('/api/v1/logout')
    expect(logoutResponse.status()).toBe(204)

    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()

    await page.getByLabel('Email').fill(adminEmail)
    await page.getByLabel('Password').fill(adminPassword)
    await page.getByRole('button', { name: 'Sign in' }).click()

    await expect(page).toHaveURL('/dashboard')
    await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
  })
})
