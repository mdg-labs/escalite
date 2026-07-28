import type { ScheduleCalendarUser } from '@escalite/ui/domain/ScheduleCalendar'

export type ScheduleOrganizationUser = {
  id: string
  email: string
  name?: string | null
}

export function scheduleUserDisplayLabel(
  user: Pick<ScheduleOrganizationUser, 'email' | 'name'>,
): string {
  const name = user.name?.trim()
  if (name) {
    return name
  }

  return user.email
}

function toScheduleCalendarUser(user: ScheduleOrganizationUser): ScheduleCalendarUser {
  return {
    id: user.id,
    label: scheduleUserDisplayLabel(user),
    email: user.email,
  }
}

export function mapOrganizationUsersToScheduleUsers(
  organizationUsers: ScheduleOrganizationUser[],
): ScheduleCalendarUser[] {
  return [...organizationUsers]
    .sort((left, right) =>
      scheduleUserDisplayLabel(left).localeCompare(scheduleUserDisplayLabel(right)),
    )
    .map(toScheduleCalendarUser)
}

export function collectScheduleUsers(
  organizationUsers: ScheduleOrganizationUser[],
  participantIds: string[],
  onCallUserIds: string[],
  overrideUserIds: string[],
): ScheduleCalendarUser[] {
  const byId = new Map(organizationUsers.map((user) => [user.id, toScheduleCalendarUser(user)]))
  const ids = new Set([...participantIds, ...onCallUserIds, ...overrideUserIds])

  return [...ids].map((id) => byId.get(id) ?? { id, label: id })
}
