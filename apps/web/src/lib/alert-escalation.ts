import { AlertStatus } from '@escalite/ts-types'

export type SnoozePreset = {
  id: string
  labelKey:
    | 'alerts.snooze.preset.15'
    | 'alerts.snooze.preset.30'
    | 'alerts.snooze.preset.60'
  durationMinutes: number
}

export const SNOOZE_PRESETS: SnoozePreset[] = [
  { id: 'snooze_15', labelKey: 'alerts.snooze.preset.15', durationMinutes: 15 },
  { id: 'snooze_30', labelKey: 'alerts.snooze.preset.30', durationMinutes: 30 },
  { id: 'snooze_60', labelKey: 'alerts.snooze.preset.60', durationMinutes: 60 },
]

export function canSnoozeAlert(status: AlertStatus): boolean {
  return status === AlertStatus.Triggered || status === AlertStatus.Acknowledged
}

export function canReEscalateAlert(status: AlertStatus): boolean {
  return status === AlertStatus.Acknowledged
}
