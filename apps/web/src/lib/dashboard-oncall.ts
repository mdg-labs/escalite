import type { OnCallNowQuery, SchedulesQuery } from '@escalite/ts-types'
import type { OnCallWidgetSchedule } from '@escalite/ui/domain/OnCallWidget'

export type OnCallLoadEntry = {
  schedule: SchedulesQuery['schedules'][number]
  onCall: NonNullable<OnCallNowQuery['onCallNow']>
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
