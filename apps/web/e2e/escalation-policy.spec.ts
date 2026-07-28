import { expect, test } from './fixtures'
import {
  e2eAdminEmail,
  e2eDefaultTeamName,
  loginAsE2EAdmin,
  postGraphQL,
} from './helpers'

test.describe('escalation policies', () => {
  test('create policy with user target', async ({ page }) => {
    const serviceName = `E2E Escalation Service ${Date.now()}`
    const policyName = `E2E Escalation Policy ${Date.now()}`

    await loginAsE2EAdmin(page)

    const usersData = await postGraphQL<{
      organizationUsers: Array<{ id: string; email: string }>
    }>(
      page.request,
      `query OrganizationUsers {
        organizationUsers {
          id
          email
        }
      }`,
    )

    const adminUser = usersData.organizationUsers.find((user) => user.email === e2eAdminEmail)
    expect(adminUser).toBeTruthy()

    await page.goto('/services')
    await page.getByRole('button', { name: 'Create service' }).click()
    await page.getByLabel('Name').fill(serviceName)
    await page.getByLabel('Team').click()
    await page.getByRole('option', { name: e2eDefaultTeamName }).click()
    await page.getByRole('button', { name: 'Create service' }).click()

    const serviceLink = page.getByRole('link', { name: serviceName })
    await expect(serviceLink).toBeVisible()
    const serviceHref = await serviceLink.getAttribute('href')
    const serviceId = serviceHref?.split('/').pop() ?? ''
    expect(serviceId).toMatch(/^[0-9a-f-]{36}$/i)

    await page.goto(`/services/${serviceId}/escalation-policies/new?serviceId=${serviceId}`)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()

    await page.getByLabel('Policy name').fill(policyName)

    const userSelect = page.getByLabel('User')
    await userSelect.click()
    await page.getByRole('option', { name: e2eAdminEmail }).click()

    await page.getByRole('button', { name: 'Save policy' }).click()
    await expect(page.getByRole('status')).toContainText('Escalation policy created')

    const policiesData = await postGraphQL<{
      escalationPolicies: Array<{ id: string; name: string }>
    }>(
      page.request,
      `query EscalationPolicies($serviceId: ID!) {
        escalationPolicies(serviceId: $serviceId) {
          id
          name
        }
      }`,
      { serviceId },
    )

    const createdPolicy = policiesData.escalationPolicies.find((policy) => policy.name === policyName)
    expect(createdPolicy).toBeTruthy()

    const policyData = await postGraphQL<{
      escalationPolicy: {
        steps: Array<{
          targets: Array<{
            targetType: string
            userId?: string | null
          }>
        }>
      }
    }>(
      page.request,
      `query EscalationPolicy($id: ID!) {
        escalationPolicy(id: $id) {
          steps {
            targets {
              targetType
              userId
            }
          }
        }
      }`,
      { id: createdPolicy?.id },
    )

    const firstTarget = policyData.escalationPolicy.steps[0]?.targets[0]
    expect(firstTarget?.targetType).toBe('user')
    expect(firstTarget?.userId).toBe(adminUser?.id)
  })
})
