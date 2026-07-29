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

test('resolveServerEndpoints reads runtime-config api and graphql URLs', () => {
  const source = readFileSync(join(root, 'src/server/url.ts'), 'utf8')
  assert.match(source, /resolveServerEndpoints/)
  assert.match(source, /resolveApiPublicUrlFrom/)
  assert.match(source, /resolveGraphqlUrlFrom/)
})

test('server setup fetches runtime-config.js from the web origin', () => {
  const runtimeSource = readFileSync(join(root, 'src/server/runtime-config.ts'), 'utf8')
  assert.match(runtimeSource, /runtime-config\.js/)
  const contextSource = readFileSync(join(root, 'src/server/context.tsx'), 'utf8')
  assert.match(contextSource, /fetchEscaliteRuntimeConfig/)
})

test('server health probe targets the API origin', () => {
  const healthSource = readFileSync(join(root, 'src/server/health.ts'), 'utf8')
  assert.match(healthSource, /probeServerHealth\(apiBaseUrl/)
  const contextSource = readFileSync(join(root, 'src/server/context.tsx'), 'utf8')
  assert.match(contextSource, /probeServerHealth\(nextEndpoints\.apiBaseUrl\)/)
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
