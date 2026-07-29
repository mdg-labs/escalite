import type { ReactElement } from 'react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router'
import { useClient } from 'urql'
import {
  AlertStatus,
  OnCallNowDocument,
  SchedulesDocument,
  UserRole,
  useAlertsQuery,
  useMeQuery,
  useMyOnCallStatusQuery,
  useOnCallUpdatedSubscription,
  useOrganizationUsersQuery,
  useTeamsQuery,
  type OnCallNowQuery,
  type SchedulesQuery,
} from '@escalite/ts-types'
import {
  Frame,
  FrameDescription,
  FrameHeader,
  FramePanel,
  FrameTitle,
} from '@escalite/ui'
import { Metric } from '@escalite/ui/charts'
import { OnCallWidget } from '@escalite/ui/domain/OnCallWidget'
import type { OnCallWidgetLabels, OnCallWidgetSchedule, OnCallWidgetUser } from '@escalite/ui/domain/OnCallWidget'
import { BellIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import { formatAlertTimestamp } from '../lib/alerts'
import {
  buildUntilByScheduleLayer,
  mapOnCallSchedules,
  updateOnCallSchedule,
  viewerIsOnCall,
} from '../lib/dashboard-oncall'
import { t } from '../lib/i18n'

function onCallWidgetLabels(): OnCallWidgetLabels {
  return {
    title: t('dashboard.onCall.title'),
    loading: t('dashboard.onCall.loading'),
    empty: t('dashboard.onCall.empty'),
    layerLabel: (layer) => t('schedule.layer', { layer: String(layer) }),
    primaryLayer: t('dashboard.onCall.primaryLayer'),
    secondaryLayer: t('dashboard.onCall.secondaryLayer'),
    computedAt: t('schedule.computedAt'),
    youAreOnCall: t('dashboard.onCall.youAreOnCall'),
    until: (until) => t('dashboard.onCall.until', { until }),
  }
}

export function DashboardPage(): ReactElement {
  const client = useClient()
  const onCallLabels = useMemo(() => onCallWidgetLabels(), [])

  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const currentUser = meData?.me
  const organizationId = currentUser?.organizationId ?? ''
  const isAdmin = currentUser?.role === UserRole.Admin

  const [{ data: alertsData, fetching: alertsFetching }] = useAlertsQuery({
    variables: { limit: 500 },
    requestPolicy: 'network-only',
  })

  const [{ data: teamsData }] = useTeamsQuery({
    pause: !isAdmin,
    requestPolicy: 'cache-first',
  })

  const [{ data: usersData }] = useOrganizationUsersQuery({
    requestPolicy: 'cache-first',
  })

  const [{ data: myOnCallData }, reexecuteMyOnCallStatus] = useMyOnCallStatusQuery({
    requestPolicy: 'cache-and-network',
  })

  const myOnCallAssignments = useMemo(
    () => myOnCallData?.myOnCallStatus ?? [],
    [myOnCallData?.myOnCallStatus],
  )

  const alertCounts = useMemo(() => {
    const counts = {
      triggered: 0,
      acknowledged: 0,
    }

    for (const alert of alertsData?.alerts ?? []) {
      if (alert.status === AlertStatus.Triggered) {
        counts.triggered += 1
      } else if (alert.status === AlertStatus.Acknowledged) {
        counts.acknowledged += 1
      }
    }

    return counts
  }, [alertsData?.alerts])

  const memberTeamIds = useMemo(() => {
    if (!currentUser?.id) {
      return []
    }

    const viewer = usersData?.organizationUsers.find((user) => user.id === currentUser.id)
    return viewer?.teamMemberships.map((membership) => membership.teamId) ?? []
  }, [currentUser?.id, usersData?.organizationUsers])

  const teamIds = useMemo(() => {
    if (isAdmin) {
      return (teamsData?.teams ?? []).map((team) => team.id)
    }
    return memberTeamIds
  }, [isAdmin, memberTeamIds, teamsData?.teams])

  const onCallUsers = useMemo((): OnCallWidgetUser[] => {
    return (usersData?.organizationUsers ?? []).map((user) => ({
      id: user.id,
      label: user.email,
    }))
  }, [usersData?.organizationUsers])

  const [onCallSchedules, setOnCallSchedules] = useState<OnCallWidgetSchedule[]>([])
  const [onCallLoading, setOnCallLoading] = useState(false)
  const scheduleByIdRef = useRef(new Map<string, SchedulesQuery['schedules'][number]>())

  const refreshScheduleOnCall = useCallback(
    async (scheduleId: string): Promise<void> => {
      const schedule = scheduleByIdRef.current.get(scheduleId)
      if (!schedule) {
        return
      }

      const onCallResult = await client
        .query(OnCallNowDocument, { scheduleId }, { requestPolicy: 'network-only' })
        .toPromise()

      if (onCallResult.error || !onCallResult.data?.onCallNow) {
        return
      }

      const [updated] = mapOnCallSchedules(
        [{ schedule, onCall: onCallResult.data.onCallNow }],
        formatAlertTimestamp,
      )
      if (!updated) {
        return
      }

      setOnCallSchedules((current) => updateOnCallSchedule(current, updated))
    },
    [client],
  )

  const refreshMyOnCallStatus = useCallback((): void => {
    reexecuteMyOnCallStatus({ requestPolicy: 'network-only' })
  }, [reexecuteMyOnCallStatus])

  useOnCallUpdatedSubscription(
    {
      variables: { orgId: organizationId },
      pause: !organizationId,
    },
    (_previous, response) => {
      const scheduleId = response.onCallUpdated?.scheduleId
      if (scheduleId) {
        void refreshScheduleOnCall(scheduleId)
      }
      refreshMyOnCallStatus()
      return response
    },
  )

  useEffect(() => {
    if (teamIds.length === 0) {
      scheduleByIdRef.current.clear()
      setOnCallSchedules([])
      setOnCallLoading(false)
      return
    }

    let cancelled = false

    async function loadOnCall(): Promise<void> {
      setOnCallLoading(true)
      const loaded: Array<{
        schedule: SchedulesQuery['schedules'][number]
        onCall: NonNullable<OnCallNowQuery['onCallNow']>
      }> = []

      for (const teamId of teamIds) {
        const schedulesResult = await client
          .query(SchedulesDocument, { teamId }, { requestPolicy: 'network-only' })
          .toPromise()

        if (cancelled || schedulesResult.error || !schedulesResult.data?.schedules) {
          continue
        }

        for (const schedule of schedulesResult.data.schedules) {
          const onCallResult = await client
            .query(OnCallNowDocument, { scheduleId: schedule.id }, { requestPolicy: 'network-only' })
            .toPromise()

          if (cancelled || onCallResult.error || !onCallResult.data?.onCallNow) {
            continue
          }

          loaded.push({
            schedule,
            onCall: onCallResult.data.onCallNow,
          })
        }
      }

      if (!cancelled) {
        scheduleByIdRef.current = new Map(
          loaded.map((entry) => [entry.schedule.id, entry.schedule]),
        )
        setOnCallSchedules(mapOnCallSchedules(loaded, formatAlertTimestamp))
        setOnCallLoading(false)
      }
    }

    void loadOnCall()

    return () => {
      cancelled = true
    }
  }, [client, teamIds])

  const untilByScheduleLayer = useMemo(
    () => buildUntilByScheduleLayer(myOnCallAssignments, formatAlertTimestamp),
    [myOnCallAssignments],
  )

  const viewerOnCall = useMemo(
    () => viewerIsOnCall(currentUser?.id, onCallSchedules, myOnCallAssignments),
    [currentUser?.id, myOnCallAssignments, onCallSchedules],
  )

  return (
    <AppShell title={t('nav.dashboard')}>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">{t('dashboard.title')}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t('dashboard.description')}</p>
          {currentUser?.email ? (
            <p className="mt-2 text-sm text-muted-foreground">
              {t('dashboard.signedInAs', { email: currentUser.email })}
            </p>
          ) : null}
        </div>

        <section aria-labelledby="dashboard-open-alerts-heading" className="space-y-3">
          <h2 className="text-sm font-medium text-foreground" id="dashboard-open-alerts-heading">
            {t('dashboard.alerts.title')}
          </h2>
          <div className="grid gap-4 sm:grid-cols-2">
            <Link
              className="rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              to="/alerts"
            >
              <Frame className="transition-colors hover:bg-muted/30">
                <FramePanel className="space-y-2">
                  <FrameHeader className="px-0 py-0">
                    <FrameTitle className="flex items-center gap-2 text-muted-foreground">
                      <BellIcon aria-hidden className="size-4" />
                      {t('dashboard.alerts.triggered')}
                    </FrameTitle>
                  </FrameHeader>
                  <Metric>{alertsFetching ? '—' : String(alertCounts.triggered)}</Metric>
                  <FrameDescription className="px-0">
                    {t('dashboard.alerts.viewTriggered')}
                  </FrameDescription>
                </FramePanel>
              </Frame>
            </Link>
            <Link
              className="rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              to="/alerts"
            >
              <Frame className="transition-colors hover:bg-muted/30">
                <FramePanel className="space-y-2">
                  <FrameHeader className="px-0 py-0">
                    <FrameTitle className="flex items-center gap-2 text-muted-foreground">
                      <BellIcon aria-hidden className="size-4" />
                      {t('dashboard.alerts.acknowledged')}
                    </FrameTitle>
                  </FrameHeader>
                  <Metric>{alertsFetching ? '—' : String(alertCounts.acknowledged)}</Metric>
                  <FrameDescription className="px-0">
                    {t('dashboard.alerts.viewAcknowledged')}
                  </FrameDescription>
                </FramePanel>
              </Frame>
            </Link>
          </div>
        </section>

        <OnCallWidget
          highlighted={viewerOnCall}
          labels={onCallLabels}
          loading={onCallLoading}
          schedules={onCallSchedules}
          untilByScheduleLayer={untilByScheduleLayer}
          users={onCallUsers}
          viewerUserId={currentUser?.id}
        />
      </div>
    </AppShell>
  )
}
