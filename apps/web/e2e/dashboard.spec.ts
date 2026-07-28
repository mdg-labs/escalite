import { expect, test } from './fixtures'
import { loginAsE2EAdmin } from './helpers'

test.describe('dashboard', () => {
  test('renders alert summary and on-call widget after login', async ({ page }) => {
    await loginAsE2EAdmin(page)

    await expect(page.getByRole('heading', { name: 'Dashboard', level: 1 })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Open alerts', level: 2 })).toBeVisible()
    await expect(page.getByText('View triggered alerts')).toBeVisible()
    await expect(page.getByText('View acknowledged alerts')).toBeVisible()
    await expect(page.getByText('On call now').first()).toBeVisible()
  })
})
