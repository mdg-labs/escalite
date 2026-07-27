import { expect, test } from './fixtures'
import { loginAsE2EAdmin } from './helpers'

const teamName = `E2E Team ${Date.now()}`

test.describe('teams', () => {
  test('create team from teams page', async ({ page }) => {
    await loginAsE2EAdmin(page)

    await page.goto('/teams')
    await expect(page.getByRole('heading', { name: 'Teams', level: 1 })).toBeVisible()

    await page.getByRole('button', { name: 'Create team' }).click()
    await page.getByLabel('Name').fill(teamName)
    await page.getByRole('button', { name: 'Create team' }).click()

    await expect(page.getByRole('cell', { name: teamName })).toBeVisible()
  })
})
