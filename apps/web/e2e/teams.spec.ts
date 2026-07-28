import { expect, test } from './fixtures'
import { loginAsE2EAdmin, postGraphQL } from './helpers'

test.describe('teams', () => {
  test('create team from teams page', async ({ page }) => {
    const teamName = `E2E Team ${Date.now()}`
    await loginAsE2EAdmin(page)

    await page.goto('/teams')
    await expect(page.getByRole('heading', { name: 'Teams', level: 1 })).toBeVisible()

    await page.getByRole('button', { name: 'Create team' }).click()
    await page.getByLabel('Name').fill(teamName)
    await page.getByRole('button', { name: 'Create team' }).click()

    await expect(page.getByRole('cell', { name: teamName })).toBeVisible()
  })

  test('add member on team detail page', async ({ page }) => {
    const teamName = `E2E Team Members ${Date.now()}`
    const memberEmail = `e2e-member-${Date.now()}@e2e.test`
    await loginAsE2EAdmin(page)

    const inviteResult = await postGraphQL<{
      inviteUser: { id: string; email: string }
    }>(
      page.request,
      `mutation InviteUser($input: InviteUserInput!) {
        inviteUser(input: $input) {
          id
          email
        }
      }`,
      { input: { email: memberEmail, role: 'MEMBER' } },
    )

    await page.goto('/teams')
    await page.getByRole('button', { name: 'Create team' }).click()
    await page.getByLabel('Name').fill(teamName)
    await page.getByRole('button', { name: 'Create team' }).click()
    await expect(page.getByRole('cell', { name: teamName })).toBeVisible()

    await page.getByRole('link', { name: teamName }).click()
    await expect(page.getByRole('heading', { name: 'Members', level: 2 })).toBeVisible()

    await page.getByLabel('Add member').fill(memberEmail)
    await page.getByRole('option', { name: memberEmail }).click()
    await page.getByRole('button', { name: 'Add member' }).click()

    await expect(page.getByRole('cell', { name: memberEmail }).first()).toBeVisible()
    expect(inviteResult.inviteUser.email).toBe(memberEmail)
  })
})
