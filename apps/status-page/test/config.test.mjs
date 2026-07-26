import assert from 'node:assert/strict'
import test from 'node:test'

import {
  resolveApiPublicUrlFrom,
  resolveStatusPollIntervalMsFrom,
} from '@escalite/runtime-config'

test('status poll interval defaults to 60000', () => {
  assert.equal(resolveStatusPollIntervalMsFrom(undefined), 60_000)
})

test('runtime api public URL overrides build-time default', () => {
  assert.equal(
    resolveApiPublicUrlFrom({ apiPublicUrl: 'https://api.example.com' }),
    'https://api.example.com',
  )
})
