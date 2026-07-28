import { useEffect, useState, type ReactElement } from 'react'
import {
  HeartbeatMonitorStatus,
  useCreateHeartbeatMonitorMutation,
  useDeleteHeartbeatMonitorMutation,
  useHeartbeatMonitorsQuery,
  useUpdateHeartbeatMonitorMutation,
  type HeartbeatMonitorsQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  AlertDialog,
  AlertDialogClose,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogPopup,
  AlertDialogTitle,
  AlertDialogTrigger,
  AlertTitle,
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
  Input,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@escalite/ui'
import { AlertTriangleIcon, CircleCheckIcon, InfoIcon, PlusIcon } from 'lucide-react'

import { appConfig } from '../lib/config'
import { formatDateTime, formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'
import { CopyButton } from './copy-button'

type HeartbeatMonitorRow = HeartbeatMonitorsQuery['heartbeatMonitors'][number]

type HeartbeatFormState = {
  name: string
  intervalSeconds: string
  graceSeconds: string
}

type RevealPingUrl = {
  title: string
  description: string
  pingURL: string
  tokenPrefix: string
}

const DEFAULT_INTERVAL_SECONDS = '3600'
const DEFAULT_GRACE_SECONDS = '300'

function emptyForm(): HeartbeatFormState {
  return {
    name: '',
    intervalSeconds: DEFAULT_INTERVAL_SECONDS,
    graceSeconds: DEFAULT_GRACE_SECONDS,
  }
}

function buildHeartbeatPingURL(apiBaseUrl: string, token: string): string {
  const base = apiBaseUrl.replace(/\/+$/, '')
  return `${base}/heartbeat/${token}`
}

function heartbeatStatusBadge(status: HeartbeatMonitorStatus): ReactElement {
  switch (status) {
    case HeartbeatMonitorStatus.Overdue:
      return (
        <Badge variant="warning">
          <AlertTriangleIcon />
          {t('heartbeats.status.overdue')}
        </Badge>
      )
    case HeartbeatMonitorStatus.Triggered:
      return (
        <Badge variant="error">
          <AlertTriangleIcon />
          {t('heartbeats.status.triggered')}
        </Badge>
      )
    default:
      return (
        <Badge variant="success">
          <CircleCheckIcon />
          {t('heartbeats.status.healthy')}
        </Badge>
      )
  }
}

type HeartbeatMonitorsPanelProps = {
  serviceId: string
}

export function HeartbeatMonitorsPanel({ serviceId }: HeartbeatMonitorsPanelProps): ReactElement {
  const [{ data, fetching, error }, reexecute] = useHeartbeatMonitorsQuery({
    variables: { serviceId },
    requestPolicy: 'network-only',
  })

  const [, createMonitor] = useCreateHeartbeatMonitorMutation()
  const [, updateMonitor] = useUpdateHeartbeatMonitorMutation()
  const [, deleteMonitor] = useDeleteHeartbeatMonitorMutation()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<HeartbeatMonitorRow | null>(null)
  const [form, setForm] = useState<HeartbeatFormState>(emptyForm)
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [revealPingUrl, setRevealPingUrl] = useState<RevealPingUrl | null>(null)

  const monitors = data?.heartbeatMonitors ?? []

  useEffect(() => {
    if (!formOpen) {
      setEditing(null)
      setForm(emptyForm())
      setFormError(null)
    }
  }, [formOpen])

  function openCreate(): void {
    setEditing(null)
    setForm(emptyForm())
    setFormError(null)
    setFormOpen(true)
  }

  function openEdit(monitor: HeartbeatMonitorRow): void {
    setEditing(monitor)
    setForm({
      name: monitor.name,
      intervalSeconds: String(monitor.intervalSeconds),
      graceSeconds: String(monitor.graceSeconds),
    })
    setFormError(null)
    setFormOpen(true)
  }

  async function handleSave(): Promise<void> {
    const name = form.name.trim()
    if (!name) {
      setFormError(t('heartbeats.error.nameRequired'))
      return
    }

    const intervalSeconds = Number.parseInt(form.intervalSeconds, 10)
    if (!Number.isFinite(intervalSeconds) || intervalSeconds < 1) {
      setFormError(t('heartbeats.error.interval'))
      return
    }

    const graceSeconds = Number.parseInt(form.graceSeconds, 10)
    if (!Number.isFinite(graceSeconds) || graceSeconds < 0) {
      setFormError(t('heartbeats.error.grace'))
      return
    }

    setSaving(true)
    setFormError(null)

    if (editing) {
      const result = await updateMonitor({
        input: {
          id: editing.id,
          name,
          intervalSeconds,
          graceSeconds,
        },
      })
      setSaving(false)

      if (result.error) {
        setFormError(formatGraphQLError(result.error.message))
        return
      }

      setFormOpen(false)
      reexecute({ requestPolicy: 'network-only' })
      return
    }

    const result = await createMonitor({
      input: {
        serviceId,
        name,
        intervalSeconds,
        graceSeconds,
      },
    })
    setSaving(false)

    if (result.error) {
      setFormError(formatGraphQLError(result.error.message))
      return
    }

    setFormOpen(false)
    reexecute({ requestPolicy: 'network-only' })

    const created = result.data?.createHeartbeatMonitor
    if (!created?.token) {
      setFormError(t('heartbeats.error.createNoToken'))
      return
    }

    setRevealPingUrl({
      title: t('heartbeats.reveal.createdTitle'),
      description: t('heartbeats.reveal.createdDescription'),
      tokenPrefix: created.tokenPrefix,
      pingURL: buildHeartbeatPingURL(appConfig.apiPublicUrl, created.token),
    })
  }

  async function handleDelete(id: string): Promise<void> {
    const result = await deleteMonitor({ id })
    if (result.error) {
      setFormError(formatGraphQLError(result.error.message))
      return
    }
    reexecute({ requestPolicy: 'network-only' })
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-foreground">{t('heartbeats.title')}</h2>
          <p className="text-sm text-muted-foreground">{t('heartbeats.description')}</p>
        </div>
        <Dialog onOpenChange={setFormOpen} open={formOpen}>
          <DialogTrigger render={<Button onClick={openCreate} type="button" />}>
            <PlusIcon />
            {t('heartbeats.add')}
          </DialogTrigger>
          <DialogPopup>
            <DialogHeader>
              <DialogTitle>
                {editing ? t('heartbeats.editTitle') : t('heartbeats.createTitle')}
              </DialogTitle>
              <DialogDescription>{t('heartbeats.formDescription')}</DialogDescription>
            </DialogHeader>
            <DialogPanel className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground" htmlFor="heartbeat-name">
                  {t('common.name')}
                </label>
                <Input
                  id="heartbeat-name"
                  onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                  value={form.name}
                />
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="heartbeat-interval">
                    {t('heartbeats.field.interval')}
                  </label>
                  <Input
                    id="heartbeat-interval"
                    inputMode="numeric"
                    min={1}
                    onChange={(event) =>
                      setForm((current) => ({ ...current, intervalSeconds: event.target.value }))
                    }
                    type="number"
                    value={form.intervalSeconds}
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="heartbeat-grace">
                    {t('heartbeats.field.grace')}
                  </label>
                  <Input
                    id="heartbeat-grace"
                    inputMode="numeric"
                    min={0}
                    onChange={(event) =>
                      setForm((current) => ({ ...current, graceSeconds: event.target.value }))
                    }
                    type="number"
                    value={form.graceSeconds}
                  />
                </div>
              </div>
              {formError ? (
                <Alert variant="error">
                  <AlertTriangleIcon />
                  <AlertDescription>{formError}</AlertDescription>
                </Alert>
              ) : null}
            </DialogPanel>
            <DialogFooter>
              <DialogClose render={<Button type="button" variant="ghost" />}>{t('common.cancel')}</DialogClose>
              <Button disabled={saving} onClick={() => void handleSave()} type="button">
                {saving
                  ? t('heartbeats.action.saving')
                  : editing
                    ? t('heartbeats.action.saveChanges')
                    : t('heartbeats.action.create')}
              </Button>
            </DialogFooter>
          </DialogPopup>
        </Dialog>
      </div>

      {fetching ? <p className="text-sm text-muted-foreground">{t('heartbeats.loading')}</p> : null}

      {error ? (
        <Alert variant="error">
          <AlertTriangleIcon />
          <AlertDescription>{formatGraphQLError(error.message)}</AlertDescription>
        </Alert>
      ) : null}

      {formError && !formOpen ? (
        <Alert variant="error">
          <AlertTriangleIcon />
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      {!fetching && !error && monitors.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t('heartbeats.empty')}</p>
      ) : null}

      {monitors.length > 0 ? (
        <Table variant="card">
          <TableHeader>
            <TableRow>
              <TableHead>{t('common.name')}</TableHead>
              <TableHead>{t('heartbeats.column.interval')}</TableHead>
              <TableHead>{t('heartbeats.column.grace')}</TableHead>
              <TableHead>{t('heartbeats.column.tokenPrefix')}</TableHead>
              <TableHead>{t('common.status')}</TableHead>
              <TableHead>{t('heartbeats.column.lastPing')}</TableHead>
              <TableHead className="text-right">{t('common.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {monitors.map((monitor) => (
              <TableRow key={monitor.id}>
                <TableCell className="font-medium text-foreground">{monitor.name}</TableCell>
                <TableCell>{monitor.intervalSeconds}s</TableCell>
                <TableCell>{monitor.graceSeconds}s</TableCell>
                <TableCell>
                  <code className="rounded bg-muted px-2 py-1 text-xs">{monitor.tokenPrefix}</code>
                </TableCell>
                <TableCell>{heartbeatStatusBadge(monitor.status)}</TableCell>
                <TableCell className="text-muted-foreground">
                  {monitor.lastPingAt ? formatDateTime(monitor.lastPingAt) : '—'}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-2">
                    <Button onClick={() => openEdit(monitor)} size="sm" type="button" variant="outline">
                      {t('heartbeats.action.edit')}
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger
                        render={<Button size="sm" type="button" variant="destructive-outline" />}
                      >
                        {t('heartbeats.action.delete')}
                      </AlertDialogTrigger>
                      <AlertDialogPopup>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{t('heartbeats.delete.title')}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {t('heartbeats.delete.description', { prefix: monitor.tokenPrefix })}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                            {t('common.cancel')}
                          </AlertDialogClose>
                          <AlertDialogClose
                            onClick={() => void handleDelete(monitor.id)}
                            render={<Button type="button" variant="destructive" />}
                          >
                            {t('heartbeats.delete.confirm')}
                          </AlertDialogClose>
                        </AlertDialogFooter>
                      </AlertDialogPopup>
                    </AlertDialog>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}

      <Dialog onOpenChange={(open) => !open && setRevealPingUrl(null)} open={revealPingUrl !== null}>
        <DialogPopup>
          {revealPingUrl ? (
            <>
              <DialogHeader>
                <DialogTitle>{revealPingUrl.title}</DialogTitle>
                <DialogDescription>{revealPingUrl.description}</DialogDescription>
              </DialogHeader>
              <DialogPanel className="space-y-4">
                <Alert variant="warning">
                  <InfoIcon />
                  <AlertTitle>{t('common.shownOnce')}</AlertTitle>
                  <AlertDescription>
                    {t('heartbeats.reveal.shownOnceDescription', { prefix: revealPingUrl.tokenPrefix })}
                  </AlertDescription>
                </Alert>
                <div className="space-y-2">
                  <p className="text-sm font-medium text-foreground">{t('heartbeats.reveal.pingUrl')}</p>
                  <code className="block overflow-x-auto rounded-md bg-muted p-3 text-xs text-foreground">
                    {revealPingUrl.pingURL}
                  </code>
                  <CopyButton label={t('common.copyUrl')} value={revealPingUrl.pingURL} />
                </div>
              </DialogPanel>
              <DialogFooter>
                <DialogClose render={<Button type="button" />}>{t('common.savedUrl')}</DialogClose>
              </DialogFooter>
            </>
          ) : null}
        </DialogPopup>
      </Dialog>
    </div>
  )
}
