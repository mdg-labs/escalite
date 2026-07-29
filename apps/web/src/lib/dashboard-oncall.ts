import type { MyOnCallStatusQuery, OnCallNowQuery, SchedulesQuery } from '@escalite/ts-types'
import type { OnCallWidgetSchedule } from '@escalite/ui/domain/OnCallWidget'

export type OnCallLoadEntry = {
  schedule: SchedulesQuery['schedules'][number]
  onCall: NonNullable<OnCallNowQuery['onCallNow']>
}

export type MyOnCallAssignmentLike = MyOnCallStatusQuery['myOnCallStatus'][number]

export function onCallScheduleLayerKey(scheduleId: string, layer: number): string {
  return `${scheduleId}:${layer}`
}

export function mapOnCallSchedules(
  schedulesByTeam: OnCallLoadEntry[],
  formatComputedAt: (value: string) => string,
): OnCallWidgetSchedule[] {
  return schedulesByTeam.map(({ schedule, onCall }) => {
    const rotationNameById = new Map(
      schedule.rotations.map((rotation) => [rotation.id, rotation.name]),
    )

    return {
      id: schedule.id,
      name: schedule.name,
      computedAt: formatComputedAt(onCall.computedAt),
      layers: onCall.layers.map((layer) => ({
        layer: layer.layer,
        rotationId: layer.rotationId,
        rotationName: rotationNameById.get(layer.rotationId),
        userId: layer.userId,
      })),
    }
  })
}

export function updateOnCallSchedule(
  schedules: OnCallWidgetSchedule[],
  updated: OnCallWidgetSchedule,
): OnCallWidgetSchedule[] {
  const index = schedules.findIndex((schedule) => schedule.id === updated.id)
  if (index === -1) {
    return [...schedules, updated]
  }

  const next = [...schedules]
  next[index] = updated
  return next
}

export function buildUntilByScheduleLayer(
  assignments: MyOnCallAssignmentLike[],
  formatUntil: (value: string) => string,
): Record<string, string> {
  const result: Record<string, string> = {}

  for (const assignment of assignments) {
    result[onCallScheduleLayerKey(assignment.scheduleId, assignment.layer)] = formatUntil(assignment.until)
  }

  return result
}

export function viewerIsOnCall(
  viewerUserId: string | undefined,
  schedules: OnCallWidgetSchedule[],
  assignments: MyOnCallAssignmentLike[],
): boolean {
  if (assignments.length > 0) {
    return true
  }

  if (!viewerUserId) {
    return false
  }

  return schedules.some((schedule) =>
    schedule.layers.some((layer) => layer.userId === viewerUserId),
  )
}
