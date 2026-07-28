import { useEffect, useMemo, useState, type ReactElement } from 'react'
import {
  useCreateIncidentRoleDefinitionMutation,
  useDeleteIncidentRoleDefinitionMutation,
  useIncidentRoleDefinitionsQuery,
  useUpdateIncidentRoleDefinitionMutation,
  type IncidentRoleDefinitionsQuery,
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
import { ArrowDownIcon, ArrowUpIcon, PlusIcon, UserCogIcon } from 'lucide-react'

import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

type RoleDefinitionRow = IncidentRoleDefinitionsQuery['incidentRoleDefinitions'][number]

function nextSortOrder(roles: RoleDefinitionRow[]): number {
  if (roles.length === 0) {
    return 0
  }
  return Math.max(...roles.map((role) => role.sortOrder)) + 1
}

export function IncidentRolesPanel(): ReactElement {
  const [{ data, fetching, error }, reexecuteQuery] = useIncidentRoleDefinitionsQuery({
    requestPolicy: 'network-only',
  })
  const [, createRoleDefinition] = useCreateIncidentRoleDefinitionMutation()
  const [, updateRoleDefinition] = useUpdateIncidentRoleDefinitionMutation()
  const [, deleteRoleDefinition] = useDeleteIncidentRoleDefinitionMutation()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<RoleDefinitionRow | null>(null)
  const [name, setName] = useState('')
  const [formError, setFormError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [actionRoleId, setActionRoleId] = useState<string | null>(null)

  const roles = useMemo(() => {
    const definitions = data?.incidentRoleDefinitions ?? []
    return [...definitions].sort((left, right) => left.sortOrder - right.sortOrder)
  }, [data?.incidentRoleDefinitions])

  useEffect(() => {
    if (!formOpen) {
      setEditing(null)
      setName('')
      setFormError(null)
    }
  }, [formOpen])

  function openCreate(): void {
    setEditing(null)
    setName('')
    setFormError(null)
    setFormOpen(true)
  }

  function openEdit(role: RoleDefinitionRow): void {
    setEditing(role)
    setName(role.name)
    setFormError(null)
    setFormOpen(true)
  }

  function refresh(): void {
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  async function handleSave(): Promise<void> {
    const trimmedName = name.trim()
    if (!trimmedName) {
      setFormError(t('settings.incidentRoles.validation.required'))
      return
    }

    setSaving(true)
    setFormError(null)

    const result = editing
      ? await updateRoleDefinition({
          input: {
            id: editing.id,
            name: trimmedName,
            sortOrder: editing.sortOrder,
          },
        })
      : await createRoleDefinition({
          input: {
            name: trimmedName,
            sortOrder: nextSortOrder(roles),
          },
        })

    setSaving(false)

    if (result.error) {
      showMutationError(result.error, 'incidentRoles.error.action')
      return
    }

    notifyMutationSuccess(
      editing ? 'incidentRoles.toast.updated' : 'incidentRoles.toast.created',
    )
    setFormOpen(false)
    refresh()
  }

  async function handleDelete(role: RoleDefinitionRow): Promise<void> {
    setActionError(null)
    setActionRoleId(role.id)

    const result = await deleteRoleDefinition({ id: role.id })
    setActionRoleId(null)

    if (result.error) {
      showMutationError(result.error, 'incidentRoles.error.action')
      return
    }

    refresh()
  }

  async function handleMove(role: RoleDefinitionRow, direction: -1 | 1): Promise<void> {
    const index = roles.findIndex((item) => item.id === role.id)
    const targetIndex = index + direction
    if (index === -1 || targetIndex < 0 || targetIndex >= roles.length) {
      return
    }

    const neighbor = roles[targetIndex]
    setActionError(null)
    setActionRoleId(role.id)

    const firstResult = await updateRoleDefinition({
      input: {
        id: role.id,
        name: role.name,
        sortOrder: neighbor.sortOrder,
      },
    })

    if (firstResult.error) {
      setActionRoleId(null)
      showMutationError(firstResult.error, 'incidentRoles.error.action')
      return
    }

    const secondResult = await updateRoleDefinition({
      input: {
        id: neighbor.id,
        name: neighbor.name,
        sortOrder: role.sortOrder,
      },
    })

    setActionRoleId(null)

    if (secondResult.error) {
      showMutationError(secondResult.error, 'incidentRoles.error.action')
      refresh()
      return
    }

    refresh()
  }

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex items-start gap-3">
          <UserCogIcon className="mt-0.5 size-5 text-muted-foreground" />
          <div>
            <h2 className="text-xl font-semibold text-foreground">{t('settings.incidentRoles.title')}</h2>
            <p className="mt-2 text-sm text-muted-foreground">{t('settings.incidentRoles.description')}</p>
          </div>
        </div>

        <Dialog onOpenChange={setFormOpen} open={formOpen}>
          <DialogTrigger render={<Button onClick={openCreate} type="button" />}>
            <PlusIcon />
            {t('settings.incidentRoles.action.create')}
          </DialogTrigger>
          <DialogPopup>
            <DialogHeader>
              <DialogTitle>
                {editing ? t('settings.incidentRoles.edit.title') : t('settings.incidentRoles.create.title')}
              </DialogTitle>
              <DialogDescription>
                {editing
                  ? t('settings.incidentRoles.edit.description')
                  : t('settings.incidentRoles.create.description')}
              </DialogDescription>
            </DialogHeader>
            <DialogPanel className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground" htmlFor="incident-role-name">
                  {t('settings.incidentRoles.field.name')}
                </label>
                <Input
                  autoComplete="off"
                  id="incident-role-name"
                  onChange={(event) => {
                    setName(event.target.value)
                  }}
                  placeholder={t('settings.incidentRoles.field.namePlaceholder')}
                  value={name}
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
                {t('settings.incidentRoles.action.cancel')}
              </DialogClose>
              <Button disabled={saving} onClick={() => void handleSave()} type="button">
                {saving ? t('settings.incidentRoles.action.saving') : t('settings.incidentRoles.action.save')}
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
        {fetching && roles.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('settings.incidentRoles.loading')}</p>
        ) : roles.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('settings.incidentRoles.empty')}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('settings.incidentRoles.column.name')}</TableHead>
                <TableHead className="text-right">{t('settings.incidentRoles.column.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {roles.map((role, index) => (
                <TableRow key={role.id}>
                  <TableCell className="font-medium text-foreground">{role.name}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        aria-label={t('settings.incidentRoles.action.moveUp')}
                        disabled={actionRoleId === role.id || index === 0}
                        onClick={() => void handleMove(role, -1)}
                        size="icon-sm"
                        type="button"
                        variant="ghost"
                      >
                        <ArrowUpIcon />
                      </Button>
                      <Button
                        aria-label={t('settings.incidentRoles.action.moveDown')}
                        disabled={actionRoleId === role.id || index === roles.length - 1}
                        onClick={() => void handleMove(role, 1)}
                        size="icon-sm"
                        type="button"
                        variant="ghost"
                      >
                        <ArrowDownIcon />
                      </Button>
                      <Button
                        disabled={actionRoleId === role.id}
                        onClick={() => openEdit(role)}
                        size="sm"
                        type="button"
                        variant="outline"
                      >
                        {t('settings.incidentRoles.action.edit')}
                      </Button>
                      <AlertDialog>
                        <AlertDialogTrigger
                          render={
                            <Button
                              disabled={actionRoleId === role.id}
                              size="sm"
                              type="button"
                              variant="destructive-outline"
                            />
                          }
                        >
                          {t('settings.incidentRoles.action.delete')}
                        </AlertDialogTrigger>
                        <AlertDialogPopup>
                          <AlertDialogHeader>
                            <AlertDialogTitle>{t('settings.incidentRoles.delete.title')}</AlertDialogTitle>
                            <AlertDialogDescription>
                              {t('settings.incidentRoles.delete.description', { name: role.name })}
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                              {t('settings.incidentRoles.action.cancel')}
                            </AlertDialogClose>
                            <AlertDialogClose
                              onClick={() => void handleDelete(role)}
                              render={<Button type="button" variant="destructive" />}
                            >
                              {t('settings.incidentRoles.delete.confirm')}
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
