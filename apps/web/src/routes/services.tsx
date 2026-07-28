import type { ReactElement } from 'react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router'
import {
  useCreateServiceMutation,
  useDeleteServiceMutation,
  useServicesQuery,
  useTeamsQuery,
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
} from '@escalite/ui'
import { AlertTriangleIcon, PlusIcon, SearchIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

export function ServicesPage(): ReactElement {
  const [{ data, fetching, error }, reexecuteQuery] = useServicesQuery({
    requestPolicy: 'network-only',
  })
  const [{ data: teamsData }] = useTeamsQuery()
  const [, createService] = useCreateServiceMutation()
  const [, deleteService] = useDeleteServiceMutation()

  const [search, setSearch] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newTeamId, setNewTeamId] = useState('')
  const [actionError, setActionError] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const teams = teamsData?.teams ?? []
  const services = useMemo(() => {
    const items = data?.services ?? []
    const query = search.trim().toLowerCase()
    if (!query) {
      return items
    }
    return items.filter((service) => service.name.toLowerCase().includes(query))
  }, [data?.services, search])

  async function handleCreate(): Promise<void> {
    const name = newName.trim()
    const teamId = newTeamId.trim()
    if (!name || !teamId) {
      setActionError(t('services.error.requiredFields'))
      return
    }

    setActionError(null)
    setCreating(true)
    const result = await createService({ input: { name, teamId } })
    setCreating(false)

    if (result.error) {
      showMutationError(result.error, 'services.error.action')
      return
    }

    setCreateOpen(false)
    setNewName('')
    setNewTeamId('')
    notifyMutationSuccess('services.toast.created')
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  async function handleDelete(serviceId: string): Promise<void> {
    setActionError(null)
    setDeletingId(serviceId)
    const result = await deleteService({ id: serviceId })
    setDeletingId(null)

    if (result.error) {
      showMutationError(result.error, 'services.error.action')
      return
    }

    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  return (
    <AppShell title={t('services.title')}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold text-foreground">{t('services.title')}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{t('services.description')}</p>
          </div>
          <Dialog onOpenChange={setCreateOpen} open={createOpen}>
            <DialogTrigger render={<Button type="button" />}>
              <PlusIcon />
              {t('services.action.create')}
            </DialogTrigger>
            <DialogPopup>
              <DialogHeader>
                <DialogTitle>{t('services.create.title')}</DialogTitle>
                <DialogDescription>{t('services.create.description')}</DialogDescription>
              </DialogHeader>
              <DialogPanel className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="service-name">
                    {t('services.field.name')}
                  </label>
                  <Input
                    id="service-name"
                    onChange={(event) => setNewName(event.target.value)}
                    placeholder={t('services.field.namePlaceholder')}
                    value={newName}
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="service-team">
                    {t('services.field.team')}
                  </label>
                  <Select onValueChange={(value) => setNewTeamId(value ?? '')} value={newTeamId}>
                    <SelectTrigger id="service-team">
                      <SelectValue placeholder={t('services.field.teamPlaceholder')} />
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
              </DialogPanel>
              <DialogFooter>
                <DialogClose render={<Button type="button" variant="ghost" />}>
                  {t('services.action.cancel')}
                </DialogClose>
                <Button disabled={creating} onClick={() => void handleCreate()} type="button">
                  {creating ? t('services.action.creating') : t('services.action.create')}
                </Button>
              </DialogFooter>
            </DialogPopup>
          </Dialog>
        </div>

        <div className="relative mt-6 max-w-md">
          <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            aria-label={t('services.search.label')}
            className="pl-9"
            onChange={(event) => setSearch(event.target.value)}
            placeholder={t('services.search.placeholder')}
            value={search}
          />
        </div>

        {error ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('services.error.load')}</AlertTitle>
            <AlertDescription>{formatGraphQLError(error.message)}</AlertDescription>
          </Alert>
        ) : null}

        {actionError ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('services.error.action')}</AlertTitle>
            <AlertDescription>{actionError}</AlertDescription>
          </Alert>
        ) : null}

        <Table className="mt-6" variant="card">
          <TableHeader>
            <TableRow>
              <TableHead>{t('services.column.name')}</TableHead>
              <TableHead>{t('services.column.team')}</TableHead>
              <TableHead>{t('services.column.updated')}</TableHead>
              <TableHead className="text-right">{t('services.column.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {fetching ? (
              <TableRow>
                <TableCell className="text-muted-foreground" colSpan={4}>
                  {t('services.loading')}
                </TableCell>
              </TableRow>
            ) : services.length === 0 ? (
              <TableRow>
                <TableCell className="text-muted-foreground" colSpan={4}>
                  {search.trim() ? t('services.empty.search') : t('services.empty.default')}
                </TableCell>
              </TableRow>
            ) : (
              services.map((service) => {
                const teamName =
                  teams.find((team) => team.id === service.teamId)?.name ?? service.teamId

                return (
                  <TableRow key={service.id}>
                    <TableCell>
                      <Link
                        className="font-medium text-foreground hover:underline"
                        to={`/services/${service.id}`}
                      >
                        {service.name}
                      </Link>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{teamName}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {new Date(service.updatedAt).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button
                          render={<Link to={`/services/${service.id}`} />}
                          size="sm"
                          type="button"
                          variant="outline"
                        >
                          {t('services.action.configure')}
                        </Button>
                        <AlertDialog>
                          <AlertDialogTrigger
                            render={
                              <Button
                                disabled={deletingId === service.id}
                                size="sm"
                                type="button"
                                variant="destructive-outline"
                              />
                            }
                          >
                            {t('services.action.delete')}
                          </AlertDialogTrigger>
                          <AlertDialogPopup>
                            <AlertDialogHeader>
                              <AlertDialogTitle>{t('services.delete.title')}</AlertDialogTitle>
                              <AlertDialogDescription>
                                {t('services.delete.description', { name: service.name })}
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                                {t('services.action.cancel')}
                              </AlertDialogClose>
                              <AlertDialogClose
                                onClick={() => void handleDelete(service.id)}
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
                )
              })
            )}
          </TableBody>
        </Table>
      </section>
    </AppShell>
  )
}
