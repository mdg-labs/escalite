import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  buildUntilByScheduleLayer,
  mapOnCallSchedules,
  onCallScheduleLayerKey,
  updateOnCallSchedule,
  viewerIsOnCall,
} from '../src/lib/dashboard-oncall.ts'

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

  it('builds until labels keyed by schedule and layer', () => {
    const untilByLayer = buildUntilByScheduleLayer(
      [
        {
          scheduleId: 'sched-1',
          scheduleName: 'Primary',
          teamName: 'Platform',
          layer: 1,
          until: '2026-07-29T18:00:00.000Z',
        },
      ],
      (value) => `until ${value}`,
    )

    assert.equal(untilByLayer[onCallScheduleLayerKey('sched-1', 1)], 'until 2026-07-29T18:00:00.000Z')
  })

  it('detects when the viewer is on call from assignments or schedule layers', () => {
    const schedules = [
      {
        id: 'sched-1',
        name: 'Primary',
        layers: [{ layer: 1, rotationId: 'rot-1', userId: 'user-1' }],
      },
    ]

    assert.equal(viewerIsOnCall('user-1', schedules, []), true)
    assert.equal(viewerIsOnCall('user-2', schedules, []), false)
    assert.equal(
      viewerIsOnCall(
        'user-2',
        [],
        [
          {
            scheduleId: 'sched-2',
            scheduleName: 'Secondary',
            teamName: 'Platform',
            layer: 2,
            until: '2026-07-29T18:00:00.000Z',
          },
        ],
      ),
      true,
    )
  })
})
