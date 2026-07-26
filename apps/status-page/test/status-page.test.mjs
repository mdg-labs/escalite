import assert from 'node:assert/strict'
import test from 'node:test'

import {
  componentStatusBadgeVariant,
  componentStatusLabel,
  overallStatusLabel,
  publicStatusPageUrl,
  sortComponents,
  worstComponentStatus,
} from '../src/lib/status-page.ts'

test('sortComponents orders by position ascending', () => {
  const sorted = sortComponents([
    { id: '2', name: 'B', status: 'operational', position: 2 },
    { id: '1', name: 'A', status: 'operational', position: 0 },
    { id: '3', name: 'C', status: 'operational', position: 1 },
  ])

  assert.deepEqual(
    sorted.map((component) => component.name),
    ['A', 'C', 'B'],
  )
})

test('worstComponentStatus returns the highest severity', () => {
  const worst = worstComponentStatus([
    { id: '1', name: 'API', status: 'operational', position: 0 },
    { id: '2', name: 'DB', status: 'degraded', position: 1 },
    { id: '3', name: 'Auth', status: 'partial_outage', position: 2 },
  ])

  assert.equal(worst, 'partial_outage')
})

test('component status helpers map public API values', () => {
  assert.equal(componentStatusLabel('major_outage'), 'statusPage.component.status.major_outage')
  assert.equal(componentStatusBadgeVariant('major_outage'), 'destructive')
  assert.equal(overallStatusLabel('degraded'), 'statusPage.overall.degraded')
})

test('publicStatusPageUrl normalizes slug casing and whitespace', () => {
  assert.equal(
    publicStatusPageUrl(' Acme-Status ', 'https://status.example.com'),
    'https://status.example.com/api/v1/public/status/acme-status',
  )
})
