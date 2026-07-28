import { useEffect, useMemo, useState, type ReactElement } from 'react'
import {
  IncidentStatus,
  useCreateStatusPageIncidentUpdateMutation,
  useStatusPageQuery,
  useUpdateStatusPageIncidentStatusMutation,
  type StatusPageQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
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
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
  Textarea,
} from '@escalite/ui'
import { AlertTriangleIcon, ExternalLinkIcon, MegaphoneIcon } from 'lucide-react'

import { formatDateTime, formatGraphQLError } from '../lib/format'
import { incidentStatusLabel } from '../lib/incidents'
import { t } from '../lib/i18n'
import { statusPagePublicAppUrl } from '../lib/status-page'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

type IncidentRow = NonNullable<StatusPageQuery['statusPage']>['incidents'][number]

const INCIDENT_STATUS_OPTIONS: IncidentStatus[] = [
  IncidentStatus.Investigating,
  IncidentStatus.Identified,
  IncidentStatus.Monitoring,
  IncidentStatus.Resolved,
]

function incidentStatusBadgeVariant(
  status: IncidentStatus,
): 'secondary' | 'info' | 'warning' | 'destructive' {
  switch (status) {
    case IncidentStatus.Resolved:
      return 'secondary'
    case IncidentStatus.Monitoring:
      return 'info'
    case IncidentStatus.Identified:
      return 'warning'
    default:
      return 'destructive'
  }
}

function IncidentStatusBadge({ status }: { status: IncidentStatus }): ReactElement {
  return (
    <Badge variant={incidentStatusBadgeVariant(status)}>
      {t(incidentStatusLabel(status) as Parameters<typeof t>[0])}
    </Badge>
  )
}

function IncidentCard({
  incident,
  componentNamesById,
  actionIncidentId,
  onPostUpdate,
  onResolve,
}: {
  incident: IncidentRow
  componentNamesById: Map<string, string>
  actionIncidentId: string | null
  onPostUpdate: (incident: IncidentRow) => void
  onResolve: (incident: IncidentRow) => void
}): ReactElement {
  const affectedNames = incident.affectedComponentIds
    .map((componentId) => componentNamesById.get(componentId))
    .filter((name): name is string => Boolean(name))
  const isBusy = actionIncidentId === incident.id

  return (
    <article className="rounded-lg border border-border bg-muted/20 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <h3 className="text-base font-semibold text-foreground">{incident.title}</h3>
          <p className="text-sm text-muted-foreground">
            {t('statusPages.incidents.field.updated', { date: formatDateTime(incident.updatedAt) })}
          </p>
        </div>
        <IncidentStatusBadge status={incident.status} />
      </div>

      {affectedNames.length > 0 ? (
        <div className="mt-4 space-y-2">
          <p className="text-sm font-medium text-foreground">{t('statusPages.incidents.components')}</p>
          <div className="flex flex-wrap gap-2">
            {affectedNames.map((name) => (
              <Badge key={name} variant="outline">
                {name}
              </Badge>
            ))}
          </div>
        </div>
      ) : null}

      {incident.updates.length > 0 ? (
        <div className="mt-4 space-y-3">
          <p className="text-sm font-medium text-foreground">{t('statusPages.incidents.updates')}</p>
          <ol className="space-y-3">
            {incident.updates.map((update) => (
              <li className="rounded-lg border border-border bg-card p-3" key={update.id}>
                <div className="mb-2 flex flex-wrap items-center gap-2">
                  <IncidentStatusBadge status={update.status} />
                  <span className="text-xs text-muted-foreground">
                    {formatDateTime(update.createdAt)}
                  </span>
                </div>
                <p className="whitespace-pre-wrap text-sm text-foreground">{update.body}</p>
              </li>
            ))}
          </ol>
        </div>
      ) : null}

      <div className="mt-4 flex flex-wrap gap-2">
        <Button disabled={isBusy} onClick={() => onPostUpdate(incident)} size="sm" type="button">
          {t('statusPages.incidents.action.postUpdate')}
        </Button>
        <Button
          disabled={isBusy}
          onClick={() => onResolve(incident)}
          size="sm"
          type="button"
          variant="outline"
        >
          {t('statusPages.incidents.action.resolve')}
        </Button>
      </div>
    </article>
  )
}

