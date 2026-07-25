import type {
  AlertFieldsFragment,
  IncidentFieldsFragment,
  IncidentRoleAssignmentFieldsFragment,
  TimelineEventFieldsFragment,
} from '@escalite/ts-types'

export const POSTMORTEM_MIME_TYPE = 'text/markdown;charset=utf-8'

export function formatUtcIso8601(value: string): string {
  return new Date(value).toISOString()
}

export function formatDurationMs(durationMs: number): string {
  if (durationMs < 0 || !Number.isFinite(durationMs)) {
    return '—'
  }

  const totalMinutes = Math.floor(durationMs / 60_000)
  const days = Math.floor(totalMinutes / (60 * 24))
  const hours = Math.floor((totalMinutes % (60 * 24)) / 60)
  const minutes = totalMinutes % 60

  const parts: string[] = []
  if (days > 0) {
    parts.push(`${days}d`)
  }
  if (hours > 0) {
    parts.push(`${hours}h`)
  }
  if (minutes > 0 || parts.length === 0) {
    parts.push(`${minutes}m`)
  }

  return parts.join(' ')
}

function incidentStatusText(status: string): string {
  switch (status) {
    case 'INVESTIGATING':
      return 'Investigating'
    case 'IDENTIFIED':
      return 'Identified'
    case 'MONITORING':
      return 'Monitoring'
    case 'RESOLVED':
      return 'Resolved'
    default:
      return status
  }
}

function timelineEventTypeText(eventType: string): string {
  switch (eventType) {
    case 'DECLARED':
      return 'Declared'
    case 'STATUS_CHANGED':
      return 'Status changed'
    case 'NOTE':
      return 'Note'
    case 'ROLE_ASSIGNED':
      return 'Role assigned'
    case 'ROLE_UNASSIGNED':
      return 'Role unassigned'
    default:
      return eventType
  }
}

function alertStatusText(status: string): string {
  switch (status) {
    case 'ACKNOWLEDGED':
      return 'Acknowledged'
    case 'CLOSED':
      return 'Closed'
    default:
      return 'Triggered'
  }
}

function alertPriorityText(priority: string): string {
  return priority === 'LOW' ? 'Low' : 'High'
}

function escapeTableCell(value: string): string {
  return value.replace(/\|/g, '\\|').replace(/\n/g, ' ')
}

function sortTimelineEvents(
  events: TimelineEventFieldsFragment[],
): TimelineEventFieldsFragment[] {
  return [...events].sort(
    (left, right) => new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime(),
  )
}

function sortAlerts(alerts: AlertFieldsFragment[]): AlertFieldsFragment[] {
  return [...alerts].sort(
    (left, right) => new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime(),
  )
}

function sortRoleAssignments(
  assignments: IncidentRoleAssignmentFieldsFragment[],
): IncidentRoleAssignmentFieldsFragment[] {
  return [...assignments].sort(
    (left, right) => new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime(),
  )
}

export type PostmortemDurationMetrics = {
  timeToResolveMs: number | null
  timeToFirstAcknowledgementMs: number | null
}

export function computePostmortemDurationMetrics(
  incident: Pick<IncidentFieldsFragment, 'createdAt' | 'resolvedAt'>,
  alerts: AlertFieldsFragment[],
): PostmortemDurationMetrics {
  const declaredAt = new Date(incident.createdAt).getTime()
  const resolvedAt = incident.resolvedAt ? new Date(incident.resolvedAt).getTime() : null

  const acknowledgementTimes = alerts
    .filter((alert) => alert.acknowledgedAt)
    .map((alert) => new Date(alert.acknowledgedAt as string).getTime())

  const firstAcknowledgementAt =
    acknowledgementTimes.length > 0 ? Math.min(...acknowledgementTimes) : null

  return {
    timeToResolveMs: resolvedAt === null ? null : resolvedAt - declaredAt,
    timeToFirstAcknowledgementMs:
      firstAcknowledgementAt === null ? null : firstAcknowledgementAt - declaredAt,
  }
}

