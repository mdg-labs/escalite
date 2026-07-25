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
  }
}

export function collectScheduleUsers(
  participantIds: string[],
  onCallUserIds: string[],
  overrideUserIds: string[],
): ScheduleCalendarUser[] {
  const ids = new Set([...participantIds, ...onCallUserIds, ...overrideUserIds])
  return [...ids].map((id) => ({
    id,
    label: id.slice(0, 8),
  }))
}

export function formatGraphQLError(message: string): string {
  return message.replace(/^(\[GraphQL\]\s*)+/, '')
}
