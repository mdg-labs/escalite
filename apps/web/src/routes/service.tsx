import type { ReactElement } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import {
  AlertPriority,
  useDeleteServiceMutation,
  useEscalationPoliciesQuery,
  useSchedulesQuery,
  useServiceQuery,
  useTeamsQuery,
  useUpdateServiceMutation,
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
  Input,
  Tabs,
  TabsList,
  TabsPanel,
  TabsTab,
} from '@escalite/ui'
import { AlertTriangleIcon, SettingsIcon } from 'lucide-react'

import { IntegrationKeysPanel } from '../components/integration-keys-panel'
import { MaintenanceWindowsPanel } from '../components/maintenance-windows-panel'
import { AppShell } from '../components/app-shell'
import { PageBreadcrumbs } from '../components/page-breadcrumbs'
import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

export function ServicePage(): ReactElement {
  const { serviceId } = useParams()
  const navigate = useNavigate()

  const [{ data, fetching, error }] = useServiceQuery({
    pause: !serviceId,
    variables: { id: serviceId ?? '' },
    requestPolicy: 'network-only',
  })

  const service = data?.service
  const [{ data: teamsData }] = useTeamsQuery()
  const [{ data: policiesData }] = useEscalationPoliciesQuery({
    pause: !serviceId,
    variables: { serviceId: serviceId ?? '' },
  })
  const [{ data: schedulesData }] = useSchedulesQuery({
    pause: !service?.teamId,
    variables: { teamId: service?.teamId ?? '' },
  })

  const [, updateService] = useUpdateServiceMutation()
  const [, deleteService] = useDeleteServiceMutation()

  const [name, setName] = useState('')
  const [autoPromoteEnabled, setAutoPromoteEnabled] = useState(false)
  const [autoPromoteAlertThreshold, setAutoPromoteAlertThreshold] = useState('3')
  const [autoPromoteWindowSeconds, setAutoPromoteWindowSeconds] = useState('300')
  const [suppressHighPriority, setSuppressHighPriority] = useState(true)
  const [suppressLowPriority, setSuppressLowPriority] = useState(true)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [savedMessage, setSavedMessage] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    if (service?.name) {
      setName(service.name)
    }
    if (service?.autoPromoteRule) {
      setAutoPromoteEnabled(service.autoPromoteRule.enabled)
      setAutoPromoteAlertThreshold(String(service.autoPromoteRule.alertThreshold))
      setAutoPromoteWindowSeconds(String(service.autoPromoteRule.windowSeconds))
      setSuppressHighPriority(
        service.autoPromoteRule.suppressEscalationPriorities.includes(AlertPriority.High),
      )
      setSuppressLowPriority(
        service.autoPromoteRule.suppressEscalationPriorities.includes(AlertPriority.Low),
      )
    }
  }, [service?.autoPromoteRule, service?.name])

  const teamName = useMemo(() => {
    if (!service) {
      return ''
    }
    return teamsData?.teams.find((team) => team.id === service.teamId)?.name ?? service.teamId
  }, [service, teamsData?.teams])

  const policies = policiesData?.escalationPolicies ?? []
  const schedules = schedulesData?.schedules ?? []
  const activeMaintenance = service?.activeMaintenanceWindows ?? []
  const serviceTitle = service?.name ?? t('services.detail.title')
  const breadcrumbItems = useMemo(
    () => [
      { label: t('nav.services'), href: '/services' },
      { label: serviceTitle },
    ],
    [serviceTitle],
  )

  async function handleSave(): Promise<void> {
    if (!serviceId) {
      return
    }

    const trimmed = name.trim()
    if (!trimmed) {
      setSaveError(t('services.error.requiredFields'))
      return
    }

    const threshold = Number.parseInt(autoPromoteAlertThreshold, 10)
    if (!Number.isFinite(threshold) || threshold < 1) {
      setSaveError(t('services.autoPromote.error.threshold'))
      return
    }

    const windowSeconds = Number.parseInt(autoPromoteWindowSeconds, 10)
    if (!Number.isFinite(windowSeconds) || windowSeconds < 1) {
      setSaveError(t('services.autoPromote.error.window'))
      return
    }

    const suppressEscalationPriorities: AlertPriority[] = []
    if (suppressHighPriority) {
      suppressEscalationPriorities.push(AlertPriority.High)
    }
    if (suppressLowPriority) {
      suppressEscalationPriorities.push(AlertPriority.Low)
    }
    if (suppressEscalationPriorities.length === 0) {
      setSaveError(t('services.autoPromote.error.priorities'))
      return
    }

    setSaveError(null)
    setSavedMessage(null)
    setSaving(true)
    const result = await updateService({
      input: {
        id: serviceId,
        name: trimmed,
        autoPromoteEnabled,
        autoPromoteAlertThreshold: threshold,
        autoPromoteWindowSeconds: windowSeconds,
        autoPromoteSuppressEscalationPriorities: suppressEscalationPriorities,
      },
    })
    setSaving(false)

    if (result.error) {
      setSaveError(formatGraphQLError(result.error.message))
      return
    }

    setSavedMessage(t('services.detail.saved'))
  }

  async function handleDelete(): Promise<void> {
    if (!serviceId) {
      return
    }

    setSaveError(null)
    setDeleting(true)
    const result = await deleteService({ id: serviceId })
    setDeleting(false)

    if (result.error) {
      setSaveError(formatGraphQLError(result.error.message))
      return
    }

    navigate('/services')
  }

  return (
    <AppShell title={serviceTitle}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <PageBreadcrumbs items={breadcrumbItems} />
            <h1 className="mt-2 text-xl font-semibold text-foreground">{serviceTitle}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{t('services.detail.description')}</p>
          </div>
          <SettingsIcon className="size-5 text-muted-foreground" />
        </div>

        {fetching ? (
          <p className="mt-6 text-sm text-muted-foreground">{t('services.detail.loading')}</p>
        ) : null}

        {error ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('services.error.load')}</AlertTitle>
            <AlertDescription>{formatGraphQLError(error.message)}</AlertDescription>
          </Alert>
        ) : null}

        {!fetching && !error && !service ? (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertTitle>{t('services.detail.notFound')}</AlertTitle>
          </Alert>
        ) : null}

        {service ? (
          <div className="mt-6">
            {activeMaintenance.length > 0 ? (
              <Alert className="mb-6" variant="warning">
                <AlertTriangleIcon />
                <AlertTitle>{t('services.maintenance.banner.title')}</AlertTitle>
                <AlertDescription>
                  {activeMaintenance.map((window) => (
                    <p key={window.id}>
                      {t('services.maintenance.banner.description', {
                        label: window.description,
                        endsAt: new Date(window.endsAt).toLocaleString(),
                      })}
                    </p>
                  ))}
                </AlertDescription>
              </Alert>
            ) : null}

            <Tabs defaultValue="general">
              <TabsList variant="underline">
                <TabsTab value="general">{t('services.tabs.general')}</TabsTab>
                <TabsTab value="escalation">{t('services.tabs.escalation')}</TabsTab>
                <TabsTab value="integrations">{t('services.tabs.integrations')}</TabsTab>
                <TabsTab value="maintenance">{t('services.tabs.maintenance')}</TabsTab>
                <TabsTab value="schedules">{t('services.tabs.schedules')}</TabsTab>
              </TabsList>

              <TabsPanel className="mt-6" value="general">
                <div className="max-w-lg space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground" htmlFor="service-edit-name">
                      {t('services.field.name')}
                    </label>
                    <Input
                      id="service-edit-name"
                      onChange={(event) => setName(event.target.value)}
                      value={name}
                    />
                  </div>
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-foreground">{t('services.field.team')}</p>
                    <p className="text-sm text-muted-foreground">{teamName}</p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button disabled={saving} onClick={() => void handleSave()} type="button">
                      {saving ? t('services.action.saving') : t('services.action.save')}
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger
                        render={
                          <Button
                            disabled={deleting}
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
                            onClick={() => void handleDelete()}
                            render={<Button type="button" variant="destructive" />}
                          >
                            {t('services.action.deleteConfirm')}
                          </AlertDialogClose>
                        </AlertDialogFooter>
                      </AlertDialogPopup>
                    </AlertDialog>
                  </div>
                  {saveError ? (
                    <Alert variant="error">
                      <AlertTriangleIcon />
                      <AlertDescription>{saveError}</AlertDescription>
                    </Alert>
                  ) : null}
                  {savedMessage ? (
                    <p className="text-sm text-success-foreground" role="status">
                      {savedMessage}
                    </p>
                  ) : null}
                </div>
              </TabsPanel>

              <TabsPanel className="mt-6" value="escalation">
                <div className="space-y-8">
                  <div className="max-w-lg space-y-4 rounded-lg border border-border p-4">
                    <div>
                      <h2 className="text-lg font-semibold text-foreground">
                        {t('services.autoPromote.title')}
                      </h2>
                      <p className="text-sm text-muted-foreground">
                        {t('services.autoPromote.description')}
                      </p>
                    </div>
                    <label className="flex items-center gap-2 text-sm text-foreground">
                      <input
                        checked={autoPromoteEnabled}
                        onChange={(event) => setAutoPromoteEnabled(event.target.checked)}
                        type="checkbox"
                      />
                      {t('services.autoPromote.enabled')}
                    </label>
                    <div className="grid gap-4 sm:grid-cols-2">
                      <div className="space-y-2">
                        <label
                          className="text-sm font-medium text-foreground"
                          htmlFor="auto-promote-threshold"
                        >
                          {t('services.autoPromote.alertThreshold')}
                        </label>
                        <Input
                          disabled={!autoPromoteEnabled}
                          id="auto-promote-threshold"
                          inputMode="numeric"
                          min={1}
                          onChange={(event) => setAutoPromoteAlertThreshold(event.target.value)}
                          type="number"
                          value={autoPromoteAlertThreshold}
                        />
                      </div>
                      <div className="space-y-2">
                        <label
                          className="text-sm font-medium text-foreground"
                          htmlFor="auto-promote-window"
                        >
                          {t('services.autoPromote.windowSeconds')}
                        </label>
                        <Input
                          disabled={!autoPromoteEnabled}
                          id="auto-promote-window"
                          inputMode="numeric"
                          min={1}
                          onChange={(event) => setAutoPromoteWindowSeconds(event.target.value)}
                          type="number"
                          value={autoPromoteWindowSeconds}
                        />
                      </div>
                    </div>
                    <div className="space-y-2">
                      <label className="flex items-center gap-2 text-sm text-foreground">
                        <input
                          checked={suppressHighPriority}
                          onChange={(event) => setSuppressHighPriority(event.target.checked)}
                          type="checkbox"
                        />
                        {t('services.autoPromote.suppressHigh')}
                      </label>
                      <label className="flex items-center gap-2 text-sm text-foreground">
                        <input
                          checked={suppressLowPriority}
                          onChange={(event) => setSuppressLowPriority(event.target.checked)}
                          type="checkbox"
                        />
                        {t('services.autoPromote.suppressLow')}
                      </label>
                    </div>
                    <Button disabled={saving} onClick={() => void handleSave()} type="button">
                      {saving ? t('services.action.saving') : t('services.action.save')}
                    </Button>
                  </div>

                  <div className="space-y-4">
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div>
                      <h2 className="text-lg font-semibold text-foreground">
                        {t('services.escalation.title')}
                      </h2>
                      <p className="text-sm text-muted-foreground">
                        {t('services.escalation.description')}
                      </p>
                    </div>
                    <Button
                      render={
                        <Link
                          to={`/services/${service.id}/escalation-policies/new?serviceId=${service.id}`}
                        />
                      }
                      type="button"
                    >
                      {t('services.escalation.create')}
                    </Button>
                  </div>
                  {policies.length === 0 ? (
                    <p className="text-sm text-muted-foreground">{t('services.escalation.empty')}</p>
                  ) : (
                    <ul className="space-y-2">
                      {policies.map((policy) => (
                        <li key={policy.id}>
                          <Link
                            className="text-sm font-medium text-foreground hover:underline"
                            to={`/services/${service.id}/escalation-policies/${policy.id}`}
                          >
                            {policy.name}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
                </div>
              </TabsPanel>

              <TabsPanel className="mt-6" value="integrations">
                <IntegrationKeysPanel embedded serviceId={service.id} />
              </TabsPanel>

              <TabsPanel className="mt-6" value="maintenance">
                <MaintenanceWindowsPanel serviceId={service.id} />
              </TabsPanel>

              <TabsPanel className="mt-6" value="schedules">
                <div className="space-y-4">
                  <div>
                    <h2 className="text-lg font-semibold text-foreground">
                      {t('services.schedules.title')}
                    </h2>
                    <p className="text-sm text-muted-foreground">
                      {t('services.schedules.description')}
                    </p>
                  </div>
                  {schedules.length === 0 ? (
                    <p className="text-sm text-muted-foreground">{t('services.schedules.empty')}</p>
                  ) : (
                    <ul className="space-y-2">
                      {schedules.map((schedule) => (
                        <li key={schedule.id}>
                          <Link
                            className="text-sm font-medium text-foreground hover:underline"
                            to={`/schedules/${schedule.id}`}
                          >
                            {schedule.name}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </TabsPanel>
            </Tabs>
          </div>
        ) : null}
      </section>
    </AppShell>
  )
}
