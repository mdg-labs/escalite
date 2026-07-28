import { useEffect, useMemo, useState, type ReactElement } from 'react'
import {
  StatusPageComponentStatus,
  useCreateStatusPageComponentMutation,
  useDeleteStatusPageComponentMutation,
  useServicesQuery,
  useStatusPageQuery,
  useUpdateStatusPageComponentMutation,
  type StatusPageQuery,
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
  Badge,
  Button,
  Combobox,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxPopup,
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
import {
  AlertTriangleIcon,
  ArrowDownIcon,
  ArrowUpIcon,
  CheckCircle2Icon,
  CircleAlertIcon,
  CircleXIcon,
  LayersIcon,
  PlusIcon,
} from 'lucide-react'

import { formatGraphQLError } from '../lib/format'
import { t, type MessageKey } from '../lib/i18n'

type ComponentRow = NonNullable<StatusPageQuery['statusPage']>['components'][number]

const COMPONENT_STATUS_OPTIONS: StatusPageComponentStatus[] = [
  StatusPageComponentStatus.Operational,
  StatusPageComponentStatus.Degraded,
  StatusPageComponentStatus.PartialOutage,
  StatusPageComponentStatus.MajorOutage,
]

const COMPONENT_STATUS_LABEL_KEYS: Record<StatusPageComponentStatus, MessageKey> = {
  [StatusPageComponentStatus.Operational]: 'statusPages.components.status.operational',
  [StatusPageComponentStatus.Degraded]: 'statusPages.components.status.degraded',
  [StatusPageComponentStatus.PartialOutage]: 'statusPages.components.status.partialOutage',
  [StatusPageComponentStatus.MajorOutage]: 'statusPages.components.status.majorOutage',
}

function componentStatusBadgeVariant(
  status: StatusPageComponentStatus,
): 'success' | 'warning' | 'error' | 'destructive' {
  switch (status) {
    case StatusPageComponentStatus.MajorOutage:
      return 'destructive'
    case StatusPageComponentStatus.PartialOutage:
      return 'error'
    case StatusPageComponentStatus.Degraded:
      return 'warning'
    default:
      return 'success'
  }
}

function componentStatusIcon(status: StatusPageComponentStatus): ReactElement {
  switch (status) {
    case StatusPageComponentStatus.MajorOutage:
      return <CircleXIcon />
    case StatusPageComponentStatus.PartialOutage:
      return <CircleAlertIcon />
    case StatusPageComponentStatus.Degraded:
      return <AlertTriangleIcon />
    default:
      return <CheckCircle2Icon />
  }
}

function nextPosition(components: ComponentRow[]): number {
  if (components.length === 0) {
    return 0
  }
  return Math.max(...components.map((component) => component.position)) + 1
}

export function StatusPageComponentsPanel(): ReactElement {
  const [{ data, fetching, error }, reexecuteQuery] = useStatusPageQuery({
    requestPolicy: 'network-only',
  })
  const [, createStatusPageComponent] = useCreateStatusPageComponentMutation()
  const [, updateStatusPageComponent] = useUpdateStatusPageComponentMutation()
  const [, deleteStatusPageComponent] = useDeleteStatusPageComponentMutation()
  const [{ data: servicesData, fetching: servicesFetching }] = useServicesQuery({
    requestPolicy: 'cache-first',
  })

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<ComponentRow | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [status, setStatus] = useState<StatusPageComponentStatus>(
    StatusPageComponentStatus.Operational,
  )
  const [serviceId, setServiceId] = useState<string | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [actionComponentId, setActionComponentId] = useState<string | null>(null)

  const statusPage = data?.statusPage
  const services = useMemo(() => servicesData?.services ?? [], [servicesData?.services])
  const serviceNameById = useMemo(
    () => new Map(services.map((service) => [service.id, service.name])),
    [services],
  )

  const components = useMemo(() => {
    const rows = statusPage?.components ?? []
    return [...rows].sort((left, right) => left.position - right.position)
  }, [statusPage?.components])

  useEffect(() => {
    if (!formOpen) {
      setEditing(null)
      setName('')
      setDescription('')
      setStatus(StatusPageComponentStatus.Operational)
      setServiceId(null)
      setFormError(null)
    }
  }, [formOpen])

  function refresh(): void {
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  function openCreate(): void {
    setEditing(null)
    setName('')
    setDescription('')
    setStatus(StatusPageComponentStatus.Operational)
    setServiceId(null)
    setFormError(null)
    setFormOpen(true)
  }

  function openEdit(component: ComponentRow): void {
    setEditing(component)
    setName(component.name)
    setDescription(component.description ?? '')
    setStatus(component.status)
    setServiceId(component.serviceId ?? null)
    setFormError(null)
    setFormOpen(true)
  }

  async function handleSave(): Promise<void> {
    const trimmedName = name.trim()
    if (!trimmedName) {
      setFormError(t('statusPages.components.validation.requiredName'))
      return
    }

    setSaving(true)
    setFormError(null)

    const trimmedDescription = description.trim()
    const result = editing
      ? await updateStatusPageComponent({
          input: {
            id: editing.id,
            name: trimmedName,
            description: trimmedDescription || null,
            status,
            position: editing.position,
            serviceId,
          },
        })
      : await createStatusPageComponent({
          input: {
            name: trimmedName,
            description: trimmedDescription || null,
            status,
            position: nextPosition(components),
            serviceId,
          },
        })

    setSaving(false)

    if (result.error) {
      setFormError(formatGraphQLError(result.error.message))
      return
    }

    setFormOpen(false)
    refresh()
  }

  async function handleDelete(component: ComponentRow): Promise<void> {
    setActionError(null)
    setActionComponentId(component.id)

    const result = await deleteStatusPageComponent({ id: component.id })
    setActionComponentId(null)

    if (result.error) {
      setActionError(formatGraphQLError(result.error.message))
      return
    }

    refresh()
  }

  async function handleMove(component: ComponentRow, direction: -1 | 1): Promise<void> {
    const index = components.findIndex((item) => item.id === component.id)
    const targetIndex = index + direction
    if (index === -1 || targetIndex < 0 || targetIndex >= components.length) {
      return
    }

    const neighbor = components[targetIndex]
    setActionError(null)
    setActionComponentId(component.id)

    const firstResult = await updateStatusPageComponent({
      input: {
        id: component.id,
        name: component.name,
        description: component.description,
        status: component.status,
        position: neighbor.position,
        serviceId: component.serviceId ?? null,
      },
    })

    if (firstResult.error) {
      setActionComponentId(null)
      setActionError(formatGraphQLError(firstResult.error.message))
      return
    }

    const secondResult = await updateStatusPageComponent({
      input: {
        id: neighbor.id,
        name: neighbor.name,
        description: neighbor.description,
        status: neighbor.status,
        position: component.position,
        serviceId: neighbor.serviceId ?? null,
      },
    })

    setActionComponentId(null)

    if (secondResult.error) {
      setActionError(formatGraphQLError(secondResult.error.message))
      refresh()
      return
    }

    refresh()
  }

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex items-start gap-3">
          <LayersIcon className="mt-0.5 size-5 text-muted-foreground" />
          <div>
            <h2 className="text-xl font-semibold text-foreground">{t('statusPages.components.title')}</h2>
            <p className="mt-2 text-sm text-muted-foreground">{t('statusPages.components.description')}</p>
          </div>
        </div>

        <Dialog onOpenChange={setFormOpen} open={formOpen}>
          <DialogTrigger
            disabled={!statusPage}
            render={<Button onClick={openCreate} type="button" />}
          >
            <PlusIcon />
            {t('statusPages.components.action.create')}
          </DialogTrigger>
          <DialogPopup>
            <DialogHeader>
              <DialogTitle>
                {editing
                  ? t('statusPages.components.edit.title')
                  : t('statusPages.components.create.title')}
              </DialogTitle>
              <DialogDescription>
                {editing
                  ? t('statusPages.components.edit.description')
                  : t('statusPages.components.create.description')}
              </DialogDescription>
            </DialogHeader>
            <DialogPanel className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground" htmlFor="status-page-component-name">
                  {t('statusPages.components.field.name')}
                </label>
                <Input
                  autoComplete="off"
                  id="status-page-component-name"
                  onChange={(event) => setName(event.target.value)}
                  placeholder={t('statusPages.components.field.namePlaceholder')}
                  value={name}
                />
              </div>

              <div className="space-y-2">
                <label
                  className="text-sm font-medium text-foreground"
                  htmlFor="status-page-component-description"
                >
                  {t('statusPages.components.field.description')}
                </label>
                <Textarea
                  id="status-page-component-description"
                  onChange={(event) => setDescription(event.target.value)}
                  placeholder={t('statusPages.components.field.descriptionPlaceholder')}
                  rows={3}
                  value={description}
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground" htmlFor="status-page-component-status">
                  {t('statusPages.components.field.status')}
                </label>
                <Select
                  onValueChange={(value) =>
                    setStatus((value as StatusPageComponentStatus) ?? StatusPageComponentStatus.Operational)
                  }
                  value={status}
                >
                  <SelectTrigger id="status-page-component-status">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectPopup>
                    {COMPONENT_STATUS_OPTIONS.map((option) => (
                      <SelectItem key={option} value={option}>
                        {t(COMPONENT_STATUS_LABEL_KEYS[option])}
                      </SelectItem>
                    ))}
                  </SelectPopup>
                </Select>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground" htmlFor="status-page-component-service">
                  {t('statusPages.components.field.service')}
                </label>
                <Combobox onValueChange={setServiceId} value={serviceId}>
                  <ComboboxInput
                    id="status-page-component-service"
                    placeholder={t('statusPages.components.field.servicePlaceholder')}
                    showClear
                  />
                  <ComboboxPopup>
                    <ComboboxList>
                      <ComboboxEmpty>
                        {servicesFetching
                          ? t('statusPages.components.field.serviceLoading')
                          : t('statusPages.components.field.serviceEmpty')}
                      </ComboboxEmpty>
                      {services.map((service) => (
                        <ComboboxItem key={service.id} value={service.id}>
                          {service.name}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxPopup>
                </Combobox>
                <p className="text-xs text-muted-foreground">
                  {t('statusPages.components.field.serviceHelp')}
                </p>
              </div>

              {formError ? (
                <Alert variant="error">
                  <AlertDescription>{formError}</AlertDescription>
                </Alert>
              ) : null}
            </DialogPanel>
            <DialogFooter>
              <DialogClose render={<Button type="button" variant="ghost" />}>
                {t('statusPages.components.action.cancel')}
              </DialogClose>
              <Button disabled={saving} onClick={() => void handleSave()} type="button">
                {saving
                  ? t('statusPages.components.action.saving')
                  : t('statusPages.components.action.save')}
              </Button>
            </DialogFooter>
          </DialogPopup>
        </Dialog>
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
          <p className="text-sm text-muted-foreground">{t('statusPages.components.loading')}</p>
        ) : !statusPage ? (
          <p className="text-sm text-muted-foreground">{t('statusPages.components.notConfigured')}</p>
        ) : components.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('statusPages.components.empty')}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('statusPages.components.column.name')}</TableHead>
                <TableHead>{t('statusPages.components.column.status')}</TableHead>
                <TableHead>{t('statusPages.components.column.service')}</TableHead>
                <TableHead className="text-right">{t('statusPages.components.column.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {components.map((component, index) => (
                <TableRow key={component.id}>
                  <TableCell>
                    <div className="min-w-0">
                      <p className="font-medium text-foreground">{component.name}</p>
                      {component.description ? (
                        <p className="mt-1 truncate text-sm text-muted-foreground">
                          {component.description}
                        </p>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge className="gap-1" variant={componentStatusBadgeVariant(component.status)}>
                      {componentStatusIcon(component.status)}
                      {t(COMPONENT_STATUS_LABEL_KEYS[component.status])}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {component.serviceId
                      ? (serviceNameById.get(component.serviceId) ?? component.serviceId)
                      : t('statusPages.components.service.unlinked')}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        aria-label={t('statusPages.components.action.moveUp')}
                        disabled={actionComponentId === component.id || index === 0}
                        onClick={() => void handleMove(component, -1)}
                        size="icon-sm"
                        type="button"
                        variant="ghost"
                      >
                        <ArrowUpIcon />
                      </Button>
                      <Button
                        aria-label={t('statusPages.components.action.moveDown')}
                        disabled={
                          actionComponentId === component.id || index === components.length - 1
                        }
                        onClick={() => void handleMove(component, 1)}
                        size="icon-sm"
                        type="button"
                        variant="ghost"
                      >
                        <ArrowDownIcon />
                      </Button>
                      <Button
                        disabled={actionComponentId === component.id}
                        onClick={() => openEdit(component)}
                        size="sm"
                        type="button"
                        variant="outline"
                      >
                        {t('statusPages.components.action.edit')}
                      </Button>
                      <AlertDialog>
                        <AlertDialogTrigger
                          render={
                            <Button
                              disabled={actionComponentId === component.id}
                              size="sm"
                              type="button"
                              variant="destructive-outline"
                            />
                          }
                        >
                          {t('statusPages.components.action.delete')}
                        </AlertDialogTrigger>
                        <AlertDialogPopup>
                          <AlertDialogHeader>
                            <AlertDialogTitle>{t('statusPages.components.delete.title')}</AlertDialogTitle>
                            <AlertDialogDescription>
                              {t('statusPages.components.delete.description', { name: component.name })}
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                              {t('statusPages.components.action.cancel')}
                            </AlertDialogClose>
                            <AlertDialogClose
                              onClick={() => void handleDelete(component)}
                              render={<Button type="button" variant="destructive" />}
                            >
                              {t('statusPages.components.delete.confirm')}
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
        )}
      </div>
    </section>
  )
}
