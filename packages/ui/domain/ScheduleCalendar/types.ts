export type ViewerRole = 'ADMIN' | 'MEMBER'

export type ScheduleCalendarUser = {
  id: string
  label: string
  email?: string
}

export type ScheduleCalendarRotation = {
  id: string
  name: string
  layer: number
  rrule: string
  participantIds: string[]
}

export type ScheduleCalendarOverride = {
  id: string
  rotationId: string
  userId: string
  replacedUserId?: string | null
  startsAt: string
  endsAt: string
}

export type ScheduleCalendarOnCallLayer = {
  layer: number
  rotationId: string
  userId: string
}

export type ScheduleCalendarData = {
  id: string
  name: string
  timezone: string
  teamId: string
  rotations: ScheduleCalendarRotation[]
}

export type CreateOverridePayload = {
  rotationId: string
  userId: string
  startsAt: string
  endsAt: string
}

export type ScheduleCalendarLabels = {
  title: string
  timezoneLabel: string
  onCallNow: string
  layerLabel: (layer: number) => string
  overrides: string
  noOverrides: string
  createOverride: string
  overrideForbidden: string
  rotation: string
  replacementUser: string
  dateRange: string
  cancel: string
  save: string
  delete: string
  loading: string
  computedAt: string
}
