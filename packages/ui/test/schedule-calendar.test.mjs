import assert from 'node:assert/strict'
import test from 'node:test'

import { canCreateScheduleOverride } from '../domain/ScheduleCalendar/permissions.ts'
import {
  formatViewerLocalDateRange,
  formatViewerLocalTime,
} from '../domain/ScheduleCalendar/timezone.ts'

test('canCreateScheduleOverride allows admin and team members only', () => {
  assert.equal(canCreateScheduleOverride('ADMIN', false), true)
  assert.equal(canCreateScheduleOverride('MEMBER', true), true)
  assert.equal(canCreateScheduleOverride('MEMBER', false), false)
})

test('formatViewerLocalTime includes timezone label', () => {
  const result = formatViewerLocalTime('2026-07-25T10:30:00.000Z', 'en-US', 'UTC')
  assert.match(result.formatted, /2026/)
  assert.equal(result.timezoneLabel, 'UTC')
})

test('formatViewerLocalDateRange renders start and end in viewer timezone', () => {
  const range = formatViewerLocalDateRange(
    '2026-07-25T00:00:00.000Z',
    '2026-07-26T23:59:59.000Z',
    'en-US',
    'UTC',
  )
  assert.match(range, /–/)
  assert.match(range, /2026/)
})