export function buildPostmortemMarkdown(incident: IncidentFieldsFragment): string {
  const exportedAt = formatUtcIso8601(new Date().toISOString())
  const timelineEvents = sortTimelineEvents(incident.timelineEvents)
  const alerts = sortAlerts(incident.alerts)
  const roleAssignments = sortRoleAssignments(incident.roleAssignments)
  const metrics = computePostmortemDurationMetrics(incident, alerts)

  const lines: string[] = [
    `# Postmortem: ${incident.title}`,
    '',
    '## Summary',
    '',
    `- **Status:** ${incidentStatusText(incident.status)}`,
    `- **Declared:** ${formatUtcIso8601(incident.createdAt)}`,
    `- **Last updated:** ${formatUtcIso8601(incident.updatedAt)}`,
  ]

  if (incident.resolvedAt) {
    lines.push(`- **Resolved:** ${formatUtcIso8601(incident.resolvedAt)}`)
  }

  lines.push(
    `- **Created by:** ${incident.createdBy.email}`,
    `- **Exported at:** ${exportedAt}`,
    '',
    '## Duration metrics',
    '',
  )

  if (metrics.timeToResolveMs !== null) {
    lines.push(`- **Time to resolve:** ${formatDurationMs(metrics.timeToResolveMs)}`)
  } else {
    lines.push('- **Time to resolve:** — (incident not resolved)')
  }

  if (metrics.timeToFirstAcknowledgementMs !== null) {
    lines.push(
      `- **Time to first acknowledgement:** ${formatDurationMs(metrics.timeToFirstAcknowledgementMs)}`,
    )
  } else {
    lines.push('- **Time to first acknowledgement:** —')
  }

  lines.push(`- **Linked alerts:** ${alerts.length}`, '', '## Roles', '')

  if (roleAssignments.length === 0) {
    lines.push('_No roles assigned._', '')
  } else {
    lines.push('| Role | Assignee | Assigned at |', '| --- | --- | --- |')
    for (const assignment of roleAssignments) {
      lines.push(
        `| ${escapeTableCell(assignment.role.name)} | ${escapeTableCell(assignment.user.email)} | ${formatUtcIso8601(assignment.createdAt)} |`,
      )
    }
    lines.push('')
  }

  lines.push('## Linked alerts', '')

  if (alerts.length === 0) {
    lines.push('_No linked alerts._', '')
  } else {
    lines.push(
      '| Summary | Status | Priority | Created | Acknowledged |',
      '| --- | --- | --- | --- | --- |',
    )
    for (const alert of alerts) {
      lines.push(
        `| ${escapeTableCell(alert.summary)} | ${alertStatusText(alert.status)} | ${alertPriorityText(alert.priority)} | ${formatUtcIso8601(alert.createdAt)} | ${alert.acknowledgedAt ? formatUtcIso8601(alert.acknowledgedAt) : '—'} |`,
      )
    }
    lines.push('')
  }

  lines.push('## Timeline', '')

  if (timelineEvents.length === 0) {
    lines.push('_No timeline events._', '')
  } else {
    for (const event of timelineEvents) {
      const actorSuffix = event.actor ? ` (${event.actor.email})` : ''
      lines.push(
        `### ${formatUtcIso8601(event.createdAt)} — ${timelineEventTypeText(event.eventType)}${actorSuffix}`,
        '',
        event.body,
        '',
      )
    }
  }

  return `${lines.join('\n').trimEnd()}\n`
}

export function postmortemFilename(incident: Pick<IncidentFieldsFragment, 'id' | 'title'>): string {
  const slug = incident.title
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60)

  const suffix = slug.length > 0 ? slug : incident.id
  return `postmortem-${suffix}.md`
}

export function downloadPostmortemMarkdown(
  incident: IncidentFieldsFragment,
  markdown = buildPostmortemMarkdown(incident),
): void {
  const blob = new Blob([markdown], { type: POSTMORTEM_MIME_TYPE })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = postmortemFilename(incident)
  anchor.rel = 'noopener'
  anchor.click()
  URL.revokeObjectURL(url)
}
