import { test as base, expect } from '@playwright/test'

const appOrigin = process.env.ESCALITE_APP_ORIGIN ?? 'http://localhost:5173'
const apiBaseUrl = process.env.ESCALITE_API_URL ?? 'http://localhost:8080'
const healthzUrl = `${apiBaseUrl.replace(/\/$/, '')}/healthz`

export const test = base.extend({
  page: async ({ page, request }, use) => {
    await expect
      .poll(
        async () => {
          const response = await request.get(healthzUrl)
          if (!response.ok()) {
            return null
          }
          return response.json()
        },
        { timeout: 60_000 },
      )
      .toEqual({ status: 'ok' })

    await expect
      .poll(
        async () => {
          const response = await request.get(appOrigin)
          return response.ok()
        },
        { timeout: 60_000 },
      )
      .toBe(true)

    await use(page)
  },
})

export { expect } from '@playwright/test'
