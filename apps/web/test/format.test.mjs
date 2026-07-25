import assert from 'node:assert/strict'
import test from 'node:test'

import { formatDateTime, formatGraphQLError } from '../src/lib/format.ts'

test('formatGraphQLError strips repeated GraphQL prefixes', () => {
  assert.equal(formatGraphQLError('[GraphQL] [GraphQL] Not found'), 'Not found')
})

test('formatDateTime renders a locale string for valid ISO timestamps', () => {
  const formatted = formatDateTime('2026-07-25T10:00:00.000Z')
  assert.match(formatted, /2026/)
})
