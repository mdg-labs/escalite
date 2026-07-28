import type { ReactElement } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router'
import { useClient } from 'urql'
import {
  SchedulesDocument,
  UserRole,
  useMeQuery,
  useOrganizationUsersQuery,
  useTeamsQuery,
  type SchedulesQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  AlertTitle,
  Button,
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
import { AlertTriangleIcon, CalendarClockIcon, PlusIcon, SearchIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

const ALL_TEAMS_VALUE = '__all__'

type ScheduleRow = SchedulesQuery['schedules'][number] & {
  teamName: string
}

export function SchedulesPage(): ReactElement {
  const client = useClient()

  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const currentUser = meData?.me
  const isAdmin = currentUser?.role === UserRole.Admin

  const [{ data: teamsData, fetching: teamsFetching }] = useTeamsQuery({
    requestPolicy: 'cache-first',
  })
  const [{ data: usersData }] = useOrganizationUsersQuery({
    requestPolicy: 'cache-first',
  })

  const [teamFilter, setTeamFilter] = useState(ALL_TEAMS_VALUE)
  const [search, setSearch] = useState('')
  const [schedules, setSchedules] = useState<ScheduleRow[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const memberTeamIds = useMemo(() => {
    if (!currentUser?.id) {
      return []
    }

    const viewer = usersData?.organizationUsers.find((user) => user.id === currentUser.id)
    return viewer?.teamMemberships.map((membership) => membership.teamId) ?? []
  }, [currentUser?.id, usersData?.organizationUsers])

  const visibleTeams = useMemo(() => {
    const teams = teamsData?.teams ?? []
    if (isAdmin) {
      return teams
    }

    const memberSet = new Set(memberTeamIds)
    return teams.filter((team) => memberSet.has(team.id))
  }, [isAdmin, memberTeamIds, teamsData?.teams])

  const teamNameById = useMemo(
    () => new Map(visibleTeams.map((team) => [team.id, team.name])),
    [visibleTeams],
  )

  const teamIdsToLoad = useMemo(() => {
    if (teamFilter !== ALL_TEAMS_VALUE) {
      return [teamFilter]
    }
    return visibleTeams.map((team) => team.id)
  }, [teamFilter, visibleTeams])

  useEffect(() => {
    if (teamsFetching) {
      return
    }

    if (teamIdsToLoad.length === 0) {
      setSchedules([])
      setLoading(false)
      setLoadError(null)
      return
    }

    let cancelled = false

    async function loadSchedules(): Promise<void> {
      setLoading(true)
      setLoadError(null)

      const loaded: ScheduleRow[] = []

      for (const teamId of teamIdsToLoad) {
        const result = await client
          .query(SchedulesDocument, { teamId }, { requestPolicy: 'network-only' })
          .toPromise()

        if (cancelled) {
          return
        }

        if (result.error) {
          setLoadError(formatGraphQLError(result.error.message))
          setSchedules([])
          setLoading(false)
          return
        }

        const teamName = teamNameById.get(teamId) ?? teamId
        for (const schedule of result.data?.schedules ?? []) {
          loaded.push({ ...schedule, teamName })
        }
      }

      if (!cancelled) {
        loaded.sort((left, right) => left.name.localeCompare(right.name))
        setSchedules(loaded)
        setLoading(false)
      }
    }

    void loadSchedules()

    return () => {
      cancelled = true
    }
  }, [client, teamIdsToLoad, teamNameById, teamsFetching])

  const filteredSchedules = useMemo(() => {
    const query = search.trim().toLowerCase()
    if (!query) {
      return schedules
    }

    return schedules.filter(
      (schedule) =>
        schedule.name.toLowerCase().includes(query) ||
        schedule.teamName.toLowerCase().includes(query) ||
        schedule.timezone.toLowerCase().includes(query),
    )
  }, [schedules, search])

  const showEmptyState = !loading && !loadError && schedules.length === 0

  return (
    <AppShell title={t('schedules.title')}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold text-foreground">{t('schedules.title')}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{t('schedules.description')}</p>
          </div>
          {!showEmptyState ? (
            <Button type="button">
              <PlusIcon />
              {t('schedules.action.create')}
            </Button>
          ) : null}
        </div>

        {loadError ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('schedules.error.load')}</AlertTitle>
            <AlertDescription>{loadError}</AlertDescription>
          </Alert>
        ) : null}

        {showEmptyState ? (
          <div className="mt-10 flex flex-col items-center justify-center rounded-lg border border-dashed border-border px-6 py-16 text-center">
            <div className="flex size-12 items-center justify-center rounded-full bg-muted">
              <CalendarClockIcon className="size-6 text-muted-foreground" />
            </div>
            <h2 className="mt-4 text-lg font-medium text-foreground">{t('schedules.empty.title')}</h2>
            <p className="mt-2 max-w-sm text-sm text-muted-foreground">
              {t('schedules.empty.description')}
            </p>
            <div className="mt-6">
              <Button type="button">
                <PlusIcon />
                {t('schedules.action.create')}
              </Button>
            </div>
          </div>
        ) : (
          <>
            <div className="mt-6 flex flex-col gap-4 sm:flex-row sm:items-end">
              <div className="relative max-w-md flex-1">
                <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  aria-label={t('schedules.search.label')}
                  className="pl-9"
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder={t('schedules.search.placeholder')}
                  value={search}
                />
              </div>
              <div className="min-w-48 space-y-1">
                <label className="text-xs font-medium text-muted-foreground" htmlFor="schedules-team">
                  {t('schedules.filter.team')}
                </label>
                <Select
                  disabled={!isAdmin && visibleTeams.length <= 1}
                  onValueChange={(value) => setTeamFilter(value ?? ALL_TEAMS_VALUE)}
                  value={teamFilter}
                >
                  <SelectTrigger id="schedules-team">
                    <SelectValue placeholder={t('schedules.filter.teamPlaceholder')} />
                  </SelectTrigger>
                  <SelectPopup>
                    {isAdmin ? (
                      <SelectItem value={ALL_TEAMS_VALUE}>{t('schedules.filter.allTeams')}</SelectItem>
                    ) : null}
                    {visibleTeams.map((team) => (
                      <SelectItem key={team.id} value={team.id}>
                        {team.name}
                      </SelectItem>
                    ))}
                  </SelectPopup>
                </Select>
              </div>
            </div>

            <Table className="mt-6" variant="card">
              <TableHeader>
                <TableRow>
                  <TableHead>{t('schedules.column.name')}</TableHead>
                  <TableHead>{t('schedules.column.team')}</TableHead>
                  <TableHead>{t('schedules.column.timezone')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={3}>
                      {t('schedules.loading')}
                    </TableCell>
                  </TableRow>
                ) : filteredSchedules.length === 0 ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={3}>
                      {search.trim() ? t('schedules.empty.search') : t('schedules.empty.default')}
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredSchedules.map((schedule) => (
                    <TableRow key={schedule.id}>
                      <TableCell className="font-medium text-foreground">
                        <Link className="text-primary hover:underline" to={`/schedules/${schedule.id}`}>
                          {schedule.name}
                        </Link>
                      </TableCell>
                      <TableCell className="text-muted-foreground">{schedule.teamName}</TableCell>
                      <TableCell className="text-muted-foreground">{schedule.timezone}</TableCell>
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
