import assert from 'node:assert/strict'
import process from 'node:process'
import test from 'node:test'

test('defaults GraphQL URL to same-origin proxy path', () => {
  const graphqlUrl = process.env.VITE_GRAPHQL_URL ?? '/graphql'
  assert.equal(graphqlUrl, '/graphql')
})
