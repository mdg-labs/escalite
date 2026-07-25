import type { ReactElement } from 'react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import {
  IncidentStatus,
  type IncidentRoleAssignmentFieldsFragment,
  type TimelineEventFieldsFragment,
  useAddIncidentTimelineNoteMutation,
  useAssignIncidentRoleMutation,
  useIncidentQuery,
  useIncidentRoleDefinitionsQuery,
  useIncidentTimelineUpdatedSubscription,
  useIncidentsQuery,
  useMeQuery,
  useUpdateIncidentStatusMutation,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  AlertTitle,
  Avatar,
  AvatarFallback,
  Badge,
  Button,
  ScrollArea,
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Textarea,
} from '@escalite/ui'
import { AlertCircleIcon, DownloadIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import { formatDateTime } from '../lib/format'
import {
  assignmentByRoleDefinitionId,
  incidentStatusLabel,
  mergeTimelineEvent,
  sortTimelineEvents,
  timelineEventLabel,
  userInitials,
} from '../lib/incidents'
import { t } from '../lib/i18n'
import { downloadPostmortemMarkdown } from '../lib/postmortem-export'

const INCIDENT_STATUSES: IncidentStatus[] = [
  IncidentStatus.Investigating,
  IncidentStatus.Identified,
  IncidentStatus.Monitoring,
  IncidentStatus.Resolved,
]

function IncidentStatusBadge({ status }: { status: IncidentStatus }): ReactElement {
  const variant =
    status === IncidentStatus.Resolved
      ? 'secondary'
      : status === IncidentStatus.Monitoring
        ? 'info'
        : status === IncidentStatus.Identified
          ? 'warning'
          : 'destructive'

  return (
    <Badge variant={variant}>
      {t(incidentStatusLabel(status) as Parameters<typeof t>[0])}
    </Badge>
  )
}

function RoleSlot({
  roleName,
  roleDefinitionId,
  assignment,
  assigningRoleId,
  onAssign,
}: {
  roleName: string
  roleDefinitionId: string
  assignment?: IncidentRoleAssignmentFieldsFragment
  assigningRoleId: string | null
  onAssign: () => void
}): ReactElement {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-border p-3">
      <div className="flex min-w-0 items-center gap-3">
        <Badge variant="outline">{roleName}</Badge>
        {assignment ? (
          <>
            <Avatar className="size-8">
              <AvatarFallback>{userInitials(assignment.user.email)}</AvatarFallback>
            </Avatar>
            <span className="truncate text-sm text-foreground">{assignment.user.email}</span>
          </>
        ) : (
          <span className="text-sm text-muted-foreground">{t('incidents.roles.unassigned')}</span>
        )}
      </div>
      {!assignment ? (
        <Button
          loading={assigningRoleId === roleDefinitionId}
          onClick={onAssign}
          size="sm"
          variant="outline"
        >
          {assigningRoleId === roleDefinitionId
            ? t('incidents.roles.assigning')
            : t('incidents.roles.assignMe')}
        </Button>
      ) : null}
    </div>
  )
}

function TimelineFeed({ events }: { events: TimelineEventFieldsFragment[] }): ReactElement {
  if (events.length === 0) {
    return <p className="text-sm text-muted-foreground">{t('incidents.timeline.empty')}</p>
  }

  return (
    <div className="space-y-3">
      {events.map((event) => (
        <article key={event.id} className="rounded-lg border border-border p-3">
          <div className="mb-2 flex flex-wrap items-center gap-2">
            <Badge size="sm" variant="secondary">
              {t(timelineEventLabel(event.eventType) as Parameters<typeof t>[0])}
            </Badge>
            <span className="text-xs text-muted-foreground">{formatDateTime(event.createdAt)}</span>
            {event.actor ? (
              <span className="text-xs text-muted-foreground">{event.actor.email}</span>
            ) : null}
          </div>
          <p className="whitespace-pre-wrap text-sm text-foreground">{event.body}</p>
        </article>
      ))}
    </div>
  )
}

