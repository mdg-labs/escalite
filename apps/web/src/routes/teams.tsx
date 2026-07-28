import type { ReactElement } from 'react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router'
import { useCreateTeamMutation, useTeamsQuery } from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
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
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  Input,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@escalite/ui'
import { AlertTriangleIcon, PlusIcon, SearchIcon, UsersIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

function TeamTableSkeletonRows(): ReactElement {
  return (
    <>
      {Array.from({ length: 5 }).map((_, index) => (
        <TableRow key={`team-skeleton-${index}`}>
          <TableCell>
            <Skeleton className="h-4 max-w-48" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-36" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-36" />
          </TableCell>
        </TableRow>
      ))}
    </>
  )
}

export function TeamsPage(): ReactElement {
  const [{ data, fetching, error }, reexecuteQuery] = useTeamsQuery({
    requestPolicy: 'network-only',
  })
  const [, createTeam] = useCreateTeamMutation()

  const [search, setSearch] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [actionError, setActionError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const teams = useMemo(() => {
    const items = data?.teams ?? []
    const query = search.trim().toLowerCase()
    if (!query) {
      return items
    }
    return items.filter((team) => team.name.toLowerCase().includes(query))
  }, [data?.teams, search])

  const allTeams = data?.teams ?? []
  const showEmptyState = !fetching && !error && allTeams.length === 0

  async function handleCreate(): Promise<void> {
    const name = newName.trim()
    if (!name) {
      setActionError(t('teams.error.requiredName'))
      return
    }

    setActionError(null)
    setCreating(true)
    const result = await createTeam({ input: { name } })
    setCreating(false)

    if (result.error) {
      showMutationError(result.error, 'teams.error.action')
      return
    }

    setCreateOpen(false)
    setNewName('')
    notifyMutationSuccess('teams.toast.created')
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  const createDialog = (
    <Dialog onOpenChange={setCreateOpen} open={createOpen}>
      <DialogTrigger render={<Button type="button" />}>
        <PlusIcon />
        {t('teams.action.create')}
      </DialogTrigger>
      <DialogPopup>
        <DialogHeader>
          <DialogTitle>{t('teams.create.title')}</DialogTitle>
          <DialogDescription>{t('teams.create.description')}</DialogDescription>
        </DialogHeader>
        <DialogPanel className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="team-name">
              {t('teams.field.name')}
            </label>
            <Input
              id="team-name"
              onChange={(event) => setNewName(event.target.value)}
              placeholder={t('teams.field.namePlaceholder')}
              value={newName}
            />
          </div>
        </DialogPanel>
        <DialogFooter>
          <DialogClose render={<Button type="button" variant="ghost" />}>
            {t('teams.action.cancel')}
          </DialogClose>
          <Button disabled={creating} onClick={() => void handleCreate()} type="button">
            {creating ? t('teams.action.creating') : t('teams.action.create')}
          </Button>
        </DialogFooter>
      </DialogPopup>
    </Dialog>
  )

  return (
    <AppShell title={t('teams.title')}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold text-foreground">{t('teams.title')}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{t('teams.description')}</p>
          </div>
          {!showEmptyState ? createDialog : null}
        </div>

        {error ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('teams.error.load')}</AlertTitle>
            <AlertDescription>{formatGraphQLError(error.message)}</AlertDescription>
          </Alert>
        ) : null}

        {actionError ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('teams.error.action')}</AlertTitle>
            <AlertDescription>{actionError}</AlertDescription>
          </Alert>
        ) : null}

        {showEmptyState ? (
          <Empty className="mt-10">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <UsersIcon />
              </EmptyMedia>
              <EmptyTitle>{t('teams.empty.title')}</EmptyTitle>
              <EmptyDescription>{t('teams.empty.description')}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>{createDialog}</EmptyContent>
          </Empty>
        ) : (
          <>
            <div className="relative mt-6 max-w-md">
              <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                aria-label={t('teams.search.label')}
                className="pl-9"
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t('teams.search.placeholder')}
                value={search}
              />
            </div>

            <Table className="mt-6" variant="card">
              <TableHeader>
                <TableRow>
                  <TableHead>{t('teams.column.name')}</TableHead>
                  <TableHead>{t('teams.column.created')}</TableHead>
                  <TableHead>{t('teams.column.updated')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {fetching ? (
                  <TeamTableSkeletonRows />
                ) : teams.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={3}>
                      <Empty>
                        <EmptyHeader>
                          <EmptyMedia variant="icon">
                            <UsersIcon />
                          </EmptyMedia>
                          <EmptyTitle>{t('teams.empty.search')}</EmptyTitle>
                        </EmptyHeader>
                      </Empty>
                    </TableCell>
                  </TableRow>
                ) : (
                  teams.map((team) => (
                    <TableRow key={team.id}>
                      <TableCell className="font-medium text-foreground">
                        <Link className="text-primary hover:underline" to={`/teams/${team.id}`}>
                          {team.name}
                        </Link>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {new Date(team.createdAt).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {new Date(team.updatedAt).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </>
        )}
      </section>
    </AppShell>
  )
}
