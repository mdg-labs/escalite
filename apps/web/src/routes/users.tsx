import type { ReactElement } from 'react'
import { useMemo, useState } from 'react'
import { Navigate } from 'react-router'
import {
  UserRole,
  useAddTeamMemberMutation,
  useInviteUserMutation,
  useMeQuery,
  useOrganizationUsersQuery,
  useTeamsQuery,
  useUpdateUserRoleMutation,
} from '@escalite/ts-types'
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
  Input,
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
  selectOptionsFromEntities,
  selectOptionsWithSentinel,
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

const NO_TEAM_VALUE = '__none__'

function roleLabel(role: UserRole): string {
  return role === UserRole.Admin ? t('org.switcher.role.admin') : t('org.switcher.role.member')
}

export function UsersPage(): ReactElement {
  const [{ data: meData, fetching: meFetching }] = useMeQuery({ requestPolicy: 'cache-first' })
  const isAdmin = meData?.me?.role === UserRole.Admin
  const currentUserId = meData?.me?.id

  const [{ data, fetching, error }, reexecuteQuery] = useOrganizationUsersQuery({
    pause: !isAdmin,
    requestPolicy: 'network-only',
  })
  const [{ data: teamsData }] = useTeamsQuery({
    pause: !isAdmin,
    requestPolicy: 'cache-first',
  })

  const [, inviteUser] = useInviteUserMutation()
  const [, addTeamMember] = useAddTeamMemberMutation()
  const [, updateUserRole] = useUpdateUserRoleMutation()

  const [search, setSearch] = useState('')
  const [inviteOpen, setInviteOpen] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState<UserRole>(UserRole.Member)
  const [inviteTeamId, setInviteTeamId] = useState(NO_TEAM_VALUE)
  const [actionError, setActionError] = useState<string | null>(null)
  const [inviting, setInviting] = useState(false)
  const [updatingUserId, setUpdatingUserId] = useState<string | null>(null)

  const teamNameById = useMemo(() => {
    const map = new Map<string, string>()
    for (const team of teamsData?.teams ?? []) {
      map.set(team.id, team.name)
    }
    return map
  }, [teamsData?.teams])

  const inviteRoleItems = useMemo(
    () => [
      { label: t('org.switcher.role.admin'), value: UserRole.Admin },
      { label: t('org.switcher.role.member'), value: UserRole.Member },
    ],
    [],
  )

  const inviteTeamItems = useMemo(
    () =>
      selectOptionsWithSentinel(
        t('users.field.teamNone'),
        NO_TEAM_VALUE,
        selectOptionsFromEntities(teamsData?.teams ?? []),
      ),
    [teamsData?.teams],
  )

  const roleItems = inviteRoleItems

  const users = useMemo(() => {
    const items = data?.organizationUsers ?? []
    const query = search.trim().toLowerCase()
    const filtered = query
      ? items.filter((user) => user.email.toLowerCase().includes(query))
      : items
    return [...filtered].sort((left, right) => left.email.localeCompare(right.email))
  }, [data?.organizationUsers, search])

  const allUsers = data?.organizationUsers ?? []
  const showEmptyState = !fetching && !error && allUsers.length === 0

  if (!meFetching && !isAdmin) {
    return <Navigate replace to="/dashboard" />
  }

  function resetInviteForm(): void {
    setInviteEmail('')
    setInviteRole(UserRole.Member)
    setInviteTeamId(NO_TEAM_VALUE)
    setActionError(null)
  }

  async function handleInvite(): Promise<void> {
    const email = inviteEmail.trim()
    if (!email) {
      setActionError(t('users.error.requiredEmail'))
      return
    }

    setActionError(null)
    setInviting(true)

    const inviteResult = await inviteUser({
      input: { email, role: inviteRole },
    })

    if (inviteResult.error) {
      setInviting(false)
      showMutationError(inviteResult.error, 'users.error.action')
      return
    }

    const invitedUserId = inviteResult.data?.inviteUser.id
    if (inviteTeamId !== NO_TEAM_VALUE && invitedUserId) {
      const addResult = await addTeamMember({ teamId: inviteTeamId, userId: invitedUserId })
      if (addResult.error) {
        setInviting(false)
        showMutationError(addResult.error, 'users.error.action')
        reexecuteQuery({ requestPolicy: 'network-only' })
        return
      }
    }

    setInviting(false)
    setInviteOpen(false)
    resetInviteForm()
    notifyMutationSuccess('users.toast.invited')
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  async function handleRoleChange(userId: string, role: UserRole | null): Promise<void> {
    if (!role) {
      return
    }

    setActionError(null)
    setUpdatingUserId(userId)
    const result = await updateUserRole({ input: { userId, role } })
    setUpdatingUserId(null)

    if (result.error) {
      showMutationError(result.error, 'users.error.action')
      return
    }

    notifyMutationSuccess('users.toast.roleUpdated')
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  function formatTeamNames(teamIds: string[]): string {
    if (teamIds.length === 0) {
      return t('users.teams.none')
    }

    return teamIds
      .map((teamId) => teamNameById.get(teamId) ?? teamId)
      .sort((left, right) => left.localeCompare(right))
      .join(', ')
  }

  const inviteDialog = (
    <Dialog
      onOpenChange={(open) => {
        setInviteOpen(open)
        if (!open) {
          resetInviteForm()
        }
      }}
      open={inviteOpen}
    >
      <DialogTrigger render={<Button type="button" />}>
        <PlusIcon />
        {t('users.action.invite')}
      </DialogTrigger>
      <DialogPopup>
        <DialogHeader>
          <DialogTitle>{t('users.invite.title')}</DialogTitle>
          <DialogDescription>{t('users.invite.description')}</DialogDescription>
        </DialogHeader>
        <DialogPanel className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="invite-email">
              {t('users.field.email')}
            </label>
            <Input
              autoComplete="email"
              id="invite-email"
              onChange={(event) => setInviteEmail(event.target.value)}
              placeholder={t('users.field.emailPlaceholder')}
              type="email"
              value={inviteEmail}
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="invite-role">
              {t('users.field.role')}
            </label>
            <Select
              items={inviteRoleItems}
              onValueChange={(value) => setInviteRole((value as UserRole) ?? UserRole.Member)}
              value={inviteRole}
            >
              <SelectTrigger id="invite-role">
                <SelectValue />
              </SelectTrigger>
              <SelectPopup>
                {inviteRoleItems.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectPopup>
            </Select>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="invite-team">
              {t('users.field.team')}
            </label>
            <Select
              items={inviteTeamItems}
              onValueChange={(value) => setInviteTeamId(value ?? NO_TEAM_VALUE)}
              value={inviteTeamId}
            >
              <SelectTrigger id="invite-team">
                <SelectValue placeholder={t('users.field.teamPlaceholder')} />
              </SelectTrigger>
              <SelectPopup>
                {inviteTeamItems.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectPopup>
            </Select>
          </div>
        </DialogPanel>
        <DialogFooter>
          <DialogClose render={<Button type="button" variant="ghost" />}>
            {t('users.action.cancel')}
          </DialogClose>
          <Button disabled={inviting} onClick={() => void handleInvite()} type="button">
            {inviting ? t('users.action.inviting') : t('users.action.invite')}
          </Button>
        </DialogFooter>
      </DialogPopup>
    </Dialog>
  )

  return (
    <AppShell title={t('users.title')}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold text-foreground">{t('users.title')}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{t('users.description')}</p>
          </div>
          {!showEmptyState ? inviteDialog : null}
        </div>

        {error ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('users.error.load')}</AlertTitle>
            <AlertDescription>{formatGraphQLError(error.message)}</AlertDescription>
          </Alert>
        ) : null}

        {actionError ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('users.error.action')}</AlertTitle>
            <AlertDescription>{actionError}</AlertDescription>
          </Alert>
        ) : null}

        {showEmptyState ? (
          <div className="mt-10 flex flex-col items-center justify-center rounded-lg border border-dashed border-border px-6 py-16 text-center">
            <div className="flex size-12 items-center justify-center rounded-full bg-muted">
              <UsersIcon className="size-6 text-muted-foreground" />
            </div>
            <h2 className="mt-4 text-lg font-medium text-foreground">{t('users.empty.title')}</h2>
            <p className="mt-2 max-w-sm text-sm text-muted-foreground">
              {t('users.empty.description')}
            </p>
            <div className="mt-6">{inviteDialog}</div>
          </div>
        ) : (
          <>
            <div className="relative mt-6 max-w-md">
              <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                aria-label={t('users.search.label')}
                className="pl-9"
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t('users.search.placeholder')}
                value={search}
              />
            </div>

            <Table className="mt-6" variant="card">
              <TableHeader>
                <TableRow>
                  <TableHead>{t('users.column.email')}</TableHead>
                  <TableHead>{t('users.column.role')}</TableHead>
                  <TableHead>{t('users.column.teams')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {fetching ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={3}>
                      {t('users.loading')}
                    </TableCell>
                  </TableRow>
                ) : users.length === 0 ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={3}>
                      {t('users.empty.search')}
                    </TableCell>
                  </TableRow>
                ) : (
                  users.map((user) => {
                    const isCurrentUser = user.id === currentUserId
                    const teamIds = user.teamMemberships.map((membership) => membership.teamId)

                    return (
                      <TableRow key={user.id}>
                        <TableCell className="font-medium text-foreground">
                          {user.email}
                          {isCurrentUser ? (
                            <span className="ml-2 text-xs text-muted-foreground">
                              ({t('users.you')})
                            </span>
                          ) : null}
                        </TableCell>
                        <TableCell>
                          <Select
                            disabled={isCurrentUser || updatingUserId === user.id}
                            items={roleItems}
                            onValueChange={(value) => void handleRoleChange(user.id, value as UserRole)}
                            value={user.role}
                          >
                            <SelectTrigger
                              aria-label={t('users.role.changeLabel', { email: user.email })}
                              className="w-36"
                              size="sm"
                            >
                              <SelectValue>{roleLabel(user.role)}</SelectValue>
                            </SelectTrigger>
                            <SelectPopup>
                              {roleItems.map((item) => (
                                <SelectItem key={item.value} value={item.value}>
                                  {item.label}
                                </SelectItem>
                              ))}
                            </SelectPopup>
                          </Select>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatTeamNames(teamIds)}
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </>
        )}
      </section>
    </AppShell>
  )
}
