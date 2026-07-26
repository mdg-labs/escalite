import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

test('normalizeServerOrigin trims input and requires http/https', () => {
  const source = readFileSync(join(root, 'src/server/url.ts'), 'utf8')
  assert.match(source, /normalizeServerOrigin/)
  assert.match(source, /https:\/\//)
})

test('deriveServerEndpoints maps graphql to origin/graphql', () => {
  const source = readFileSync(join(root, 'src/server/url.ts'), 'utf8')
  assert.match(source, /deriveServerEndpoints/)
  assert.match(source, /graphqlUrl: `\$\{normalizedOrigin\}\/graphql`/)
})

test('server origin is stored in SecureStore', () => {
  const source = readFileSync(join(root, 'src/server/store.ts'), 'utf8')
  assert.match(source, /expo-secure-store/)
  assert.match(source, /escalite\.mobile\.server_origin/)
})

test('auth config no longer hardcodes api base url', () => {
  const source = readFileSync(join(root, 'src/auth/config.ts'), 'utf8')
  assert.doesNotMatch(source, /apiBaseUrl/)
  assert.match(source, /mobileLoginUrl/)
})

test('graphql requests use runtime server endpoints', () => {
  const source = readFileSync(join(root, 'src/api/graphql.ts'), 'utf8')
  assert.match(source, /getServerEndpoints/)
})
