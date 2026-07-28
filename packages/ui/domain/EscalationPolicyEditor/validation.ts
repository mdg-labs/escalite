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

  const lastStep = steps[steps.length - 1]
  if (lastStep.repeatLastStep) {
    if (lastStep.maxRepeats == null || !Number.isInteger(lastStep.maxRepeats)) {
      issues.push({
        stepId: lastStep.id,
        message: 'Max repeat count is required when repeat is enabled.',
      })
    } else if (lastStep.maxRepeats < 1) {
      issues.push({
        stepId: lastStep.id,
        message: 'Max repeat count must be at least 1 when repeat is enabled.',
      })
    }
  } else if (lastStep.maxRepeats != null && lastStep.maxRepeats < 1) {
    issues.push({
      stepId: lastStep.id,
      message: 'Max repeat count must be at least 1 when set.',
    })
  }

  return issues
}

export function canSaveEscalationPolicy(steps: EscalationEditorStep[]): boolean {
  return validateEscalationPolicySteps(steps).length === 0
}

function toTargetInputPayload(
  target: EscalationStepTarget,
): EscalationPolicySavePayload['steps'][number]['targets'][number] {
  const payload: EscalationPolicySavePayload['steps'][number]['targets'][number] = {
    targetType: target.targetType,
  }

  const userId = target.userId?.trim()
  if (userId) {
    payload.userId = userId
  }

  const scheduleId = target.scheduleId?.trim()
  if (scheduleId) {
    payload.scheduleId = scheduleId
  }

  const webhookUrl = target.webhookUrl?.trim()
  if (webhookUrl) {
    payload.webhookUrl = webhookUrl
  }

  return payload
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
      maxRepeats: step.repeatLastStep ? step.maxRepeats ?? undefined : undefined,
      targets: step.targets.map(toTargetInputPayload),
    })),
  }
}
