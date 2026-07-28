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
          Overdue
        </Badge>
      )
    case HeartbeatMonitorStatus.Triggered:
      return (
        <Badge variant="error">
          <AlertTriangleIcon />
          Triggered
        </Badge>
      )
    default:
      return (
        <Badge variant="success">
          <CircleCheckIcon />
          Healthy
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
      setFormError('Name is required.')
      return
    }

    const intervalSeconds = Number.parseInt(form.intervalSeconds, 10)
    if (!Number.isFinite(intervalSeconds) || intervalSeconds < 1) {
      setFormError('Interval must be at least 1 second.')
      return
    }

    const graceSeconds = Number.parseInt(form.graceSeconds, 10)
    if (!Number.isFinite(graceSeconds) || graceSeconds < 0) {
      setFormError('Grace period must be zero or more seconds.')
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
      setFormError('Monitor was created but the ping token was not returned.')
      return
    }

    setRevealPingUrl({
      title: 'Heartbeat monitor created',
      description:
        'Save the ping URL below. After you close this dialog, only the token prefix remains visible.',
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
          <h2 className="text-lg font-semibold text-foreground">Heartbeat monitors</h2>
          <p className="text-sm text-muted-foreground">
            Dead man&apos;s switch monitors that alert when expected pings stop arriving.
          </p>
        </div>
        <Dialog onOpenChange={setFormOpen} open={formOpen}>
          <DialogTrigger render={<Button onClick={openCreate} type="button" />}>
            <PlusIcon />
            Add monitor
          </DialogTrigger>
          <DialogPopup>
            <DialogHeader>
              <DialogTitle>{editing ? 'Edit heartbeat monitor' : 'Add heartbeat monitor'}</DialogTitle>
              <DialogDescription>
                Configure how often a ping is expected and how long to wait before alerting.
              </DialogDescription>
            </DialogHeader>
            <DialogPanel className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground" htmlFor="heartbeat-name">
                  Name
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
                    Interval (seconds)
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
                    Grace (seconds)
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
              <DialogClose render={<Button type="button" variant="ghost" />}>Cancel</DialogClose>
              <Button disabled={saving} onClick={() => void handleSave()} type="button">
                {saving ? 'Saving…' : editing ? 'Save changes' : 'Create monitor'}
              </Button>
            </DialogFooter>
          </DialogPopup>
        </Dialog>
      </div>

      {fetching ? <p className="text-sm text-muted-foreground">Loading heartbeat monitors…</p> : null}

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
        <p className="text-sm text-muted-foreground">No heartbeat monitors configured.</p>
      ) : null}

      {monitors.length > 0 ? (
        <Table variant="card">
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Interval</TableHead>
              <TableHead>Grace</TableHead>
              <TableHead>Token prefix</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Last ping</TableHead>
              <TableHead className="text-right">Actions</TableHead>
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
                      Edit
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger
                        render={<Button size="sm" type="button" variant="destructive-outline" />}
                      >
                        Delete
                      </AlertDialogTrigger>
                      <AlertDialogPopup>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete heartbeat monitor?</AlertDialogTitle>
                          <AlertDialogDescription>
                            Pings to token prefix{' '}
                            <span className="font-mono text-foreground">{monitor.tokenPrefix}</span> will
                            stop updating this monitor.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                            Cancel
                          </AlertDialogClose>
                          <AlertDialogClose
                            onClick={() => void handleDelete(monitor.id)}
                            render={<Button type="button" variant="destructive" />}
                          >
                            Delete monitor
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
                  <AlertTitle>Shown once</AlertTitle>
                  <AlertDescription>
                    Token prefix{' '}
                    <span className="font-mono text-foreground">{revealPingUrl.tokenPrefix}</span> will
                    remain visible in the monitor list after you close this dialog.
                  </AlertDescription>
                </Alert>
                <div className="space-y-2">
                  <p className="text-sm font-medium text-foreground">Ping URL</p>
                  <code className="block overflow-x-auto rounded-md bg-muted p-3 text-xs text-foreground">
                    {revealPingUrl.pingURL}
                  </code>
                  <CopyButton label="Copy URL" value={revealPingUrl.pingURL} />
                </div>
              </DialogPanel>
              <DialogFooter>
                <DialogClose render={<Button type="button" />}>I saved the URL</DialogClose>
              </DialogFooter>
            </>
          ) : null}
        </DialogPopup>
      </Dialog>
    </div>
  )
}
