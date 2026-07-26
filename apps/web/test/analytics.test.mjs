import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  formatDurationSeconds,
  responseTimeChartData,
  rollupByWindowDays,
  volumeChartData,
} from '../src/lib/analytics.ts'

describe('analytics helpers', () => {
  const rollups = [
    {
      windowDays: 30,
      mttaSeconds: 7200,
      acknowledgedCount: 120,
      mttrSeconds: 14400,
      resolvedCount: 95,
    },
    {
      windowDays: 7,
      mttaSeconds: 300,
      acknowledgedCount: 40,
      mttrSeconds: 900,
      resolvedCount: 32,
    },
  ]

  it('formats durations for display', () => {
    assert.equal(formatDurationSeconds(null), '—')
    assert.equal(formatDurationSeconds(45), '45s')
    assert.equal(formatDurationSeconds(150), '3m')
    assert.equal(formatDurationSeconds(5400), '1.5h')
  })

  it('finds rollups by window length', () => {
    assert.equal(rollupByWindowDays(rollups, 7)?.acknowledgedCount, 40)
    assert.equal(rollupByWindowDays(rollups, 14), undefined)
  })

  it('builds response-time chart rows in ascending window order', () => {
    const rows = responseTimeChartData(rollups)
    assert.deepEqual(rows, [
      { window: '7d', MTTA: 5, MTTR: 15 },
      { window: '30d', MTTA: 120, MTTR: 240 },
    ])
  })

  it('builds volume chart rows in ascending window order', () => {
    const rows = volumeChartData(rollups)
    assert.deepEqual(rows, [
      { window: '7d', Acknowledged: 40, Resolved: 32 },
      { window: '30d', Acknowledged: 120, Resolved: 95 },
    ])
  })
})
