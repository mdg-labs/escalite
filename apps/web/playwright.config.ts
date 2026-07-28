import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig, devices } from '@playwright/test'

const appOrigin = process.env.ESCALITE_APP_ORIGIN ?? 'http://localhost:5173'
const configDir = path.dirname(fileURLToPath(import.meta.url))
const allureResultsDir = path.resolve(
  configDir,
  '../../allure-results/e2e/web',
)

const allureReporter = [
  'allure-playwright',
  { resultsDir: allureResultsDir },
] as const

export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.ts',
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  outputDir: 'test-results',
  reporter: process.env.CI
    ? [
        ['github'],
        ['html', { open: 'never', outputFolder: 'playwright-report' }],
        allureReporter,
      ]
    : [['list'], allureReporter],
  use: {
    baseURL: appOrigin,
    trace: process.env.CI ? 'retain-on-failure' : 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    headless: true,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
