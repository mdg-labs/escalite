import assert from 'node:assert/strict'
import process from 'node:process'
import test from 'node:test'

test('defaults poll interval env to 60 seconds when unset', () => {
  const pollIntervalMs = Number.parseInt(process.env.VITE_STATUS_POLL_INTERVAL_MS ?? '60000', 10)
  assert.equal(pollIntervalMs, 60_000)
})
