import assert from 'node:assert/strict'
import test from 'node:test'

import {
  resolveApiPublicUrlFrom,
  resolveGraphqlUrlFrom,
  resolveStatusPollIntervalMsFrom,
} from '../src/index.ts'

test('resolveGraphqlUrlFrom prefers runtime over vite and default', () => {
  assert.equal(
    resolveGraphqlUrlFrom({ graphqlUrl: 'https://api.example.com/graphql' }),
    'https://api.example.com/graphql',
  )
  assert.equal(resolveGraphqlUrlFrom(undefined, 'https://build.example/graphql'), 'https://build.example/graphql')
  assert.equal(resolveGraphqlUrlFrom(undefined), '/graphql')
})

test('resolveApiPublicUrlFrom prefers runtime over vite and origin fallback', () => {
  assert.equal(
    resolveApiPublicUrlFrom({ apiPublicUrl: 'https://api.example.com' }),
    'https://api.example.com',
  )
  assert.equal(resolveApiPublicUrlFrom(undefined, 'https://build.example'), 'https://build.example')
  assert.equal(resolveApiPublicUrlFrom(undefined), 'https://app.example.com')
})

test('resolveStatusPollIntervalMsFrom parses runtime and falls back safely', () => {
  assert.equal(resolveStatusPollIntervalMsFrom({ statusPollIntervalMs: '45000' }), 45_000)
  assert.equal(resolveStatusPollIntervalMsFrom(undefined, '30000'), 30_000)
  assert.equal(resolveStatusPollIntervalMsFrom(undefined), 60_000)
  assert.equal(resolveStatusPollIntervalMsFrom({ statusPollIntervalMs: '0' }), 60_000)
})
