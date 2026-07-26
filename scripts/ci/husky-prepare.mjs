#!/usr/bin/env node

import { execSync } from 'node:child_process'

const isCi =
  process.env.CI === '1' ||
  process.env.CI === 'true' ||
  process.env.EAS_BUILD === 'true' ||
  process.env.HUSKY === '0'

if (isCi) {
  process.exit(0)
}

try {
  execSync('husky', { stdio: 'inherit' })
} catch (error) {
  // Husky is a root devDependency; skip when it is not installed.
  if (error?.status === 127) {
    process.exit(0)
  }
  throw error
}
