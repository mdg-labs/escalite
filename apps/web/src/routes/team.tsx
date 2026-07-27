import type { ReactElement } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router'
import {
  useAddTeamMemberMutation,
  useOrganizationUsersQuery,
  useRemoveTeamMemberMutation,
  useTeamsQuery,
  useUpdateTeamMutation,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  AlertTitle,
  Button,
  Combobox,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxPopup,
  Input,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@escalite/ui'
import { AlertTriangleIcon, PlusIcon, TrashIcon, UserPlusIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import { PageBreadcrumbs } from '../components/page-breadcrumbs'
import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

type TeamMemberRow = {
  membershipId: string
  userId: string
  email: string
  role: string
  joinedAt: string
}

export function TeamPage(): ReactElement {
  const { teamId } = useParams()

  const [{ data: teamsData, fetching: teamsFetching, error: teamsError }, reexecuteTeams] =
    useTeamsQuery({ requestPolicy: 'network-only' })
  const [
    { data: usersData, fetching: usersFetching, error: usersError },
    reexecuteUsers,
  ] = useOrganizationUsersQuery({ requestPolicy: 'network-only' })

  const [, updateTeam] = useUpdateTeamMutation()
  const [, addTeamMember] = useAddTeamMemberMutation()
  const [, removeTeamMember] = useRemoveTeamMemberMutation()

  const [teamName, setTeamName] = useState('')
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [renameError, setRenameError] = useState<string | null>(null)
  const [renaming, setRenaming] = useState(false)
  const [adding, setAdding] = useState(false)
  const [removingUserId, setRemovingUserId] = useState<string | null>(null)

  const team = useMemo(
    () => teamsData?.teams.find((item) => item.id === teamId),
    [teamId, teamsData?.teams],
  )

  useEffect(() => {
    if (team?.name) {
      setTeamName(team.name)
    }
  }, [team?.name])

  const members = useMemo((): TeamMemberRow[] => {
    if (!teamId) {
      return []
    }

    const rows: TeamMemberRow[] = []
    for (const user of usersData?.organizationUsers ?? []) {
      const membership = user.teamMemberships.find((item) => item.teamId === teamId)
      if (!membership) {
        continue
      }

      rows.push({
        membershipId: membership.id,
        userId: user.id,
        email: user.email,
        role: user.role,
        joinedAt: membership.createdAt,
      })
    }

    return rows.sort((left, right) => left.email.localeCompare(right.email))
  }, [teamId, usersData?.organizationUsers])

  const memberUserIds = useMemo(() => new Set(members.map((member) => member.userId)), [members])

  const availableUsers = useMemo(() => {
    return (usersData?.organizationUsers ?? [])
      .filter((user) => !memberUserIds.has(user.id))
      .sort((left, right) => left.email.localeCompare(right.email))
  }, [memberUserIds, usersData?.organizationUsers])

  const loading = teamsFetching || usersFetching
  const loadError = teamsError ?? usersError
  const teamTitle = team?.name ?? t('team.detail.title')
  const breadcrumbItems = useMemo(
    () => [
      { label: t('nav.teams'), href: '/teams' },
      { label: teamTitle },
    ],
    [teamTitle],
  )

  function refresh(): void {
    reexecuteTeams({ requestPolicy: 'network-only' })
    reexecuteUsers({ requestPolicy: 'network-only' })
  }

  async function handleRename(): Promise<void> {
    if (!teamId || !team) {
      return
    }

    const trimmed = teamName.trim()
    if (!trimmed) {
      setRenameError(t('teams.error.requiredName'))
      setTeamName(team.name)
      return
    }

    if (trimmed === team.name) {
      return
    }

    setRenameError(null)
    setRenaming(true)
    const result = await updateTeam({ input: { id: teamId, name: trimmed } })
    setRenaming(false)

    if (result.error) {
      setRenameError(formatGraphQLError(result.error.message))
      setTeamName(team.name)
      return
    }

    refresh()
  }

  async function handleAddMember(): Promise<void> {
    if (!teamId || !selectedUserId) {
      return
    }

    setActionError(null)
    setAdding(true)
    const result = await addTeamMember({ teamId, userId: selectedUserId })
    setAdding(false)

    if (result.error) {
      setActionError(formatGraphQLError(result.error.message))
      return
    }

    setSelectedUserId(null)
    refresh()
  }

  async function handleRemoveMember(userId: string): Promise<void> {
    if (!teamId) {
      return
    }

    setActionError(null)
    setRemovingUserId(userId)
    const result = await removeTeamMember({ teamId, userId })
    setRemovingUserId(null)

    if (result.error) {
      setActionError(formatGraphQLError(result.error.message))
      return
    }

    refresh()
  }

  return (
    <AppShell title={teamTitle}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <PageBreadcrumbs items={breadcrumbItems} />

        <div className="mt-4 flex items-start gap-3">
          <UserPlusIcon className="mt-2 size-5 shrink-0 text-muted-foreground" />
          <div className="min-w-0 flex-1">
            <label className="text-sm font-medium text-foreground" htmlFor="team-name">
              {t('teams.field.name')}
            </label>
            <Input
              className="mt-2 max-w-md"
              disabled={!team || renaming}
              id="team-name"
              onBlur={() => void handleRename()}
              onChange={(event) => setTeamName(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  event.currentTarget.blur()
                }
              }}
              placeholder={t('teams.field.namePlaceholder')}
              value={teamName}
            />
            {renaming ? (
              <p className="mt-2 text-sm text-muted-foreground">{t('team.detail.renaming')}</p>
            ) : null}
          </div>
        </div>

        <p className="mt-4 text-sm text-muted-foreground">{t('team.detail.description')}</p>

        {loadError ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('team.error.load')}</AlertTitle>
            <AlertDescription>{formatGraphQLError(loadError.message)}</AlertDescription>
          </Alert>
        ) : null}

        {renameError ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('team.error.rename')}</AlertTitle>
            <AlertDescription>{renameError}</AlertDescription>
          </Alert>
        ) : null}

        {actionError ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('team.error.action')}</AlertTitle>
            <AlertDescription>{actionError}</AlertDescription>
          </Alert>
        ) : null}

        {!loading && !loadError && teamId && !team ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('team.error.notFound')}</AlertTitle>
            <AlertDescription>{t('team.error.notFoundDescription')}</AlertDescription>
          </Alert>
        ) : null}

        {team ? (
          <>
            <div className="mt-8">
              <h2 className="text-lg font-medium text-foreground">{t('team.members.title')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('team.members.description')}</p>
            </div>

            <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end">
              <div className="min-w-0 flex-1 space-y-2">
                <label className="text-sm font-medium text-foreground" htmlFor="team-member-picker">
                  {t('team.members.addLabel')}
                </label>
                <Combobox
                  onValueChange={(value) => setSelectedUserId(value)}
                  value={selectedUserId}
                >
                  <ComboboxInput
                    id="team-member-picker"
                    placeholder={t('team.members.addPlaceholder')}
                    showClear
                  />
                  <ComboboxPopup>
                    <ComboboxList>
                      <ComboboxEmpty>{t('team.members.noUsers')}</ComboboxEmpty>
                      {availableUsers.map((user) => (
                        <ComboboxItem key={user.id} value={user.id}>
                          {user.email}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxPopup>
                </Combobox>
              </div>
              <Button
                disabled={!selectedUserId || adding || availableUsers.length === 0}
                onClick={() => void handleAddMember()}
                type="button"
              >
                <PlusIcon />
                {adding ? t('team.members.adding') : t('team.members.add')}
              </Button>
            </div>

            <Table className="mt-6" variant="card">
              <TableHeader>
                <TableRow>
                  <TableHead>{t('team.members.column.email')}</TableHead>
                  <TableHead>{t('team.members.column.role')}</TableHead>
                  <TableHead>{t('team.members.column.joined')}</TableHead>
                  <TableHead className="w-24 text-right">{t('team.members.column.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={4}>
                      {t('team.members.loading')}
                    </TableCell>
                  </TableRow>
                ) : members.length === 0 ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={4}>
                      {t('team.members.empty')}
                    </TableCell>
                  </TableRow>
                ) : (
                  members.map((member) => (
                    <TableRow key={member.membershipId}>
                      <TableCell className="font-medium text-foreground">{member.email}</TableCell>
                      <TableCell className="text-muted-foreground">{member.role}</TableCell>
                      <TableCell className="text-muted-foreground">
                        {new Date(member.joinedAt).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          aria-label={t('team.members.remove', { email: member.email })}
                          disabled={removingUserId === member.userId}
                          onClick={() => void handleRemoveMember(member.userId)}
                          size="sm"
                          type="button"
                          variant="ghost"
                        >
                          <TrashIcon />
                          {removingUserId === member.userId
                            ? t('team.members.removing')
                            : t('team.members.removeAction')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </>
        ) : null}
      </section>
    </AppShell>
  )
}
