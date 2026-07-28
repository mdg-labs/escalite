import assert from 'node:assert/strict'
import test from 'node:test'

import { scheduleICalFilename } from '../src/lib/schedule-ical-filename.ts'

test('scheduleICalFilename slugifies schedule names', () => {
  assert.equal(scheduleICalFilename('Primary On-Call'), 'primary-on-call.ics')
  assert.equal(scheduleICalFilename('!!!'), 'schedule.ics')
})
