import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  buildPostmortemMarkdown,
  computePostmortemDurationMetrics,
  formatDurationMs,
  formatUtcIso8601,
  postmortemFilename,
  POSTMORTEM_MIME_TYPE,
} from '../src/lib/postmortem-export.ts'

const sampleIncident = {
  id: 'inc-1',
  organizationId: 'org-1',
  teamId: 'team-1',
  title: 'Checkout API outage',
  status: 'RESOLVED',
  resolvedAt: '2026-07-25T14:30:00.000Z',
  createdAt: '2026-07-25T11:00:00.000Z',
  updatedAt: '2026-07-25T14:30:00.000Z',
  createdBy: { id: 'u1', email: 'admin@example.com' },
  alerts: [
    {
      id: 'alert-1',
      organizationId: 'org-1',
      serviceId: 'svc-1',
      status: 'ACKNOWLEDGED',
      dedupKey: 'checkout-down',
      summary: 'Checkout latency spike',
      description: 'p99 above threshold',
      priority: 'HIGH',
      eventCount: 1,
      acknowledgedAt: '2026-07-25T11:15:00.000Z',
      acknowledgedBy: { id: 'u2', email: 'ic@example.com' },
      closedAt: null,
      incidentId: 'inc-1',
      createdAt: '2026-07-25T10:55:00.000Z',
      updatedAt: '2026-07-25T11:15:00.000Z',
    },
  ],
  timelineEvents: [
    {
      id: 'evt-1',
      incidentId: 'inc-1',
      eventType: 'DECLARED',
      body: 'Incident declared from alert grouping.',
      metadata: {},
      createdAt: '2026-07-25T11:00:00.000Z',
      actor: { id: 'u1', email: 'admin@example.com' },
    },
    {
      id: 'evt-2',
      incidentId: 'inc-1',
      eventType: 'NOTE',
      body: 'Root cause identified in payment gateway.',
      metadata: {},
      createdAt: '2026-07-25T12:00:00.000Z',
      actor: { id: 'u2', email: 'ic@example.com' },
    },
  ],
  roleAssignments: [
    {
      id: 'assign-1',
      createdAt: '2026-07-25T11:05:00.000Z',
      role: { id: 'role-ic', name: 'Incident Commander', sortOrder: 0 },
      user: { id: 'u2', email: 'ic@example.com' },
      assignedBy: { id: 'u1', email: 'admin@example.com' },
    },
  ],
}

describe('postmortem export', () => {
  it('formats timestamps as UTC ISO8601', () => {
    assert.equal(formatUtcIso8601('2026-07-25T11:00:00.000Z'), '2026-07-25T11:00:00.000Z')
    assert.equal(formatUtcIso8601('2026-07-25T13:00:00+02:00'), '2026-07-25T11:00:00.000Z')
  })

  it('formats duration metrics', () => {
    assert.equal(formatDurationMs(3_600_000), '1h')
    assert.equal(formatDurationMs(12_600_000), '3h 30m')
    assert.equal(formatDurationMs(0), '0m')
  })

  it('computes resolve and acknowledgement durations', () => {
    const metrics = computePostmortemDurationMetrics(sampleIncident, sampleIncident.alerts)

    assert.equal(metrics.timeToResolveMs, 12_600_000)
    assert.equal(metrics.timeToFirstAcknowledgementMs, 900_000)
  })

  it('builds markdown with timeline, roles, alerts, and metrics', () => {
    const markdown = buildPostmortemMarkdown(sampleIncident)

    assert.match(markdown, /^# Postmortem: Checkout API outage/)
    assert.match(markdown, /\*\*Declared:\*\* 2026-07-25T11:00:00\.000Z/)
    assert.match(markdown, /\*\*Resolved:\*\* 2026-07-25T14:30:00\.000Z/)
    assert.match(markdown, /\*\*Time to resolve:\*\* 3h 30m/)
    assert.match(markdown, /\*\*Time to first acknowledgement:\*\* 15m/)
    assert.match(markdown, /\| Incident Commander \| ic@example\.com \| 2026-07-25T11:05:00\.000Z \|/)
    assert.match(markdown, /\| Checkout latency spike \| Acknowledged \| High \| 2026-07-25T10:55:00\.000Z \|/)
    assert.match(markdown, /### 2026-07-25T11:00:00\.000Z — Declared/)
    assert.match(markdown, /### 2026-07-25T12:00:00\.000Z — Note/)
    assert.match(markdown, /Root cause identified in payment gateway\./)
  })

  it('uses text/markdown mime type constant', () => {
    assert.equal(POSTMORTEM_MIME_TYPE, 'text/markdown;charset=utf-8')
  })

  it('derives a safe download filename', () => {
    assert.equal(postmortemFilename(sampleIncident), 'postmortem-checkout-api-outage.md')
    assert.equal(
      postmortemFilename({ id: 'inc-2', title: '!!!' }),
      'postmortem-inc-2.md',
    )
  })
})
