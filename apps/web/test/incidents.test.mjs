import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  assignmentByRoleDefinitionId,
  mergeTimelineEvent,
  sortTimelineEvents,
  userInitials,
} from '../src/lib/incidents.ts'

describe('incidents helpers', () => {
  it('sorts timeline events chronologically', () => {
    const sorted = sortTimelineEvents([
      {
        id: '2',
        incidentId: 'inc-1',
        eventType: 'NOTE',
        body: 'later',
        metadata: {},
        createdAt: '2026-07-25T12:00:00.000Z',
        actor: null,
      },
      {
        id: '1',
        incidentId: 'inc-1',
        eventType: 'DECLARED',
        body: 'earlier',
        metadata: {},
        createdAt: '2026-07-25T11:00:00.000Z',
        actor: null,
      },
    ])

    assert.equal(sorted[0]?.id, '1')
    assert.equal(sorted[1]?.id, '2')
  })

  it('merges timeline events without duplicates', () => {
    const initial = [
      {
        id: '1',
        incidentId: 'inc-1',
        eventType: 'DECLARED',
        body: 'declared',
        metadata: {},
        createdAt: '2026-07-25T11:00:00.000Z',
        actor: null,
      },
    ]

    const merged = mergeTimelineEvent(initial, {
      id: '2',
      incidentId: 'inc-1',
      eventType: 'NOTE',
      body: 'note',
      metadata: {},
      createdAt: '2026-07-25T12:00:00.000Z',
      actor: null,
    })

    assert.equal(merged.length, 2)
    assert.equal(mergeTimelineEvent(merged, merged[1]).length, 2)
  })

  it('derives user initials from email', () => {
    assert.equal(userInitials('ic.lead@example.com'), 'IL')
    assert.equal(userInitials('solo@example.com'), 'SO')
  })

  it('finds assignment by role definition id', () => {
    const assignment = assignmentByRoleDefinitionId(
      [
        {
          id: 'a1',
          createdAt: '2026-07-25T11:00:00.000Z',
          role: { id: 'role-ic', name: 'IC', sortOrder: 0 },
          user: { id: 'u1', email: 'ic@example.com' },
          assignedBy: { id: 'u2', email: 'admin@example.com' },
        },
      ],
      'role-ic',
    )

    assert.equal(assignment?.role.name, 'IC')
  })
})
