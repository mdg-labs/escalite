export type AlertAnalyticsRollup = {
  windowDays: number
  mttaSeconds?: number | null
  acknowledgedCount: number
  mttrSeconds?: number | null
  resolvedCount: number
}

export type ResponseTimeChartRow = {
  window: string
  MTTA: number
  MTTR: number
}

export type VolumeChartRow = {
  window: string
  Acknowledged: number
  Resolved: number
}

export function formatDurationSeconds(seconds: number | null | undefined): string {
  if (seconds == null || Number.isNaN(seconds)) {
    return '—'
  }
  if (seconds < 60) {
    return `${Math.round(seconds)}s`
  }
  if (seconds < 3600) {
    return `${Math.round(seconds / 60)}m`
  }
  const hours = seconds / 3600
  if (hours < 24) {
    return `${hours.toFixed(1)}h`
  }
  return `${(hours / 24).toFixed(1)}d`
}

export function rollupByWindowDays(
  rollups: AlertAnalyticsRollup[],
  windowDays: number,
): AlertAnalyticsRollup | undefined {
  return rollups.find((rollup) => rollup.windowDays === windowDays)
}

export function responseTimeChartData(rollups: AlertAnalyticsRollup[]): ResponseTimeChartRow[] {
  return [...rollups]
    .sort((left, right) => left.windowDays - right.windowDays)
    .map((rollup) => ({
      window: `${rollup.windowDays}d`,
      MTTA: rollup.mttaSeconds != null ? rollup.mttaSeconds / 60 : 0,
      MTTR: rollup.mttrSeconds != null ? rollup.mttrSeconds / 60 : 0,
    }))
}

export function volumeChartData(rollups: AlertAnalyticsRollup[]): VolumeChartRow[] {
  return [...rollups]
    .sort((left, right) => left.windowDays - right.windowDays)
    .map((rollup) => ({
      window: `${rollup.windowDays}d`,
      Acknowledged: rollup.acknowledgedCount,
      Resolved: rollup.resolvedCount,
    }))
}

export function minutesValueFormatter(value: number): string {
  if (value <= 0) {
    return '0m'
  }
  if (value < 60) {
    return `${Math.round(value)}m`
  }
  return `${(value / 60).toFixed(1)}h`
}
