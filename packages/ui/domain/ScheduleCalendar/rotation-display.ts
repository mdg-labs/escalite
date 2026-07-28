import { parseRRule } from './rrule'
import type { ScheduleCalendarLabels } from './types'

export function layerLabel(labels: ScheduleCalendarLabels, layer: number): string {
  if (layer === 1) {
    return labels.rotationLayerPrimary
  }

  if (layer === 2) {
    return labels.rotationLayerSecondary
  }

  return labels.layerLabel(layer)
}

export function describeRotationRrule(
  labels: ScheduleCalendarLabels,
  rrule: string,
): string {
  const parsed = parseRRule(rrule)
  if (!parsed) {
    return rrule
  }

  const frequencyLabel =
    parsed.frequency === 'HOURLY'
      ? labels.rotationFrequencyHourly
      : parsed.frequency === 'DAILY'
        ? labels.rotationFrequencyDaily
        : labels.rotationFrequencyWeekly

  return labels.describeRrule(frequencyLabel, parsed.interval)
}
