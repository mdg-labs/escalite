import type {
  EscalationEditorStep,
  EscalationPolicySavePayload,
  EscalationStepTarget,
  EscalationValidationIssue,
} from './types'

export function isTargetComplete(target: EscalationStepTarget): boolean {
  switch (target.targetType) {
    case 'user':
      return Boolean(target.userId?.trim())
    case 'rotation':
      return Boolean(target.scheduleId?.trim())
    case 'webhook':
      return Boolean(target.webhookUrl?.trim())
    default:
      return false
  }
}

export function stepHasTargets(step: EscalationEditorStep): boolean {
  return step.targets.length > 0 && step.targets.every(isTargetComplete)
}

export function validateEscalationPolicySteps(
  steps: EscalationEditorStep[],
): EscalationValidationIssue[] {
  const issues: EscalationValidationIssue[] = []

  if (steps.length === 0) {
    issues.push({ message: 'At least one escalation step is required.' })
    return issues
  }

  steps.forEach((step, index) => {
    if (step.targets.length === 0) {
      issues.push({
        stepId: step.id,
        message: `Step ${index + 1} must include at least one target.`,
      })
      return
    }

    if (!stepHasTargets(step)) {
      issues.push({
        stepId: step.id,
        message: `Step ${index + 1} has incomplete targets.`,
      })
    }

    if (step.delayMinutes < 0) {
      issues.push({
        stepId: step.id,
        message: `Step ${index + 1} delay must be non-negative.`,
      })
    }
  })

  return issues
}

export function canSaveEscalationPolicy(steps: EscalationEditorStep[]): boolean {
  return validateEscalationPolicySteps(steps).length === 0
}

export function toEscalationPolicySavePayload(
  name: string,
  steps: EscalationEditorStep[],
): EscalationPolicySavePayload {
  return {
    name: name.trim(),
    steps: steps.map((step, index) => ({
      stepOrder: index + 1,
      delayMinutes: index === 0 ? 0 : step.delayMinutes,
      repeatLastStep: step.repeatLastStep,
      maxRepeats: step.maxRepeats ?? undefined,
    })),
  }
}
