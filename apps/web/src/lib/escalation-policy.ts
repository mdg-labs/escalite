import type {
  EscalationEditorOptions,
  EscalationEditorPolicy,
  EscalationEditorStep,
  EscalationStepTarget,
  EscalationTargetType,
} from '@escalite/ui/domain/EscalationPolicyEditor'

type OrganizationUserOption = {
  id: string
  email: string
}

type ScheduleOption = {
  id: string
  name: string
}

type ApiEscalationStepTarget = {
  id: string
  targetType: string
  userId?: string | null
  scheduleId?: string | null
  webhookUrl?: string | null
}

type ApiEscalationStep = {
  id: string
  stepOrder: number
  delayMinutes: number
  repeatLastStep: boolean
  maxRepeats?: number | null
  targets?: ApiEscalationStepTarget[]
}

function mapApiTargetToEditor(target: ApiEscalationStepTarget): EscalationStepTarget {
  return {
    id: target.id,
    targetType: target.targetType as EscalationTargetType,
    userId: target.userId ?? undefined,
    scheduleId: target.scheduleId ?? undefined,
    webhookUrl: target.webhookUrl ?? undefined,
  }
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
          targets: (step.targets ?? targetsByStepId[step.id] ?? []).map(mapApiTargetToEditor),
        }),
      ),
  }
}

export function mapOrganizationUsersToEditorOptions(
  users: OrganizationUserOption[],
): EscalationEditorOptions['users'] {
  return [...users]
    .sort((left, right) => left.email.localeCompare(right.email))
    .map((user) => ({
      id: user.id,
      label: user.email,
    }))
}

export function mapTeamSchedulesToEditorOptions(
  schedules: ScheduleOption[],
): EscalationEditorOptions['schedules'] {
  return [...schedules]
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((schedule) => ({
      id: schedule.id,
      label: schedule.name,
    }))
}

export function buildEscalationEditorOptions(
  users: OrganizationUserOption[],
  schedules: ScheduleOption[],
): EscalationEditorOptions {
  return {
    users: mapOrganizationUsersToEditorOptions(users),
    schedules: mapTeamSchedulesToEditorOptions(schedules),
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
