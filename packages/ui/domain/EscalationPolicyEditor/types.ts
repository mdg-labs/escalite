export type EscalationTargetType = 'user' | 'rotation' | 'webhook'

export type EscalationStepTarget = {
  id: string
  targetType: EscalationTargetType
  userId?: string
  scheduleId?: string
  webhookUrl?: string
}

export type EscalationEditorStep = {
  id: string
  stepOrder: number
  delayMinutes: number
  repeatLastStep: boolean
  maxRepeats?: number | null
  targets: EscalationStepTarget[]
}

export type EscalationEditorPolicy = {
  id?: string
  name: string
  steps: EscalationEditorStep[]
}

export type EscalationTargetOption = {
  id: string
  label: string
}

export type EscalationEditorOptions = {
  users: EscalationTargetOption[]
  schedules: EscalationTargetOption[]
}

export type EscalationStepTargetInputPayload = {
  targetType: EscalationTargetType
  userId?: string
  scheduleId?: string
  webhookUrl?: string
}

export type EscalationStepInputPayload = {
  stepOrder: number
  delayMinutes: number
  repeatLastStep?: boolean
  maxRepeats?: number | null
  targets: EscalationStepTargetInputPayload[]
}

export type EscalationPolicySavePayload = {
  name: string
  steps: EscalationStepInputPayload[]
}

export type EscalationValidationIssue = {
  stepId?: string
  message: string
}

export function createEmptyTarget(): EscalationStepTarget {
  return {
    id: crypto.randomUUID(),
    targetType: 'user',
  }
}

export function createEmptyStep(order: number): EscalationEditorStep {
  return {
    id: crypto.randomUUID(),
    stepOrder: order,
    delayMinutes: order === 1 ? 0 : 5,
    repeatLastStep: false,
    maxRepeats: null,
    targets: [createEmptyTarget()],
  }
}
