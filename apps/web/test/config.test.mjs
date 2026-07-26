import assert from 'node:assert/strict'
import test from 'node:test'

import {
  resolveApiPublicUrlFrom,
  resolveGraphqlUrlFrom,
} from '@escalite/runtime-config'

test('defaults GraphQL URL to same-origin proxy path', () => {
  assert.equal(resolveGraphqlUrlFrom(undefined), '/graphql')
})

test('runtime GraphQL URL overrides build-time default', () => {
  assert.equal(
    resolveGraphqlUrlFrom({ graphqlUrl: 'https://api.example.com/graphql' }),
    'https://api.example.com/graphql',
  )
})

test('api public URL falls back to origin when unset', () => {
  assert.equal(resolveApiPublicUrlFrom(undefined, undefined, 'https://app.example.com'), 'https://app.example.com')
})
