import assert from 'node:assert/strict'
import test from 'node:test'

import {
  collectScheduleUsers,
  mapOrganizationUsersToScheduleUsers,
  scheduleUserDisplayLabel,
} from '../src/lib/schedule-users.ts'

test('scheduleUserDisplayLabel prefers name over email', () => {
  assert.equal(
    scheduleUserDisplayLabel({ name: 'Grace Hopper', email: 'grace@example.com' }),
    'Grace Hopper',
  )
})

test('scheduleUserDisplayLabel falls back to email when name is blank', () => {
  assert.equal(scheduleUserDisplayLabel({ name: '   ', email: 'oncall@example.com' }), 'oncall@example.com')
  assert.equal(scheduleUserDisplayLabel({ email: 'oncall@example.com' }), 'oncall@example.com')
})

test('mapOrganizationUsersToScheduleUsers sorts by display label', () => {
  const users = mapOrganizationUsersToScheduleUsers([
    { id: 'user-2', email: 'zoe@example.com' },
    { id: 'user-1', email: 'amy@example.com', name: 'Amy Adams' },
  ])

  assert.deepEqual(users, [
    { id: 'user-1', label: 'Amy Adams', email: 'amy@example.com' },
    { id: 'user-2', label: 'zoe@example.com', email: 'zoe@example.com' },
  ])
})

test('collectScheduleUsers resolves labels from organization directory', () => {
  const users = collectScheduleUsers(
    [
      { id: 'user-1', email: 'amy@example.com', name: 'Amy Adams' },
      { id: 'user-2', email: 'zoe@example.com' },
    ],
    ['user-1', 'user-2'],
    ['user-1'],
    ['user-2'],
  )

  assert.deepEqual(users, [
    { id: 'user-1', label: 'Amy Adams', email: 'amy@example.com' },
    { id: 'user-2', label: 'zoe@example.com', email: 'zoe@example.com' },
  ])
})

test('collectScheduleUsers does not truncate unknown user ids', () => {
  const unknownId = '01234567-89ab-cdef-0123-456789abcdef'
  const users = collectScheduleUsers([], [], [unknownId], [])

  assert.deepEqual(users, [{ id: unknownId, label: unknownId }])
})
