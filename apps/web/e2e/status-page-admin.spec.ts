import { expect, test } from './fixtures'
import { e2eDefaultTeamName, loginAsE2EAdmin } from './helpers'

const statusPageOrigin =
  process.env.ESCALITE_STATUS_PAGE_ORIGIN ?? 'http://localhost:5174'

test.describe('status page admin to public', () => {
  test('enable page, add component, publish incident, and verify public view', async ({
    page,
  }) => {
    const slug = `e2e-admin-${Date.now()}`
    const pageTitle = 'E2E Status Page Admin'
    const componentName = 'Payment API'
    const incidentTitle = `E2E Incident ${Date.now()}`
    const publicUpdateBody = 'We are investigating elevated error rates.'

    await loginAsE2EAdmin(page)

    await page.goto('/status-pages')
    await expect(page.locator('[data-slot="frame-panel-title"]').filter({ hasText: 'Status pages' })).toBeVisible()

    await page.getByLabel('URL slug').fill(slug)
    await page.getByLabel('Page title').fill(pageTitle)
    await page.getByRole('checkbox', { name: 'Enable public status page' }).check()
    await page.getByRole('button', { name: 'Save settings' }).click()
    await expect(page.getByText('Status page saved.')).toBeVisible()

    await page.getByRole('tab', { name: 'Components' }).click()
    await page.getByRole('button', { name: 'Add component' }).click()
    const componentDialog = page.getByRole('dialog', { name: 'Add status page component' })
    await componentDialog.getByLabel('Name').fill(componentName)
    await componentDialog.getByRole('button', { name: 'Save' }).click()
    await expect(page.getByText('Component added.')).toBeVisible()
    await expect(page.getByRole('cell', { name: componentName }).first()).toBeVisible()

    await page.goto('/incidents')
    await page.getByRole('button', { name: 'New incident' }).first().click()
    const createDialog = page.getByRole('dialog', { name: 'Create incident' })
    await createDialog.getByLabel('Title').fill(incidentTitle)
    await createDialog.getByLabel('Team').click()
    await page.getByRole('option', { name: e2eDefaultTeamName }).click()
    await createDialog.getByRole('button', { name: 'Create incident' }).click()

    await expect(page).toHaveURL(/\/incidents\/[0-9a-f-]{36}$/i)
    await expect(page.getByRole('heading', { level: 2, name: incidentTitle })).toBeVisible()

    await page.getByRole('button', { name: 'Publish to status page' }).click()
    const publishDialog = page.getByRole('dialog', { name: 'Publish to status page' })
    await publishDialog.getByRole('checkbox', { name: componentName }).first().check()
    await publishDialog.getByLabel('Initial update').fill(publicUpdateBody)
    await publishDialog.getByRole('button', { name: 'Publish to status page' }).click()
    await expect(page.getByText('Published to status page.')).toBeVisible()

    await expect
      .poll(
        async () => {
          const response = await page.request.get(statusPageOrigin)
          return response.ok()
        },
        { timeout: 60_000 },
      )
      .toBe(true)

    await page.goto(`${statusPageOrigin}/${slug}`)
    await expect(page.getByRole('heading', { level: 1, name: pageTitle })).toBeVisible()
    await expect(page.getByText(componentName).first()).toBeVisible()
    await expect(page.getByRole('heading', { level: 2, name: 'Active incidents' })).toBeVisible()
    await expect(page.getByRole('heading', { level: 3, name: incidentTitle })).toBeVisible()
    await expect(page.getByText(publicUpdateBody)).toBeVisible()
  })
})
