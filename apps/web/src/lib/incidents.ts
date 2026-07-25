import type { IncidentRoleAssignmentFieldsFragment, TimelineEventFieldsFragment } from '@escalite/ts-types'

export function sortTimelineEvents(
  events: TimelineEventFieldsFragment[],
): TimelineEventFieldsFragment[] {
  return [...events].sort(
    (left, right) => new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime(),
  )
}

export function mergeTimelineEvent(
  events: TimelineEventFieldsFragment[],
  incoming: TimelineEventFieldsFragment,
): TimelineEventFieldsFragment[] {
  const existingIndex = events.findIndex((event) => event.id === incoming.id)
  if (existingIndex === -1) {
    return sortTimelineEvents([...events, incoming])
  }

  const next = [...events]
  next[existingIndex] = incoming
  return next
}

export function userInitials(email: string): string {
  const localPart = email.split('@')[0] ?? email
  const segments = localPart.split(/[._-]+/).filter(Boolean)
  if (segments.length >= 2) {
    return `${segments[0]?.[0] ?? ''}${segments[1]?.[0] ?? ''}`.toUpperCase()
  }
  return localPart.slice(0, 2).toUpperCase()
}

export function assignmentByRoleDefinitionId(
  assignments: IncidentRoleAssignmentFieldsFragment[],
  roleDefinitionId: string,
): IncidentRoleAssignmentFieldsFragment | undefined {
  return assignments.find((assignment) => assignment.role.id === roleDefinitionId)
}

export function incidentStatusLabel(status: string): string {
  switch (status) {
    case 'INVESTIGATING':
      return 'incidents.status.investigating'
    case 'IDENTIFIED':
      return 'incidents.status.identified'
    case 'MONITORING':
      return 'incidents.status.monitoring'
    case 'RESOLVED':
      return 'incidents.status.resolved'
    default:
      return status
  }
}

export function timelineEventLabel(eventType: string): string {
  switch (eventType) {
    case 'DECLARED':
      return 'incidents.timeline.declared'
    case 'STATUS_CHANGED':
      return 'incidents.timeline.statusChanged'
    case 'NOTE':
      return 'incidents.timeline.note'
    case 'ROLE_ASSIGNED':
      return 'incidents.timeline.roleAssigned'
    case 'ROLE_UNASSIGNED':
      return 'incidents.timeline.roleUnassigned'
    default:
      return eventType
  }
}
