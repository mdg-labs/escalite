import type { ScheduleCalendarLabels, ScheduleCalendarUser } from '@escalite/ui/domain/ScheduleCalendar'

import { t } from './i18n'

export function scheduleCalendarLabels(): ScheduleCalendarLabels {
  return {
    title: t('schedule.calendar.title'),
    timezoneLabel: t('schedule.timezone.label'),
    onCallNow: t('schedule.onCallNow'),
    layerLabel: (layer) => t('schedule.layer', { layer: String(layer) }),
    overrides: t('schedule.overrides.title'),
    noOverrides: t('schedule.overrides.empty'),
    createOverride: t('schedule.override.create'),
    overrideForbidden: t('schedule.override.forbidden'),
    rotation: t('schedule.override.rotation'),
    replacementUser: t('schedule.override.user'),
    dateRange: t('schedule.override.dateRange'),
    cancel: t('schedule.action.cancel'),
    save: t('schedule.action.save'),
    delete: t('schedule.action.delete'),
    loading: t('schedule.loading'),
    computedAt: t('schedule.computedAt'),
    rotationsTitle: t('schedule.rotations.title'),
    rotationsEmpty: t('schedule.rotations.empty'),
    createRotation: t('schedule.rotations.create'),
    editRotation: t('schedule.rotations.edit'),
    deleteRotation: t('schedule.rotations.delete'),
    deleteRotationConfirm: t('schedule.rotations.deleteConfirm'),
    rotationName: t('schedule.rotations.name'),
    rotationLayer: t('schedule.rotations.layer'),
    rotationLayerPrimary: t('schedule.rotations.layerPrimary'),
    rotationLayerSecondary: t('schedule.rotations.layerSecondary'),
    rotationParticipants: t('schedule.rotations.participants'),
    rotationParticipantsEmpty: t('schedule.rotations.participantsEmpty'),
    rotationFrequency: t('schedule.rotations.frequency'),
    rotationFrequencyHourly: t('schedule.rotations.frequencyHourly'),
    rotationFrequencyDaily: t('schedule.rotations.frequencyDaily'),
    rotationFrequencyWeekly: t('schedule.rotations.frequencyWeekly'),
    rotationInterval: t('schedule.rotations.interval'),
    rotationCustomRrule: t('schedule.rotations.customRrule'),
    rotationRrulePreview: t('schedule.rotations.rrulePreview'),
    rotationForbidden: t('schedule.rotations.forbidden'),
    rotationErrorName: t('schedule.rotations.error.name'),
    rotationErrorParticipants: t('schedule.rotations.error.participants'),
    rotationErrorRrule: t('schedule.rotations.error.rrule'),
    rotationErrorLayer: t('schedule.rotations.error.layer'),
    describeRrule: (frequency, interval) =>
      interval === 1
        ? t('schedule.rotations.describe.single', { frequency })
        : t('schedule.rotations.describe.interval', { frequency, interval: String(interval) }),
  }
}

export function mapOrganizationUsersToScheduleUsers(
  organizationUsers: Array<{ id: string; email: string }>,
): ScheduleCalendarUser[] {
  return [...organizationUsers]
    .sort((left, right) => left.email.localeCompare(right.email))
    .map((user) => ({
      id: user.id,
      label: user.email,
      email: user.email,
    }))
}

export function collectScheduleUsers(
  organizationUsers: Array<{ id: string; email: string }>,
  participantIds: string[],
  onCallUserIds: string[],
  overrideUserIds: string[],
): ScheduleCalendarUser[] {
  const byId = new Map(
    organizationUsers.map((user) => [
      user.id,
      { id: user.id, label: user.email, email: user.email },
    ]),
  )
  const ids = new Set([...participantIds, ...onCallUserIds, ...overrideUserIds])

  return [...ids].map((id) => byId.get(id) ?? { id, label: id.slice(0, 8) })
}

export function formatGraphQLError(message: string): string {
  return message.replace(/^(\[GraphQL\]\s*)+/, '')
}
