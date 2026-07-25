export type EscalationTargetKind = 're_escalate' | 'snooze'

export type EscalationTarget = {
  id: string
  label: string
  kind: EscalationTargetKind
  durationMinutes?: number
}

export const ESCALATION_TARGETS: EscalationTarget[] = [
  { id: 're_escalate', label: 'Re-escalate to step 1', kind: 're_escalate' },
  { id: 'snooze_15', label: 'Snooze 15 minutes', kind: 'snooze', durationMinutes: 15 },
  { id: 'snooze_30', label: 'Snooze 30 minutes', kind: 'snooze', durationMinutes: 30 },
  { id: 'snooze_60', label: 'Snooze 1 hour', kind: 'snooze', durationMinutes: 60 },
]
