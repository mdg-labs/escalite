import { expect, test } from './fixtures'
import {
  apiBaseUrl,
  e2eAdminEmail,
  e2eDefaultTeamName,
  loginAsE2EAdmin,
  postGraphQL,
} from './helpers'

test.describe('escalation policies', () => {
  test('create policy with user target, trigger alert, and verify step 1 target', async ({
    page,
  }) => {
    test.setTimeout(120_000)

    const serviceName = `E2E Escalation Service ${Date.now()}`
    const policyName = `E2E Escalation Policy ${Date.now()}`
    const alertSummary = `E2E escalation alert ${Date.now()}`

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

    const userSelect = page
      .locator('label')
      .filter({ hasText: /^User$/ })
      .locator('..')
      .getByRole('combobox')
    await userSelect.click()
    await page.getByRole('option', { name: e2eAdminEmail }).click()

    await page.getByRole('button', { name: 'Save policy' }).click()
    await expect(page.getByText('Escalation policy created').first()).toBeVisible()

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
          stepOrder: number
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
            stepOrder
            targets {
              targetType
              userId
            }
          }
        }
      }`,
      { id: createdPolicy?.id },
    )

    const step1 = policyData.escalationPolicy.steps.find((step) => step.stepOrder === 1)
    const firstTarget = step1?.targets[0]
    expect(firstTarget?.targetType).toBe('user')
    expect(firstTarget?.userId).toBe(adminUser?.id)

    const keyData = await postGraphQL<{
      createIntegrationKey: { token: string }
    }>(
      page.request,
      `mutation CreateIntegrationKey($input: CreateIntegrationKeyInput!) {
        createIntegrationKey(input: $input) {
          token
        }
      }`,
      {
        input: {
          serviceId,
          pluginName: 'generic-rest-api',
          config: {},
        },
      },
    )

    const token = keyData.createIntegrationKey.token
    expect(token).toBeTruthy()

    const dedupKey = `e2e-escalation-${Date.now()}`
    const alertResponse = await page.request.post(`${apiBaseUrl.replace(/\/$/, '')}/api/v1/alerts`, {
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: {
        summary: alertSummary,
        description: 'Playwright escalation policy alert',
        dedup_key: dedupKey,
        priority: 'high',
      },
    })

    expect(alertResponse.status()).toBe(201)
    const alertBody = (await alertResponse.json()) as { id: string }
    expect(alertBody.id).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
    )

    await expect
      .poll(
        async () => {
          const data = await postGraphQL<{ alert: { summary: string } | null }>(
            page.request,
            `query Alert($id: ID!) {
              alert(id: $id) {
                summary
              }
            }`,
            { id: alertBody.id },
          )
          return data.alert?.summary ?? null
        },
        { timeout: 30_000 },
      )
      .toBe(alertSummary)
  })
})
