import type { ReactElement } from 'react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import {
  AlertStatus,
  IncidentStatus,
  type IncidentRoleAssignmentFieldsFragment,
  type TimelineEventFieldsFragment,
  useAddIncidentTimelineNoteMutation,
  useAlertsQuery,
  useAssignIncidentRoleMutation,
  useUnassignIncidentRoleMutation,
  useCreateIncidentMutation,
  useIncidentQuery,
  useIncidentRoleDefinitionsQuery,
  useIncidentTimelineUpdatedSubscription,
  useIncidentsQuery,
  useMeQuery,
  usePromoteAlertToIncidentMutation,
  usePublishIncidentToStatusPageMutation,
  useServicesQuery,
  useStatusPageQuery,
  useTeamsQuery,
  useUpdateIncidentStatusMutation,
  UserRole,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  AlertTitle,
  Avatar,
  AvatarFallback,
  Badge,
  Button,
  Dialog,
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogPanel,
  DialogPopup,
  DialogTitle,
  DialogTrigger,
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  Input,
  ScrollArea,
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Textarea,
} from '@escalite/ui'
import { AlertCircleIcon, DownloadIcon, ExternalLinkIcon, PlusIcon, SirenIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import { formatDateTime, formatGraphQLError } from '../lib/format'
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
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

const INCIDENT_STATUSES: IncidentStatus[] = [
  IncidentStatus.Investigating,
  IncidentStatus.Identified,
  IncidentStatus.Monitoring,
  IncidentStatus.Resolved,
]

function statusPagePublicAppUrl(slug: string): string {
  const configured = import.meta.env.VITE_STATUS_PAGE_PUBLIC_URL?.trim()
  const base =
    configured ||
    (import.meta.env.DEV ? 'http://localhost:5174' : window.location.origin)
  return `${base.replace(/\/$/, '')}/${encodeURIComponent(slug.trim().toLowerCase())}`
}

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
  unassigningRoleId,
  onAssign,
  onUnassign,
}: {
  roleName: string
  roleDefinitionId: string
  assignment?: IncidentRoleAssignmentFieldsFragment
  assigningRoleId: string | null
  unassigningRoleId: string | null
  onAssign: () => void
  onUnassign: () => void
}): ReactElement {
  const isAssigning = assigningRoleId === roleDefinitionId
  const isUnassigning = assignment != null && unassigningRoleId === assignment.id

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
      {assignment ? (
        <Button
          loading={isUnassigning}
          onClick={onUnassign}
          size="sm"
          variant="outline"
        >
          {isUnassigning ? t('incidents.roles.unassigning') : t('incidents.roles.unassign')}
        </Button>
      ) : (
        <Button loading={isAssigning} onClick={onAssign} size="sm" variant="outline">
          {isAssigning ? t('incidents.roles.assigning') : t('incidents.roles.assignMe')}
        </Button>
      )}
    </div>
  )
}

