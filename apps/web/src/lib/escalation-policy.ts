import type {
  EscalationEditorPolicy,
  EscalationEditorStep,
  EscalationStepTarget,
} from '@escalite/ui/domain/EscalationPolicyEditor'

type ApiEscalationStep = {
  id: string
  stepOrder: number
  delayMinutes: number
  repeatLastStep: boolean
  maxRepeats?: number | null
}

export function mapApiPolicyToEditor(
  policy: {
    id: string
    name: string
    steps: ApiEscalationStep[]
  },
  targetsByStepId: Record<string, EscalationStepTarget[]> = {},
): EscalationEditorPolicy {
  return {
    id: policy.id,
    name: policy.name,
    steps: [...policy.steps]
      .sort((left, right) => left.stepOrder - right.stepOrder)
      .map(
        (step): EscalationEditorStep => ({
          id: step.id,
          stepOrder: step.stepOrder,
          delayMinutes: step.delayMinutes,
          repeatLastStep: step.repeatLastStep,
          maxRepeats: step.maxRepeats ?? null,
          targets: targetsByStepId[step.id] ?? [],
        }),
      ),
  }
}

export function createDefaultEditorPolicy(name = 'Default escalation'): EscalationEditorPolicy {
  const stepId = crypto.randomUUID()
  return {
    name,
    steps: [
      {
        id: stepId,
        stepOrder: 1,
        delayMinutes: 0,
        repeatLastStep: false,
        maxRepeats: null,
        targets: [
          {
            id: crypto.randomUUID(),
            targetType: 'user',
          },
        ],
      },
    ],
  }
}
