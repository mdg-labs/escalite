import assert from 'node:assert/strict'
import test from 'node:test'

import {
  alertPriorityBadgeVariant,
  alertStatusBadgeVariant,
  alertStatusIndicatorClass,
  formatAlertCardTimestamp,
} from '../domain/AlertCard/utils.ts'

test('alertStatusIndicatorClass maps status to severity token classes', () => {
  assert.equal(alertStatusIndicatorClass('Triggered'), 'bg-severity-high')
  assert.equal(alertStatusIndicatorClass('Acknowledged'), 'bg-severity-info')
  assert.equal(alertStatusIndicatorClass('Closed'), 'bg-severity-resolved')
})

test('alertStatusBadgeVariant maps status to severity badge variants', () => {
  assert.equal(alertStatusBadgeVariant('Triggered'), 'severity-high')
  assert.equal(alertStatusBadgeVariant('Acknowledged'), 'severity-info')
  assert.equal(alertStatusBadgeVariant('Closed'), 'severity-resolved')
})

test('alertPriorityBadgeVariant maps priority to severity badge variants', () => {
  assert.equal(alertPriorityBadgeVariant('High'), 'severity-critical')
  assert.equal(alertPriorityBadgeVariant('Low'), 'severity-low')
})

test('formatAlertCardTimestamp returns em dash for empty values', () => {
  assert.equal(formatAlertCardTimestamp(null), '—')
  assert.equal(formatAlertCardTimestamp(undefined), '—')
  assert.equal(formatAlertCardTimestamp(''), '—')
})

test('formatAlertCardTimestamp formats ISO timestamps', () => {
  const formatted = formatAlertCardTimestamp('2026-07-27T10:30:00.000Z')
  assert.match(formatted, /2026/)
})