function CreateIncidentDialog({
  onCreated,
}: {
  onCreated: (incidentId: string) => void
}): ReactElement {
  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [teamId, setTeamId] = useState('')
  const [selectedAlertIds, setSelectedAlertIds] = useState<string[]>([])
  const [formError, setFormError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const [{ data: teamsData }] = useTeamsQuery({
    requestPolicy: 'cache-first',
    pause: !open,
  })
  const [{ data: alertsData }] = useAlertsQuery({
    variables: { limit: 100 },
    requestPolicy: 'cache-first',
    pause: !open,
  })
  const [{ data: servicesData }] = useServicesQuery({
    requestPolicy: 'cache-first',
    pause: !open,
  })

  const [, createIncident] = useCreateIncidentMutation()
  const [, promoteAlertToIncident] = usePromoteAlertToIncidentMutation()

  const teams = teamsData?.teams ?? []
  const serviceTeamById = useMemo(() => {
    const map = new Map<string, string>()
    for (const service of servicesData?.services ?? []) {
      map.set(service.id, service.teamId)
    }
    return map
  }, [servicesData?.services])

  const linkableAlerts = useMemo(() => {
    if (!teamId) {
      return []
    }
    return (alertsData?.alerts ?? []).filter((alert) => {
      if (alert.incidentId) {
        return false
      }
      if (alert.status === AlertStatus.Closed) {
        return false
      }
      return serviceTeamById.get(alert.serviceId) === teamId
    })
  }, [alertsData?.alerts, serviceTeamById, teamId])

  useEffect(() => {
    setSelectedAlertIds((current) =>
      current.filter((alertId) => linkableAlerts.some((alert) => alert.id === alertId)),
    )
  }, [linkableAlerts])

  function resetForm(): void {
    setTitle('')
    setTeamId('')
    setSelectedAlertIds([])
    setFormError(null)
  }

  function toggleAlertSelection(alertId: string): void {
    setSelectedAlertIds((current) =>
      current.includes(alertId)
        ? current.filter((id) => id !== alertId)
        : [...current, alertId],
    )
  }

  async function handleCreate(): Promise<void> {
    const trimmedTitle = title.trim()
    if (!trimmedTitle) {
      setFormError(t('incidents.error.requiredTitle'))
      return
    }
    if (!teamId) {
      setFormError(t('incidents.error.requiredTeam'))
      return
    }

    setFormError(null)
    setCreating(true)

    const result = await createIncident({
      input: {
        title: trimmedTitle,
        teamId,
      },
    })

    if (result.error || !result.data?.createIncident) {
      setCreating(false)
      if (result.error) {
        showMutationError(result.error, 'incidents.error.action')
      }
      setFormError(
        result.error ? formatGraphQLError(result.error.message) : t('incidents.error.action'),
      )
      return
    }

    const createdIncidentId = result.data.createIncident.id

    for (const alertId of selectedAlertIds) {
      const linkResult = await promoteAlertToIncident({
        input: {
          alertId,
          incidentId: createdIncidentId,
        },
      })
      if (linkResult.error) {
        setCreating(false)
        showMutationError(linkResult.error, 'incidents.error.action')
        setFormError(formatGraphQLError(linkResult.error.message))
        onCreated(createdIncidentId)
        return
      }
    }

    setCreating(false)
    setOpen(false)
    resetForm()
    notifyMutationSuccess('incidents.toast.created')
    onCreated(createdIncidentId)
  }

  return (
    <Dialog
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen)
        if (!nextOpen) {
          resetForm()
        }
      }}
      open={open}
    >
      <DialogTrigger render={<Button type="button" />}>
        <PlusIcon />
        {t('incidents.action.create')}
      </DialogTrigger>
      <DialogPopup>
        <DialogHeader>
          <DialogTitle>{t('incidents.create.title')}</DialogTitle>
          <DialogDescription>{t('incidents.create.description')}</DialogDescription>
        </DialogHeader>
        <DialogPanel className="space-y-4">
          {formError ? (
            <Alert variant="error">
              <AlertCircleIcon />
              <AlertTitle>{t('incidents.error.action')}</AlertTitle>
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          ) : null}
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="incident-title">
              {t('incidents.field.title')}
            </label>
            <Input
              id="incident-title"
              onChange={(event) => setTitle(event.target.value)}
              placeholder={t('incidents.field.titlePlaceholder')}
              value={title}
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="incident-team">
              {t('incidents.field.team')}
            </label>
            <Select
              onValueChange={(value) => {
                setTeamId(value ?? '')
              }}
              value={teamId || null}
            >
              <SelectTrigger id="incident-team">
                <SelectValue placeholder={t('incidents.field.teamPlaceholder')} />
              </SelectTrigger>
              <SelectPopup>
                {teams.map((team) => (
                  <SelectItem key={team.id} value={team.id}>
                    {team.name}
                  </SelectItem>
                ))}
              </SelectPopup>
            </Select>
          </div>
          <div className="space-y-2">
            <div>
              <p className="text-sm font-medium text-foreground">{t('incidents.field.linkedAlerts')}</p>
              <p className="text-sm text-muted-foreground">
                {t('incidents.field.linkedAlertsDescription')}
              </p>
            </div>
            <ScrollArea className="h-40 rounded-lg border border-border">
              <div className="space-y-1 p-2">
                {!teamId ? (
                  <p className="px-2 py-3 text-sm text-muted-foreground">
                    {t('incidents.field.teamPlaceholder')}
                  </p>
                ) : linkableAlerts.length === 0 ? (
                  <p className="px-2 py-3 text-sm text-muted-foreground">
                    {t('incidents.field.linkedAlertsEmpty')}
                  </p>
                ) : (
                  linkableAlerts.map((alert) => (
                    <label
                      key={alert.id}
                      className="flex cursor-pointer items-start gap-3 rounded-md px-2 py-2 hover:bg-muted/50"
                    >
                      <input
                        checked={selectedAlertIds.includes(alert.id)}
                        className="mt-1"
                        onChange={() => toggleAlertSelection(alert.id)}
                        type="checkbox"
                      />
                      <span className="min-w-0">
                        <span className="block truncate text-sm font-medium text-foreground">
                          {alert.summary}
                        </span>
                        <span className="block truncate text-xs text-muted-foreground">
                          {alert.service.name}
                        </span>
                      </span>
                    </label>
                  ))
                )}
              </div>
            </ScrollArea>
          </div>
        </DialogPanel>
        <DialogFooter>
          <DialogClose render={<Button type="button" variant="ghost" />}>
            {t('incidents.action.cancel')}
          </DialogClose>
          <Button disabled={creating} onClick={() => void handleCreate()} type="button">
            {creating ? t('incidents.action.creating') : t('incidents.create.title')}
          </Button>
        </DialogFooter>
      </DialogPopup>
    </Dialog>
  )
}

