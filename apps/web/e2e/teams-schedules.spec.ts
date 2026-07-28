import { expect, test } from './fixtures'
import { e2eAdminEmail, loginAsE2EAdmin, postGraphQL } from './helpers'

test.describe('teams and schedules', () => {
  test('create team, schedule, and override', async ({ page }) => {
    const teamName = `E2E Team Schedules ${Date.now()}`
    const scheduleName = `E2E Schedule ${Date.now()}`
    const rotationName = `E2E Rotation ${Date.now()}`

    await loginAsE2EAdmin(page)

    await page.goto('/teams')
    await expect(page.getByRole('heading', { name: 'Teams', level: 1 })).toBeVisible()

    await page.getByRole('button', { name: 'Create team' }).click()
    await page.getByLabel('Name').fill(teamName)
    await page.getByRole('button', { name: 'Create team' }).click()
    await expect(page.getByRole('cell', { name: teamName })).toBeVisible()

    await page.goto('/schedules')
    await expect(page.getByRole('heading', { name: 'Schedules', level: 1 })).toBeVisible()

    await page.getByRole('button', { name: 'Create schedule' }).click()
    await page.getByLabel('Name').fill(scheduleName)
    await page.getByLabel('Team').click()
    await page.getByRole('option', { name: teamName }).click()
    await page.getByRole('button', { name: 'Create schedule' }).click()

    await expect(page).toHaveURL(/\/schedules\/[0-9a-f-]{36}$/i)
    await expect(page.getByRole('heading', { level: 1, name: scheduleName })).toBeVisible()

    const scheduleId = page.url().split('/').pop() ?? ''
    expect(scheduleId).toMatch(/^[0-9a-f-]{36}$/i)

    await page.getByRole('button', { name: 'Add rotation' }).click()
    const rotationDialog = page.getByRole('dialog', { name: 'Add rotation' })
    await rotationDialog.getByLabel('Rotation name').fill(rotationName)
    await rotationDialog.getByRole('checkbox', { name: e2eAdminEmail }).check()
    await rotationDialog.getByRole('button', { name: 'Save override' }).click()
    await expect(page.getByRole('status')).toContainText('Rotation added.')
    await expect(page.getByText(rotationName)).toBeVisible()

    await page.getByRole('button', { name: 'Create override' }).click()
    const overrideDialog = page.getByRole('dialog', { name: 'Create override' })
    await overrideDialog.getByPlaceholder('Replacement user').fill(e2eAdminEmail)
    await page.getByRole('option', { name: e2eAdminEmail }).click()
    await overrideDialog.getByRole('button', { name: 'Override window' }).click()

    const calendar = page.locator('[data-slot="calendar"]').first()
    const enabledDayButtons = calendar.locator('button:not([disabled])')
    await enabledDayButtons.nth(10).click()
    await enabledDayButtons.nth(12).click()

    await overrideDialog.getByRole('button', { name: 'Save override' }).click()
    await expect(page.getByRole('status')).toContainText('Override added.')
    await expect(page.getByText('Overrides')).toBeVisible()
    await expect(page.getByText(e2eAdminEmail).first()).toBeVisible()

    const overridesData = await postGraphQL<{
      overrides: Array<{ id: string; userId: string; scheduleId: string }>
    }>(
      page.request,
      `query ScheduleOverrides($scheduleId: ID!) {
        overrides(scheduleId: $scheduleId) {
          id
          userId
          scheduleId
        }
      }`,
      { scheduleId },
    )

    expect(overridesData.overrides.length).toBeGreaterThan(0)
    expect(overridesData.overrides.every((override) => override.scheduleId === scheduleId)).toBe(
      true,
    )
  })
})
