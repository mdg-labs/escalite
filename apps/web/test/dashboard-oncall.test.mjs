import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import { mapOnCallSchedules, updateOnCallSchedule } from '../src/lib/dashboard-oncall.ts'

describe('dashboard on-call helpers', () => {
  it('maps schedule and on-call layers into widget shape', () => {
    const mapped = mapOnCallSchedules(
      [
        {
          schedule: {
            id: 'sched-1',
            name: 'Primary',
            rotations: [{ id: 'rot-1', name: 'Week A' }],
          },
          onCall: {
            scheduleId: 'sched-1',
            computedAt: '2026-07-27T10:00:00.000Z',
            layers: [{ layer: 1, rotationId: 'rot-1', userId: 'user-1' }],
          },
        },
      ],
      (value) => `at ${value}`,
    )

    assert.equal(mapped.length, 1)
    assert.equal(mapped[0]?.id, 'sched-1')
    assert.equal(mapped[0]?.name, 'Primary')
    assert.equal(mapped[0]?.computedAt, 'at 2026-07-27T10:00:00.000Z')
    assert.equal(mapped[0]?.layers[0]?.rotationName, 'Week A')
    assert.equal(mapped[0]?.layers[0]?.userId, 'user-1')
  })

  it('updates an existing schedule entry in place', () => {
    const initial = [
      {
        id: 'sched-1',
        name: 'Primary',
        computedAt: 'old',
        layers: [{ layer: 1, rotationId: 'rot-1', userId: 'user-1' }],
      },
      {
        id: 'sched-2',
        name: 'Secondary',
        computedAt: 'old',
        layers: [{ layer: 1, rotationId: 'rot-2', userId: 'user-2' }],
      },
    ]

    const updated = updateOnCallSchedule(initial, {
      id: 'sched-1',
      name: 'Primary',
      computedAt: 'new',
      layers: [{ layer: 1, rotationId: 'rot-1', userId: 'user-3' }],
    })

    assert.equal(updated.length, 2)
    assert.equal(updated[0]?.computedAt, 'new')
    assert.equal(updated[0]?.layers[0]?.userId, 'user-3')
    assert.equal(updated[1]?.id, 'sched-2')
  })

  it('appends a schedule when it was not previously loaded', () => {
    const updated = updateOnCallSchedule([], {
      id: 'sched-1',
      name: 'Primary',
      computedAt: 'new',
      layers: [{ layer: 1, rotationId: 'rot-1', userId: 'user-1' }],
    })

    assert.equal(updated.length, 1)
    assert.equal(updated[0]?.id, 'sched-1')
  })
})