export function StatusPageIncidentsPanel(): ReactElement {
  const [{ data, fetching, error }, reexecuteQuery] = useStatusPageQuery({
    requestPolicy: 'network-only',
  })
  const [, createStatusPageIncidentUpdate] = useCreateStatusPageIncidentUpdateMutation()
  const [, updateStatusPageIncidentStatus] = useUpdateStatusPageIncidentStatusMutation()

  const [updateDialogOpen, setUpdateDialogOpen] = useState(false)
  const [selectedIncident, setSelectedIncident] = useState<IncidentRow | null>(null)
  const [updateStatus, setUpdateStatus] = useState<IncidentStatus>(IncidentStatus.Investigating)
  const [updateBody, setUpdateBody] = useState('')
  const [resolveDialogOpen, setResolveDialogOpen] = useState(false)
  const [resolveBody, setResolveBody] = useState('')
  const [formError, setFormError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [actionIncidentId, setActionIncidentId] = useState<string | null>(null)

  const statusPage = data?.statusPage
  const incidents = useMemo(() => statusPage?.incidents ?? [], [statusPage?.incidents])
  const componentNamesById = useMemo(
    () => new Map((statusPage?.components ?? []).map((component) => [component.id, component.name])),
    [statusPage?.components],
  )
  const publicUrl = statusPage?.slug ? statusPagePublicAppUrl(statusPage.slug) : null

  useEffect(() => {
    if (!updateDialogOpen) {
      setSelectedIncident(null)
      setUpdateStatus(IncidentStatus.Investigating)
      setUpdateBody('')
      setFormError(null)
    }
  }, [updateDialogOpen])

  useEffect(() => {
    if (!resolveDialogOpen) {
      setSelectedIncident(null)
      setResolveBody('')
      setFormError(null)
    }
  }, [resolveDialogOpen])

  function refresh(): void {
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  function openPostUpdate(incident: IncidentRow): void {
    setSelectedIncident(incident)
    setUpdateStatus(incident.status)
    setUpdateBody('')
    setFormError(null)
    setUpdateDialogOpen(true)
  }

  function openResolve(incident: IncidentRow): void {
    setSelectedIncident(incident)
    setResolveBody('')
    setFormError(null)
    setResolveDialogOpen(true)
  }

  async function handlePostUpdate(): Promise<void> {
    if (!selectedIncident) {
      return
    }

    const trimmedBody = updateBody.trim()
    if (!trimmedBody) {
      setFormError(t('statusPages.incidents.validation.requiredBody'))
      return
    }

    setSaving(true)
    setFormError(null)

    const result = await createStatusPageIncidentUpdate({
      input: {
        statusPageIncidentId: selectedIncident.id,
        body: trimmedBody,
        status: updateStatus,
      },
    })

    setSaving(false)

    if (result.error) {
      showMutationError(result.error, 'statusPage.error.action')
      return
    }

    notifyMutationSuccess('statusPage.toast.incidentUpdatePosted')
    setUpdateDialogOpen(false)
    refresh()
  }

  async function handleResolve(incident: IncidentRow): Promise<void> {
    setActionError(null)
    setFormError(null)
    setSaving(true)
    setActionIncidentId(incident.id)

    const trimmedBody = resolveBody.trim()
    const result = await updateStatusPageIncidentStatus({
      input: {
        id: incident.id,
        status: IncidentStatus.Resolved,
        body: trimmedBody || undefined,
      },
    })

    setSaving(false)
    setActionIncidentId(null)

    if (result.error) {
      showMutationError(result.error, 'statusPage.error.action')
      return
    }

    notifyMutationSuccess('statusPage.toast.incidentStatusUpdated')
    setResolveDialogOpen(false)
    refresh()
  }

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex items-start gap-3">
          <MegaphoneIcon className="mt-0.5 size-5 text-muted-foreground" />
          <div>
            <h2 className="text-xl font-semibold text-foreground">{t('statusPages.incidents.title')}</h2>
            <p className="mt-2 text-sm text-muted-foreground">{t('statusPages.incidents.description')}</p>
          </div>
        </div>

        {publicUrl ? (
          <a
            className="inline-flex items-center gap-1 text-sm font-medium text-primary underline-offset-4 hover:underline"
            href={publicUrl}
            rel="noreferrer"
            target="_blank"
          >
            {t('statusPages.incidents.viewPublic')}
            <ExternalLinkIcon className="size-3.5" />
          </a>
        ) : null}
      </div>

      {(error || actionError) ? (
        <Alert className="mt-6" variant="error">
          <AlertDescription>
            {actionError ?? (error ? formatGraphQLError(error.message) : null)}
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="mt-6">
        {fetching && !statusPage ? (
          <p className="text-sm text-muted-foreground">{t('statusPages.incidents.loading')}</p>
        ) : !statusPage ? (
          <p className="text-sm text-muted-foreground">{t('statusPages.incidents.notConfigured')}</p>
        ) : incidents.length === 0 ? (
          <div className="flex items-start gap-3 rounded-lg border border-dashed border-border p-4">
            <AlertTriangleIcon className="mt-0.5 size-4 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">{t('statusPages.incidents.empty')}</p>
          </div>
        ) : (
          <div className="space-y-4">
            {incidents.map((incident) => (
              <IncidentCard
                actionIncidentId={actionIncidentId}
                componentNamesById={componentNamesById}
                incident={incident}
                key={incident.id}
                onPostUpdate={openPostUpdate}
                onResolve={openResolve}
              />
            ))}
          </div>
        )}
      </div>

      <Dialog onOpenChange={setUpdateDialogOpen} open={updateDialogOpen}>
        <DialogPopup>
          <DialogHeader>
            <DialogTitle>{t('statusPages.incidents.update.title')}</DialogTitle>
            <DialogDescription>
              {selectedIncident
                ? t('statusPages.incidents.update.description', { title: selectedIncident.title })
                : null}
            </DialogDescription>
          </DialogHeader>
          <DialogPanel className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground" htmlFor="status-page-incident-status">
                {t('statusPages.incidents.update.field.status')}
              </label>
              <Select
                onValueChange={(value) =>
                  setUpdateStatus((value as IncidentStatus) ?? IncidentStatus.Investigating)
                }
                value={updateStatus}
              >
                <SelectTrigger id="status-page-incident-status">
                  <SelectValue />
                </SelectTrigger>
                <SelectPopup>
                  {INCIDENT_STATUS_OPTIONS.map((option) => (
                    <SelectItem key={option} value={option}>
                      {t(incidentStatusLabel(option) as Parameters<typeof t>[0])}
                    </SelectItem>
                  ))}
                </SelectPopup>
              </Select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground" htmlFor="status-page-incident-body">
                {t('statusPages.incidents.update.field.body')}
              </label>
              <Textarea
                id="status-page-incident-body"
                onChange={(event) => setUpdateBody(event.target.value)}
                placeholder={t('statusPages.incidents.update.field.bodyPlaceholder')}
                rows={4}
                value={updateBody}
              />
            </div>

            {formError ? (
              <Alert variant="error">
                <AlertDescription>{formError}</AlertDescription>
              </Alert>
            ) : null}
          </DialogPanel>
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="ghost" />}>
              {t('statusPages.incidents.action.cancel')}
            </DialogClose>
            <Button disabled={saving} onClick={() => void handlePostUpdate()} type="button">
              {saving
                ? t('statusPages.incidents.action.posting')
                : t('statusPages.incidents.action.post')}
            </Button>
          </DialogFooter>
        </DialogPopup>
      </Dialog>

      <Dialog onOpenChange={setResolveDialogOpen} open={resolveDialogOpen}>
        <DialogPopup>
          <DialogHeader>
            <DialogTitle>{t('statusPages.incidents.resolve.title')}</DialogTitle>
            <DialogDescription>
              {selectedIncident
                ? t('statusPages.incidents.resolve.description', { title: selectedIncident.title })
                : null}
            </DialogDescription>
          </DialogHeader>
          <DialogPanel className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground" htmlFor="status-page-resolve-body">
                {t('statusPages.incidents.resolve.field.body')}
              </label>
              <Textarea
                id="status-page-resolve-body"
                onChange={(event) => setResolveBody(event.target.value)}
                placeholder={t('statusPages.incidents.resolve.field.bodyPlaceholder')}
                rows={4}
                value={resolveBody}
              />
            </div>

            {formError ? (
              <Alert variant="error">
                <AlertDescription>{formError}</AlertDescription>
              </Alert>
            ) : null}
          </DialogPanel>
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="ghost" />}>
              {t('statusPages.incidents.action.cancel')}
            </DialogClose>
            <Button
              disabled={saving || !selectedIncident}
              onClick={() => selectedIncident && void handleResolve(selectedIncident)}
              type="button"
              variant="destructive"
            >
              {saving
                ? t('statusPages.incidents.action.resolving')
                : t('statusPages.incidents.action.confirmResolve')}
            </Button>
          </DialogFooter>
        </DialogPopup>
      </Dialog>
    </section>
  )
}
