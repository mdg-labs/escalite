import type { ReactElement } from 'react'
import { useMemo, useState } from 'react'
import {
  UserRole,
  useAlertAnalyticsQuery,
  useAnalyticsSettingsQuery,
  useMeQuery,
  useSaveAnalyticsSettingsMutation,
  useServicesQuery,
  useTeamsQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  Frame,
  FrameDescription,
  FrameHeader,
  FramePanel,
  FrameTitle,
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
  selectOptionsFromEntities,
  selectOptionsWithSentinel,
} from '@escalite/ui'
import { BarChart, Metric } from '@escalite/ui/charts'
import { BarChart3Icon, ClockIcon, TimerIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import {
  formatDurationSeconds,
  minutesValueFormatter,
  responseTimeChartData,
  rollupByWindowDays,
  volumeChartData,
} from '../lib/analytics'
import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

const ALL_TEAMS_VALUE = '__all__'
const ALL_SERVICES_VALUE = '__all__'

export function AnalyticsPage(): ReactElement {
  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const isAdmin = meData?.me?.role === UserRole.Admin

  const [{ data: teamsData, fetching: teamsFetching }] = useTeamsQuery({
    requestPolicy: 'cache-first',
  })
  const [{ data: servicesData }] = useServicesQuery({ requestPolicy: 'cache-first' })

  const teams = useMemo(() => teamsData?.teams ?? [], [teamsData?.teams])
  const services = useMemo(() => servicesData?.services ?? [], [servicesData?.services])

  const [teamId, setTeamId] = useState<string>(ALL_TEAMS_VALUE)
  const [serviceId, setServiceId] = useState<string>(ALL_SERVICES_VALUE)
  const [settingsError, setSettingsError] = useState<string | null>(null)
  const [savingSettings, setSavingSettings] = useState(false)

  const [{ data: settingsData, fetching: settingsFetching }] = useAnalyticsSettingsQuery({
    pause: !isAdmin,
    requestPolicy: 'cache-first',
  })
  const [, saveAnalyticsSettings] = useSaveAnalyticsSettingsMutation()

  const effectiveTeamId = useMemo(() => {
    if (isAdmin) {
      return teamId === ALL_TEAMS_VALUE ? undefined : teamId
    }
    if (teamId !== ALL_TEAMS_VALUE) {
      return teamId
    }
    return teams[0]?.id
  }, [isAdmin, teamId, teams])

  const effectiveServiceId =
    serviceId === ALL_SERVICES_VALUE ? undefined : serviceId

  const filteredServices = useMemo(() => {
    if (!effectiveTeamId) {
      return services
    }
    return services.filter((service) => service.teamId === effectiveTeamId)
  }, [effectiveTeamId, services])

  const teamItems = useMemo(() => {
    const teamOptions = selectOptionsFromEntities(teams)
    if (isAdmin) {
      return selectOptionsWithSentinel(t('analytics.filter.allTeams'), ALL_TEAMS_VALUE, teamOptions)
    }
    return teamOptions
  }, [isAdmin, teams])

  const serviceItems = useMemo(
    () =>
      selectOptionsWithSentinel(
        t('analytics.filter.allServices'),
        ALL_SERVICES_VALUE,
        selectOptionsFromEntities(filteredServices),
      ),
    [filteredServices],
  )

  const [{ data, fetching, error }, reexecuteAnalytics] = useAlertAnalyticsQuery({
    variables: {
      teamId: effectiveTeamId,
      serviceId: effectiveServiceId,
    },
    pause: !isAdmin && !effectiveTeamId,
    requestPolicy: 'network-only',
  })

  const excludeMaintenance =
    settingsData?.analyticsSettings.excludeMaintenanceWindowAlerts ?? false

  async function handleExcludeMaintenanceChange(checked: boolean): Promise<void> {
    setSettingsError(null)
    setSavingSettings(true)

    const result = await saveAnalyticsSettings({
      input: { excludeMaintenanceWindowAlerts: checked },
    })
    setSavingSettings(false)

    if (result.error) {
      showMutationError(result.error, 'analytics.error.settings')
      return
    }

    notifyMutationSuccess('analytics.toast.saved')
    reexecuteAnalytics({ requestPolicy: 'network-only' })
  }

  const rollups = useMemo(() => data?.alertAnalytics.rollups ?? [], [data?.alertAnalytics.rollups])
  const rollup7d = rollupByWindowDays(rollups, 7)
  const rollup30d = rollupByWindowDays(rollups, 30)
  const mttaCategory = t('analytics.chart.category.mtta')
  const mttrCategory = t('analytics.chart.category.mttr')
  const acknowledgedCategory = t('analytics.chart.category.acknowledged')
  const resolvedCategory = t('analytics.chart.category.resolved')
  const responseRows = useMemo(
    () =>
      responseTimeChartData(rollups).map((row) => ({
        window: row.window,
        [mttaCategory]: row.MTTA,
        [mttrCategory]: row.MTTR,
      })),
    [mttaCategory, mttrCategory, rollups],
  )
  const volumeRows = useMemo(
    () =>
      volumeChartData(rollups).map((row) => ({
        window: row.window,
        [acknowledgedCategory]: row.Acknowledged,
        [resolvedCategory]: row.Resolved,
      })),
    [acknowledgedCategory, resolvedCategory, rollups],
  )

  const needsTeamSelection = !isAdmin && teams.length === 0 && !teamsFetching

  return (
    <AppShell title={t('analytics.title')}>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">{t('analytics.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('analytics.description')}</p>
          </div>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="min-w-48 space-y-1">
              <label className="text-xs font-medium text-muted-foreground" htmlFor="analytics-team">
                {t('analytics.filter.team')}
              </label>
              <Select
                disabled={!isAdmin && teams.length <= 1}
                items={teamItems}
                onValueChange={(value) => {
                  setTeamId(value ?? ALL_TEAMS_VALUE)
                  setServiceId(ALL_SERVICES_VALUE)
                }}
                value={isAdmin ? teamId : (effectiveTeamId ?? ALL_TEAMS_VALUE)}
              >
                <SelectTrigger id="analytics-team">
                  <SelectValue placeholder={t('analytics.filter.teamPlaceholder')} />
                </SelectTrigger>
                <SelectPopup>
                  {teamItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectPopup>
              </Select>
            </div>
            <div className="min-w-48 space-y-1">
              <label
                className="text-xs font-medium text-muted-foreground"
                htmlFor="analytics-service"
              >
                {t('analytics.filter.service')}
              </label>
              <Select
                items={serviceItems}
                onValueChange={(value) => setServiceId(value ?? ALL_SERVICES_VALUE)}
                value={serviceId}
              >
                <SelectTrigger id="analytics-service">
                  <SelectValue placeholder={t('analytics.filter.servicePlaceholder')} />
                </SelectTrigger>
                <SelectPopup>
                  {serviceItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectPopup>
              </Select>
            </div>
          </div>
        </div>

        {isAdmin ? (
          <div className="rounded-lg border border-border bg-card p-4">
            <label className="flex items-start gap-3 text-sm text-foreground">
              <input
                checked={excludeMaintenance}
                className="mt-0.5 size-4 rounded border border-input"
                disabled={settingsFetching || savingSettings}
                onChange={(event) => void handleExcludeMaintenanceChange(event.target.checked)}
                type="checkbox"
              />
              <span>
                <span className="font-medium">{t('analytics.settings.excludeMaintenance')}</span>
                <span className="mt-1 block text-muted-foreground">
                  {t('analytics.settings.excludeMaintenanceDescription')}
                </span>
              </span>
            </label>
            {settingsError ? (
              <Alert className="mt-4" variant="error">
                <AlertDescription>{settingsError}</AlertDescription>
              </Alert>
            ) : null}
          </div>
        ) : null}

        {data?.alertAnalytics.excludeMaintenanceWindowAlerts ? (
          <Alert>
            <ClockIcon />
            <AlertDescription>{t('analytics.maintenanceExcluded')}</AlertDescription>
          </Alert>
        ) : null}

        {error ? (
          <Alert variant="error">
            <AlertDescription>{formatGraphQLError(error.message)}</AlertDescription>
          </Alert>
        ) : null}

        {needsTeamSelection ? (
          <Alert>
            <AlertDescription>{t('analytics.empty.noTeam')}</AlertDescription>
          </Alert>
        ) : null}

        {fetching && rollups.length === 0 ? (
          <div className="py-16 text-center text-sm text-muted-foreground">
            {t('analytics.loading')}
          </div>
        ) : (
          <>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <Frame>
                <FramePanel className="space-y-2">
                  <FrameHeader className="px-0 py-0">
                    <FrameTitle className="flex items-center gap-2 text-muted-foreground">
                      <TimerIcon className="size-4" />
                      {t('analytics.metric.mtta7d')}
                    </FrameTitle>
                  </FrameHeader>
                  <Metric>{formatDurationSeconds(rollup7d?.mttaSeconds)}</Metric>
                  <FrameDescription className="px-0">
                    {t('analytics.metric.acknowledgedCount', {
                      count: String(rollup7d?.acknowledgedCount ?? 0),
                    })}
                  </FrameDescription>
                </FramePanel>
              </Frame>
              <Frame>
                <FramePanel className="space-y-2">
                  <FrameHeader className="px-0 py-0">
                    <FrameTitle className="flex items-center gap-2 text-muted-foreground">
                      <TimerIcon className="size-4" />
                      {t('analytics.metric.mttr7d')}
                    </FrameTitle>
                  </FrameHeader>
                  <Metric>{formatDurationSeconds(rollup7d?.mttrSeconds)}</Metric>
                  <FrameDescription className="px-0">
                    {t('analytics.metric.resolvedCount', {
                      count: String(rollup7d?.resolvedCount ?? 0),
                    })}
                  </FrameDescription>
                </FramePanel>
              </Frame>
              <Frame>
                <FramePanel className="space-y-2">
                  <FrameHeader className="px-0 py-0">
                    <FrameTitle className="flex items-center gap-2 text-muted-foreground">
                      <TimerIcon className="size-4" />
                      {t('analytics.metric.mtta30d')}
                    </FrameTitle>
                  </FrameHeader>
                  <Metric>{formatDurationSeconds(rollup30d?.mttaSeconds)}</Metric>
                  <FrameDescription className="px-0">
                    {t('analytics.metric.acknowledgedCount', {
                      count: String(rollup30d?.acknowledgedCount ?? 0),
                    })}
                  </FrameDescription>
                </FramePanel>
              </Frame>
              <Frame>
                <FramePanel className="space-y-2">
                  <FrameHeader className="px-0 py-0">
                    <FrameTitle className="flex items-center gap-2 text-muted-foreground">
                      <TimerIcon className="size-4" />
                      {t('analytics.metric.mttr30d')}
                    </FrameTitle>
                  </FrameHeader>
                  <Metric>{formatDurationSeconds(rollup30d?.mttrSeconds)}</Metric>
                  <FrameDescription className="px-0">
                    {t('analytics.metric.resolvedCount', {
                      count: String(rollup30d?.resolvedCount ?? 0),
                    })}
                  </FrameDescription>
                </FramePanel>
              </Frame>
            </div>

            <div className="grid gap-4 xl:grid-cols-2">
              <Frame>
                <FramePanel>
                  <FrameHeader className="px-0 pt-0">
                    <FrameTitle className="flex items-center gap-2">
                      <BarChart3Icon className="size-4 text-muted-foreground" />
                      {t('analytics.chart.responseTime')}
                    </FrameTitle>
                    <FrameDescription className="px-0">
                      {t('analytics.chart.responseTimeDescription')}
                    </FrameDescription>
                  </FrameHeader>
                  <BarChart
                    categories={[mttaCategory, mttrCategory]}
                    data={responseRows}
                    index="window"
                    showAnimation={false}
                    showLegend
                    valueFormatter={minutesValueFormatter}
                    yAxisWidth={48}
                  />
                </FramePanel>
              </Frame>
              <Frame>
                <FramePanel>
                  <FrameHeader className="px-0 pt-0">
                    <FrameTitle className="flex items-center gap-2">
                      <BarChart3Icon className="size-4 text-muted-foreground" />
                      {t('analytics.chart.volume')}
                    </FrameTitle>
                    <FrameDescription className="px-0">
                      {t('analytics.chart.volumeDescription')}
                    </FrameDescription>
                  </FrameHeader>
                  <BarChart
                    categories={[acknowledgedCategory, resolvedCategory]}
                    colors={['blue', 'emerald']}
                    data={volumeRows}
                    index="window"
                    showAnimation={false}
                    showLegend
                    yAxisWidth={48}
                  />
                </FramePanel>
              </Frame>
            </div>
          </>
        )}
      </div>
    </AppShell>
  )
}
