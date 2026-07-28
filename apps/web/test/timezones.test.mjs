import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  defaultIanaTimezone,
  filterIanaTimezones,
  isValidIanaTimezone,
  listIanaTimezones,
} from '../src/lib/timezones.ts'

describe('timezones helpers', () => {
  it('accepts valid IANA timezones', () => {
    assert.equal(isValidIanaTimezone('UTC'), true)
    assert.equal(isValidIanaTimezone('Europe/Vienna'), true)
    assert.equal(isValidIanaTimezone('America/New_York'), true)
  })

  it('rejects invalid timezone identifiers', () => {
    assert.equal(isValidIanaTimezone(''), false)
    assert.equal(isValidIanaTimezone('Not/A_Timezone'), false)
    assert.equal(isValidIanaTimezone('GMT+5'), false)
  })

  it('lists and filters IANA timezones', () => {
    const timezones = listIanaTimezones()
    assert.ok(timezones.length > 0)
    assert.ok(timezones.includes('UTC'))

    const filtered = filterIanaTimezones(timezones, 'Europe/')
    assert.ok(filtered.every((timezone) => timezone.startsWith('Europe/')))
    assert.ok(filtered.includes('Europe/Vienna'))
  })

  it('returns a valid default timezone', () => {
    assert.equal(isValidIanaTimezone(defaultIanaTimezone()), true)
  })
})