function PublishToStatusPageDialog({
  incidentId,
  incidentTitle,
  isAdmin,
}: {
  incidentId: string
  incidentTitle: string
  isAdmin: boolean
}): ReactElement | null {
  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [selectedComponentIds, setSelectedComponentIds] = useState<string[]>([])
  const [formError, setFormError] = useState<string | null>(null)
  const [publishing, setPublishing] = useState(false)
  const [published, setPublished] = useState(false)

  const [{ data: statusPageData }] = useStatusPageQuery({
    requestPolicy: 'cache-first',
    pause: !isAdmin,
  })
  const [, publishIncidentToStatusPage] = usePublishIncidentToStatusPageMutation()

  const statusPage = statusPageData?.statusPage
  const slug = statusPage?.slug
  const enabled = statusPage?.enabled ?? false
  const components = statusPage?.components ?? []

  const existingPublication = useMemo(() => {
    return (
      statusPage?.incidents.find((statusPageIncident) => statusPageIncident.incidentId === incidentId) ??
      null
    )
  }, [incidentId, statusPage?.incidents])

  const isPublished = existingPublication != null || published
  const publicUrl = slug ? statusPagePublicAppUrl(slug) : null

  useEffect(() => {
    if (!open) {
      return
    }
    setTitle(incidentTitle)
    setBody('')
    setSelectedComponentIds([])
    setFormError(null)
  }, [incidentTitle, open])

  function resetForm(): void {
    setTitle('')
    setBody('')
    setSelectedComponentIds([])
    setFormError(null)
  }

  function toggleComponentSelection(componentId: string): void {
    setSelectedComponentIds((current) =>
      current.includes(componentId)
        ? current.filter((id) => id !== componentId)
        : [...current, componentId],
    )
  }

  async function handlePublish(): Promise<void> {
    const trimmedBody = body.trim()
    if (!trimmedBody) {
      setFormError(t('incidents.statusPage.error.requiredBody'))
      return
    }
    if (selectedComponentIds.length === 0) {
      setFormError(t('incidents.statusPage.error.requiredComponents'))
      return
    }

    setFormError(null)
    setPublishing(true)

    const result = await publishIncidentToStatusPage({
      input: {
        incidentId,
        title: title.trim() || undefined,
        affectedComponentIds: selectedComponentIds,
        body: trimmedBody,
      },
    })

    setPublishing(false)

    if (result.error || !result.data?.publishIncidentToStatusPage) {
      if (result.error) {
        showMutationError(result.error, 'incidents.error.action')
      }
      setFormError(
        result.error ? formatGraphQLError(result.error.message) : t('incidents.error.action'),
      )
      return
    }

    setPublished(true)
    setOpen(false)
    resetForm()
    notifyMutationSuccess('incidents.toast.published')
  }

  if (!isAdmin || !enabled || !slug) {
    return null
  }

  return (
    <div className="space-y-2">
      {isPublished && publicUrl ? (
        <p className="text-sm text-muted-foreground">
          {t('incidents.statusPage.published')}{' '}
          <a
            className="inline-flex items-center gap-1 font-medium text-primary underline-offset-4 hover:underline"
            href={publicUrl}
            rel="noreferrer"
            target="_blank"
          >
            {t('incidents.statusPage.viewPublic')}
            <ExternalLinkIcon className="size-3.5" />
          </a>
        </p>
      ) : null}
      {!isPublished ? (
        <Dialog
          onOpenChange={(nextOpen) => {
            setOpen(nextOpen)
            if (!nextOpen) {
              resetForm()
            }
          }}
          open={open}
        >
          <DialogTrigger render={<Button size="sm" type="button" variant="outline" />}>
            {t('incidents.statusPage.publish')}
          </DialogTrigger>
          <DialogPopup>
            <DialogHeader>
              <DialogTitle>{t('incidents.statusPage.title')}</DialogTitle>
              <DialogDescription>{t('incidents.statusPage.description')}</DialogDescription>
            </DialogHeader>
            <DialogPanel className="space-y-4">
              {formError ? (
                <Alert variant="error">
                  <AlertCircleIcon />
                  <AlertTitle>{t('incidents.error.action')}</AlertTitle>
                  <AlertDescription>{formError}</AlertDescription>
                </Alert>
              ) : null}
              <div className="space-y-2">
                <div>
                  <label
                    className="text-sm font-medium text-foreground"
                    htmlFor="status-page-title"
                  >
                    {t('incidents.statusPage.field.publicTitle')}
                  </label>
                  <p className="text-sm text-muted-foreground">
                    {t('incidents.statusPage.field.publicTitleDescription')}
                  </p>
                </div>
                <Input
                  id="status-page-title"
                  onChange={(event) => setTitle(event.target.value)}
                  placeholder={incidentTitle}
                  value={title}
                />
              </div>
              <div className="space-y-2">
                <div>
                  <p className="text-sm font-medium text-foreground">
                    {t('incidents.statusPage.field.components')}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {t('incidents.statusPage.field.componentsDescription')}
                  </p>
                </div>
                <ScrollArea className="h-36 rounded-lg border border-border">
                  <div className="space-y-1 p-2">
                    {components.length === 0 ? (
                      <p className="px-2 py-3 text-sm text-muted-foreground">
                        {t('incidents.statusPage.field.componentsEmpty')}
                      </p>
                    ) : (
                      components.map((component) => (
                        <label
                          key={component.id}
                          className="flex cursor-pointer items-start gap-3 rounded-md px-2 py-2 hover:bg-muted/50"
                        >
                          <input
                            checked={selectedComponentIds.includes(component.id)}
                            className="mt-1"
                            onChange={() => toggleComponentSelection(component.id)}
                            type="checkbox"
                          />
                          <span className="min-w-0">
                            <span className="block truncate text-sm font-medium text-foreground">
                              {component.name}
                            </span>
                            {component.description ? (
                              <span className="block truncate text-xs text-muted-foreground">
                                {component.description}
                              </span>
                            ) : null}
                          </span>
                        </label>
                      ))
                    )}
                  </div>
                </ScrollArea>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground" htmlFor="status-page-body">
                  {t('incidents.statusPage.field.body')}
                </label>
                <Textarea
                  id="status-page-body"
                  onChange={(event) => setBody(event.target.value)}
                  placeholder={t('incidents.statusPage.field.bodyPlaceholder')}
                  value={body}
                />
              </div>
            </DialogPanel>
            <DialogFooter>
              <DialogClose render={<Button type="button" variant="ghost" />}>
                {t('incidents.action.cancel')}
              </DialogClose>
              <Button
                disabled={publishing || components.length === 0}
                onClick={() => {
                  void handlePublish()
                }}
                type="button"
              >
                {publishing ? t('incidents.statusPage.publishing') : t('incidents.statusPage.publish')}
              </Button>
            </DialogFooter>
          </DialogPopup>
        </Dialog>
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

function IncidentListSkeletonRows(): ReactElement {
  return (
    <>
      {Array.from({ length: 5 }).map((_, index) => (
        <TableRow key={`incident-skeleton-${index}`}>
          <TableCell>
            <Skeleton className="h-4 max-w-xs" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-32" />
          </TableCell>
        </TableRow>
      ))}
    </>
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
  const [unassigningRoleId, setUnassigningRoleId] = useState<string | null>(null)

  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const currentUserId = meData?.me?.id ?? ''
  const isAdmin = meData?.me?.role === UserRole.Admin

  const [{ data: incidentsData, fetching: incidentsFetching }, reexecuteIncidentsQuery] =
    useIncidentsQuery({
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
  const [, unassignIncidentRole] = useUnassignIncidentRoleMutation()

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
      showMutationError(result.error, 'incidents.error.action')
      return
    }
    if (result.data?.addIncidentTimelineNote) {
      applyTimelineEvent(result.data.addIncidentTimelineNote)
      setNoteBody('')
      notifyMutationSuccess('incidents.toast.noteAdded')
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
      showMutationError(result.error, 'incidents.error.action')
      return
    }
    if (result.data?.updateIncidentStatus) {
      setTimelineEvents(sortTimelineEvents(result.data.updateIncidentStatus.timelineEvents))
      setIncidentStatus(result.data.updateIncidentStatus.status)
      notifyMutationSuccess('incidents.toast.statusUpdated')
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
      showMutationError(result.error, 'incidents.error.action')
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
      notifyMutationSuccess('incidents.toast.roleAssigned')
    }
  }

  async function handleUnassignRole(assignmentId: string): Promise<void> {
    setActionError(null)
    setUnassigningRoleId(assignmentId)
    const result = await unassignIncidentRole({ id: assignmentId })
    setUnassigningRoleId(null)
    if (result.error) {
      showMutationError(result.error, 'incidents.error.action')
      return
    }
    if (result.data?.unassignIncidentRole) {
      setRoleAssignments((current) => current.filter((assignment) => assignment.id !== assignmentId))
    }
  }

  return (
    <AppShell title={t('incidents.title')}>
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1.4fr)]">
        <section className="rounded-xl border border-border bg-card shadow-xs/5">
          <div className="flex items-center justify-between gap-3 border-b border-border p-4">
            <h2 className="text-sm font-medium text-foreground">{t('incidents.title')}</h2>
            <CreateIncidentDialog
              onCreated={(createdIncidentId) => {
                reexecuteIncidentsQuery({ requestPolicy: 'network-only' })
                navigate(`/incidents/${createdIncidentId}`)
              }}
            />
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
                {incidentsFetching && incidents.length === 0 ? <IncidentListSkeletonRows /> : null}
                {!incidentsFetching && incidents.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={3}>
                      <Empty>
                        <EmptyHeader>
                          <EmptyMedia variant="icon">
                            <SirenIcon />
                          </EmptyMedia>
                          <EmptyTitle>{t('incidents.empty')}</EmptyTitle>
                          <EmptyDescription>{t('incidents.empty.description')}</EmptyDescription>
                        </EmptyHeader>
                        <EmptyContent>
                          <CreateIncidentDialog
                            onCreated={(createdIncidentId) => {
                              reexecuteIncidentsQuery({ requestPolicy: 'network-only' })
                              navigate(`/incidents/${createdIncidentId}`)
                            }}
                          />
                        </EmptyContent>
                      </Empty>
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

              <PublishToStatusPageDialog
                incidentId={incident.id}
                incidentTitle={incident.title}
                isAdmin={isAdmin}
              />

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
                          onUnassign={() => {
                            void handleUnassignRole(assignment.id)
                          }}
                          roleDefinitionId={assignment.role.id}
                          roleName={assignment.role.name}
                          unassigningRoleId={unassigningRoleId}
                        />
                      ))
                    : roleDefinitions.map((roleDefinition) => {
                        const assignment = assignmentByRoleDefinitionId(
                          roleAssignments,
                          roleDefinition.id,
                        )
                        return (
                        <RoleSlot
                          key={roleDefinition.id}
                          assignment={assignment}
                          assigningRoleId={assigningRoleId}
                          onAssign={() => {
                            void handleAssignRole(roleDefinition.id)
                          }}
                          onUnassign={() => {
                            if (assignment) {
                              void handleUnassignRole(assignment.id)
                            }
                          }}
                          roleDefinitionId={roleDefinition.id}
                          roleName={roleDefinition.name}
                          unassigningRoleId={unassigningRoleId}
                        />
                        )
                      })}
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