export function IncidentsPage(): ReactElement {
  const navigate = useNavigate()
  const { incidentId } = useParams()
  const [timelineEvents, setTimelineEvents] = useState<TimelineEventFieldsFragment[]>([])
  const [roleAssignments, setRoleAssignments] = useState<IncidentRoleAssignmentFieldsFragment[]>([])
  const [incidentStatus, setIncidentStatus] = useState<IncidentStatus>(IncidentStatus.Investigating)
  const [noteBody, setNoteBody] = useState('')
  const [actionError, setActionError] = useState<string | null>(null)
  const [noteLoading, setNoteLoading] = useState(false)
  const [statusLoading, setStatusLoading] = useState(false)
  const [assigningRoleId, setAssigningRoleId] = useState<string | null>(null)

  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const currentUserId = meData?.me?.id ?? ''

  const [{ data: incidentsData, fetching: incidentsFetching }] = useIncidentsQuery({
    variables: { limit: 100 },
    requestPolicy: 'network-only',
  })

  const [{ data: incidentData, fetching: incidentFetching }] = useIncidentQuery({
    variables: { id: incidentId ?? '' },
    pause: !incidentId,
    requestPolicy: 'network-only',
  })

  const [{ data: roleDefinitionsData }] = useIncidentRoleDefinitionsQuery({
    requestPolicy: 'cache-first',
  })

  const [, addTimelineNote] = useAddIncidentTimelineNoteMutation()
  const [, updateIncidentStatus] = useUpdateIncidentStatusMutation()
  const [, assignIncidentRole] = useAssignIncidentRoleMutation()

  const incident = incidentData?.incident

  useEffect(() => {
    if (!incident) {
      return
    }
    setTimelineEvents(sortTimelineEvents(incident.timelineEvents))
    setRoleAssignments(incident.roleAssignments)
    setIncidentStatus(incident.status)
  }, [incident])

  const applyTimelineEvent = useCallback((event: TimelineEventFieldsFragment) => {
    setTimelineEvents((current) => mergeTimelineEvent(current, event))
  }, [])

  useIncidentTimelineUpdatedSubscription(
    {
      variables: { incidentId: incidentId ?? '' },
      pause: !incidentId,
    },
    (_previous, response) => {
      const updated = response.incidentTimelineUpdated
      if (updated) {
        applyTimelineEvent(updated)
      }
      return response
    },
  )

  const roleDefinitions = useMemo(
    () => roleDefinitionsData?.incidentRoleDefinitions ?? [],
    [roleDefinitionsData?.incidentRoleDefinitions],
  )

  const incidents = incidentsData?.incidents ?? []

  async function handleAddNote(): Promise<void> {
    if (!incidentId || !noteBody.trim()) {
      return
    }
    setActionError(null)
    setNoteLoading(true)
    const result = await addTimelineNote({
      input: {
        incidentId,
        body: noteBody.trim(),
      },
    })
    setNoteLoading(false)
    if (result.error) {
      setActionError(t('incidents.error.action'))
      return
    }
    if (result.data?.addIncidentTimelineNote) {
      applyTimelineEvent(result.data.addIncidentTimelineNote)
      setNoteBody('')
    }
  }

  async function handleStatusUpdate(): Promise<void> {
    if (!incidentId || !incident || incident.status === incidentStatus) {
      return
    }
    setActionError(null)
    setStatusLoading(true)
    const result = await updateIncidentStatus({
      input: {
        id: incidentId,
        status: incidentStatus,
      },
    })
    setStatusLoading(false)
    if (result.error) {
      setActionError(t('incidents.error.action'))
      return
    }
    if (result.data?.updateIncidentStatus) {
      setTimelineEvents(sortTimelineEvents(result.data.updateIncidentStatus.timelineEvents))
      setIncidentStatus(result.data.updateIncidentStatus.status)
    }
  }

  async function handleAssignRole(roleDefinitionId: string): Promise<void> {
    if (!incidentId || !currentUserId) {
      return
    }
    setActionError(null)
    setAssigningRoleId(roleDefinitionId)
    const result = await assignIncidentRole({
      input: {
        incidentId,
        roleDefinitionId,
        userId: currentUserId,
      },
    })
    setAssigningRoleId(null)
    if (result.error) {
      setActionError(t('incidents.error.action'))
      return
    }
    if (result.data?.assignIncidentRole) {
      const assigned = result.data.assignIncidentRole
      setRoleAssignments((current) => {
        const withoutRole = current.filter(
          (assignment) => assignment.role.id !== assigned.role.id,
        )
        return [...withoutRole, assigned]
      })
    }
  }

  return (
    <AppShell title={t('incidents.title')}>
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1.4fr)]">
        <section className="rounded-xl border border-border bg-card shadow-xs/5">
          <div className="border-b border-border p-4">
            <h2 className="text-sm font-medium text-foreground">{t('incidents.title')}</h2>
          </div>
          <ScrollArea className="h-[40rem]">
            <Table variant="card">
              <TableHeader>
                <TableRow>
                  <TableHead>{t('incidents.column.title')}</TableHead>
                  <TableHead>{t('incidents.column.status')}</TableHead>
                  <TableHead>{t('incidents.column.updated')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {incidentsFetching && incidents.length === 0 ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={3}>
                      {t('incidents.loading')}
                    </TableCell>
                  </TableRow>
                ) : null}
                {!incidentsFetching && incidents.length === 0 ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={3}>
                      {t('incidents.empty')}
                    </TableCell>
                  </TableRow>
                ) : null}
                {incidents.map((item) => (
                  <TableRow
                    key={item.id}
                    className="cursor-pointer"
                    data-state={item.id === incidentId ? 'selected' : undefined}
                    onClick={() => {
                      navigate(`/incidents/${item.id}`)
                    }}
                  >
                    <TableCell className="max-w-xs truncate font-medium text-foreground">
                      {item.title}
                    </TableCell>
                    <TableCell>
                      <IncidentStatusBadge status={item.status} />
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDateTime(item.updatedAt)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
        </section>

        <section className="space-y-6 rounded-xl border border-border bg-card p-6 shadow-xs/5">
          {!incidentId ? (
            <p className="text-sm text-muted-foreground">{t('incidents.detail.select')}</p>
          ) : incidentFetching && !incident ? (
            <p className="text-sm text-muted-foreground">{t('incidents.detail.loading')}</p>
          ) : !incident ? (
            <p className="text-sm text-muted-foreground">{t('incidents.detail.notFound')}</p>
          ) : (
            <>
              <div className="space-y-3">
                <div className="flex flex-wrap items-center gap-2">
                  <IncidentStatusBadge status={incident.status} />
                  <span className="text-xs text-muted-foreground">
                    {formatDateTime(incident.createdAt)}
                  </span>
                </div>
                <h2 className="text-lg font-semibold text-foreground">{incident.title}</h2>
              </div>

              {actionError ? (
                <Alert variant="error">
                  <AlertCircleIcon />
                  <AlertTitle>{t('incidents.error.action')}</AlertTitle>
                  <AlertDescription>{actionError}</AlertDescription>
                </Alert>
              ) : null}

              <div className="flex flex-wrap items-end gap-3">
                <div className="min-w-48 flex-1">
                  <label className="mb-1 block text-sm font-medium text-foreground">
                    {t('incidents.column.status')}
                  </label>
                  <Select
                    onValueChange={(value) => {
                      setIncidentStatus(value as IncidentStatus)
                    }}
                    value={incidentStatus}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectPopup>
                      {INCIDENT_STATUSES.map((status) => (
                        <SelectItem key={status} value={status}>
                          {t(incidentStatusLabel(status) as Parameters<typeof t>[0])}
                        </SelectItem>
                      ))}
                    </SelectPopup>
                  </Select>
                </div>
                <Button
                  disabled={incident.status === incidentStatus}
                  loading={statusLoading}
                  onClick={() => {
                    void handleStatusUpdate()
                  }}
                >
                  {statusLoading ? t('incidents.status.updating') : t('incidents.status.update')}
                </Button>
              </div>

              <div className="space-y-3">
                <h3 className="text-sm font-medium text-foreground">{t('incidents.roles.title')}</h3>
                <div className="space-y-2">
                  {roleDefinitions.length === 0
                    ? roleAssignments.map((assignment) => (
                        <RoleSlot
                          key={assignment.id}
                          assignment={assignment}
                          assigningRoleId={assigningRoleId}
                          onAssign={() => {
                            void handleAssignRole(assignment.role.id)
                          }}
                          roleDefinitionId={assignment.role.id}
                          roleName={assignment.role.name}
                        />
                      ))
                    : roleDefinitions.map((roleDefinition) => (
                        <RoleSlot
                          key={roleDefinition.id}
                          assignment={assignmentByRoleDefinitionId(
                            roleAssignments,
                            roleDefinition.id,
                          )}
                          assigningRoleId={assigningRoleId}
                          onAssign={() => {
                            void handleAssignRole(roleDefinition.id)
                          }}
                          roleDefinitionId={roleDefinition.id}
                          roleName={roleDefinition.name}
                        />
                      ))}
                </div>
              </div>

              <div className="space-y-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <h3 className="text-sm font-medium text-foreground">{t('incidents.timeline.title')}</h3>
                  <Button
                    onClick={() => {
                      downloadPostmortemMarkdown({
                        ...incident,
                        status: incidentStatus,
                        timelineEvents,
                        roleAssignments,
                      })
                    }}
                    size="sm"
                    variant="outline"
                  >
                    <DownloadIcon className="size-4" />
                    {t('incidents.export.markdown')}
                  </Button>
                </div>
                <ScrollArea className="h-80">
                  <TimelineFeed events={timelineEvents} />
                </ScrollArea>
              </div>

              <div className="space-y-3">
                <Textarea
                  aria-label={t('incidents.note.placeholder')}
                  onChange={(event) => {
                    setNoteBody(event.target.value)
                  }}
                  placeholder={t('incidents.note.placeholder')}
                  value={noteBody}
                />
                <div className="flex justify-end">
                  <Button
                    disabled={!noteBody.trim()}
                    loading={noteLoading}
                    onClick={() => {
                      void handleAddNote()
                    }}
                  >
                    {noteLoading ? t('incidents.note.adding') : t('incidents.note.add')}
                  </Button>
                </div>
              </div>
            </>
          )}
        </section>
      </div>
    </AppShell>
  )
}
