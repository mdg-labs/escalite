import { useEffect, useState, type ReactElement } from 'react'
import {
  useCreateMaintenanceWindowMutation,
  useDeleteMaintenanceWindowMutation,
  useMaintenanceWindowsQuery,
  useUpdateMaintenanceWindowMutation,
  type MaintenanceWindowsQuery,
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
import { AlertTriangleIcon, PlusIcon } from 'lucide-react'

import { formatDateTime, formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

type MaintenanceWindowRow = MaintenanceWindowsQuery['maintenanceWindows'][number]

type MaintenanceFormState = {
  description: string
  startsAt: string
  endsAt: string
  suppressNotifications: boolean
  suppressIngestion: boolean
}

function emptyForm(): MaintenanceFormState {
  const now = new Date()
  const later = new Date(now.getTime() + 60 * 60 * 1000)
  return {
    description: '',
    startsAt: toLocalInputValue(now),
    endsAt: toLocalInputValue(later),
    suppressNotifications: true,
    suppressIngestion: false,
  }
}

function toLocalInputValue(date: Date): string {
  const offset = date.getTimezoneOffset()
  const local = new Date(date.getTime() - offset * 60_000)
  return local.toISOString().slice(0, 16)
}

function fromLocalInputValue(value: string): string {
  return new Date(value).toISOString()
}

function isActive(window: MaintenanceWindowRow, now: Date): boolean {
  const startsAt = new Date(window.startsAt)
  const endsAt = new Date(window.endsAt)
  return startsAt <= now && endsAt > now
}

type MaintenanceWindowsPanelProps = {
  serviceId: string
}

export function MaintenanceWindowsPanel({ serviceId }: MaintenanceWindowsPanelProps): ReactElement {
  const [{ data, fetching, error }, reexecute] = useMaintenanceWindowsQuery({
    variables: { serviceId },
    requestPolicy: 'network-only',
  })

  const [, createWindow] = useCreateMaintenanceWindowMutation()
  const [, updateWindow] = useUpdateMaintenanceWindowMutation()
  const [, deleteWindow] = useDeleteMaintenanceWindowMutation()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<MaintenanceWindowRow | null>(null)
  const [form, setForm] = useState<MaintenanceFormState>(emptyForm)
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const windows = data?.maintenanceWindows ?? []
  const now = new Date()

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

  function openEdit(window: MaintenanceWindowRow): void {
    setEditing(window)
    setForm({
      description: window.description,
      startsAt: toLocalInputValue(new Date(window.startsAt)),
      endsAt: toLocalInputValue(new Date(window.endsAt)),
      suppressNotifications: window.suppressNotifications,
      suppressIngestion: window.suppressIngestion,
    })
    setFormError(null)
    setFormOpen(true)
  }

  async function handleSave(): Promise<void> {
    const description = form.description.trim()
    if (!description) {
      setFormError(t('services.maintenance.error.required'))
      return
    }

    setSaving(true)
    setFormError(null)

    const input = {
      description,
      startsAt: fromLocalInputValue(form.startsAt),
      endsAt: fromLocalInputValue(form.endsAt),
      suppressNotifications: form.suppressNotifications,
      suppressIngestion: form.suppressIngestion,
    }

    const result = editing
      ? await updateWindow({ input: { id: editing.id, ...input } })
      : await createWindow({ input: { serviceId, ...input } })

    setSaving(false)

    if (result.error) {
      setFormError(formatGraphQLError(result.error.message))
      return
    }

    setFormOpen(false)
    reexecute({ requestPolicy: 'network-only' })
  }

  async function handleDelete(id: string): Promise<void> {
    const result = await deleteWindow({ id })
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
          <h2 className="text-lg font-semibold text-foreground">{t('services.maintenance.title')}</h2>
          <p className="text-sm text-muted-foreground">{t('services.maintenance.description')}</p>
        </div>
        <Dialog onOpenChange={setFormOpen} open={formOpen}>
          <DialogTrigger render={<Button onClick={openCreate} type="button" />}>
            <PlusIcon />
            {t('services.maintenance.create')}
          </DialogTrigger>
          <DialogPopup>
            <DialogHeader>
              <DialogTitle>
                {editing ? t('services.maintenance.edit') : t('services.maintenance.create')}
              </DialogTitle>
              <DialogDescription>{t('services.maintenance.formDescription')}</DialogDescription>
            </DialogHeader>
            <DialogPanel className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground" htmlFor="maintenance-description">
                  {t('services.maintenance.field.description')}
                </label>
                <Input
                  id="maintenance-description"
                  onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
                  value={form.description}
                />
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="maintenance-starts">
                    {t('services.maintenance.field.startsAt')}
                  </label>
                  <Input
                    id="maintenance-starts"
                    onChange={(event) => setForm((current) => ({ ...current, startsAt: event.target.value }))}
                    type="datetime-local"
                    value={form.startsAt}
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="maintenance-ends">
                    {t('services.maintenance.field.endsAt')}
                  </label>
                  <Input
                    id="maintenance-ends"
                    onChange={(event) => setForm((current) => ({ ...current, endsAt: event.target.value }))}
                    type="datetime-local"
                    value={form.endsAt}
                  />
                </div>
              </div>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input
                  checked={form.suppressNotifications}
                  className="size-4 rounded border border-border"
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      suppressNotifications: event.target.checked,
                    }))
                  }
                  type="checkbox"
                />
                {t('services.maintenance.field.suppressNotifications')}
              </label>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input
                  checked={form.suppressIngestion}
                  className="size-4 rounded border border-border"
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      suppressIngestion: event.target.checked,
                    }))
                  }
                  type="checkbox"
                />
                {t('services.maintenance.field.suppressIngestion')}
              </label>
              {formError ? (
                <Alert variant="error">
                  <AlertTriangleIcon />
                  <AlertDescription>{formError}</AlertDescription>
                </Alert>
              ) : null}
            </DialogPanel>
            <DialogFooter>
              <DialogClose render={<Button type="button" variant="ghost" />}>
                {t('services.action.cancel')}
              </DialogClose>
              <Button disabled={saving} onClick={() => void handleSave()} type="button">
                {saving ? t('services.action.saving') : t('services.action.save')}
              </Button>
            </DialogFooter>
          </DialogPopup>
        </Dialog>
      </div>

      {fetching ? <p className="text-sm text-muted-foreground">{t('services.maintenance.loading')}</p> : null}

      {error ? (
        <Alert variant="error">
          <AlertTriangleIcon />
          <AlertDescription>{formatGraphQLError(error.message)}</AlertDescription>
        </Alert>
      ) : null}

      {!fetching && !error && windows.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t('services.maintenance.empty')}</p>
      ) : null}

      {windows.length > 0 ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('services.maintenance.field.description')}</TableHead>
              <TableHead>{t('services.maintenance.field.startsAt')}</TableHead>
              <TableHead>{t('services.maintenance.field.endsAt')}</TableHead>
              <TableHead>{t('services.maintenance.column.status')}</TableHead>
              <TableHead className="text-right">{t('services.column.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {windows.map((window) => (
              <TableRow key={window.id}>
                <TableCell>{window.description}</TableCell>
                <TableCell>{formatDateTime(window.startsAt)}</TableCell>
                <TableCell>{formatDateTime(window.endsAt)}</TableCell>
                <TableCell>
                  {isActive(window, now)
                    ? t('services.maintenance.status.active')
                    : t('services.maintenance.status.scheduled')}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-2">
                    <Button onClick={() => openEdit(window)} size="sm" type="button" variant="outline">
                      {t('services.maintenance.edit')}
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger render={<Button size="sm" type="button" variant="destructive-outline" />}>
                        {t('services.action.delete')}
                      </AlertDialogTrigger>
                      <AlertDialogPopup>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{t('services.maintenance.delete.title')}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {t('services.maintenance.delete.description')}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                            {t('services.action.cancel')}
                          </AlertDialogClose>
                          <AlertDialogClose
                            onClick={() => void handleDelete(window.id)}
                            render={<Button type="button" variant="destructive" />}
                          >
                            {t('services.action.deleteConfirm')}
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
    </div>
  )
}
