import type { EscalationEditorStep } from './types'

export function reorderEscalationSteps(
  steps: EscalationEditorStep[],
  activeId: string,
  overId: string,
): EscalationEditorStep[] {
  if (activeId === overId) {
    return steps
  }

  const oldIndex = steps.findIndex((step) => step.id === activeId)
  const newIndex = steps.findIndex((step) => step.id === overId)

  if (oldIndex < 0 || newIndex < 0) {
    return steps
  }

  const next = [...steps]
  const [moved] = next.splice(oldIndex, 1)
  if (!moved) {
    return steps
  }
  next.splice(newIndex, 0, moved)

  return normalizeStepOrders(next)
}

export function normalizeStepOrders(steps: EscalationEditorStep[]): EscalationEditorStep[] {
  return steps.map((step, index) => ({
    ...step,
    stepOrder: index + 1,
    delayMinutes: index === 0 ? 0 : step.delayMinutes,
  }))
}

export function removeEscalationStep(
  steps: EscalationEditorStep[],
  stepId: string,
): EscalationEditorStep[] {
  if (steps.length <= 1) {
    return steps
  }

  return normalizeStepOrders(steps.filter((step) => step.id !== stepId))
}

export function addEscalationStep(steps: EscalationEditorStep[]): EscalationEditorStep[] {
  const nextOrder = steps.length + 1
  return normalizeStepOrders([
    ...steps,
    {
      id: crypto.randomUUID(),
      stepOrder: nextOrder,
      delayMinutes: 5,
      repeatLastStep: false,
      maxRepeats: null,
      targets: [
        {
          id: crypto.randomUUID(),
          targetType: 'user',
        },
      ],
    },
  ])
}
