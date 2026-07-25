import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

test('push registration uses bearer GraphQL mutation', () => {
  const source = readFileSync(join(root, 'src/push/register-device.ts'), 'utf8')
  assert.match(source, /registerMobileDevice/)
  assert.match(source, /Authorization: `Bearer \$\{refreshToken\}`/)
  assert.match(source, /expo-notifications/)
})

test('auth context syncs push token after sign-in', () => {
  const source = readFileSync(join(root, 'src/auth/context.tsx'), 'utf8')
  assert.match(source, /syncMobileDevicePushToken/)
})
