import type { ReactElement } from 'react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import {
  AlertStatus,
  type AlertFieldsFragment,
  useAcknowledgeAlertMutation,
  useAlertQuery,
  useAlertUpdatedSubscription,
  useAlertsQuery,
  useCloseAlertMutation,
  useMeQuery,
  usePromoteAlertToIncidentMutation,
  useReEscalateAlertMutation,
  useSnoozeAlertMutation,
} from '@escalite/ts-types'
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
  Badge,
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
  Group,
  GroupSeparator,
  ScrollArea,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tabs,
  TabsList,
  TabsPanel,
  TabsTab,
  toastManager,
} from '@escalite/ui'
import { AlertCircleIcon } from 'lucide-react'

import { AlertPriorityBadge } from '../components/alert-priority-badge'
import { AlertStatusBadge } from '../components/alert-status-badge'
import { AppShell } from '../components/app-shell'
import {
  canReEscalateAlert,
  canSnoozeAlert,
  SNOOZE_PRESETS,
} from '../lib/alert-escalation'
import {
  formatAlertTimestamp,
  mergeAlertUpdate,
  sortAlertsByUpdatedAt,
} from '../lib/alerts'
import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

type StatusFilter = 'ALL' | AlertStatus

const STATUS_FILTERS: StatusFilter[] = [
  'ALL',
  AlertStatus.Triggered,
  AlertStatus.Acknowledged,
  AlertStatus.Closed,
]

function matchesStatusFilter(alert: AlertFieldsFragment, filter: StatusFilter): boolean {
  if (filter === 'ALL') {
    return true
  }
  return alert.status === filter
}

function statusFilterLabel(filter: StatusFilter): string {
  switch (filter) {
    case AlertStatus.Triggered:
      return t('alerts.filter.triggered')
    case AlertStatus.Acknowledged:
      return t('alerts.filter.acknowledged')
    case AlertStatus.Closed:
      return t('alerts.filter.closed')
    default:
      return t('alerts.filter.all')
  }
}

export function AlertsPage(): ReactElement {
  const navigate = useNavigate()
  const { alertId } = useParams()
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('ALL')
  const [alerts, setAlerts] = useState<AlertFieldsFragment[]>([])
  const [actionError, setActionError] = useState<string | null>(null)

  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const organizationId = meData?.me?.organizationId ?? ''

  const [{ data, fetching }] = useAlertsQuery({
    variables: { limit: 100 },
    requestPolicy: 'network-only',
  })

  const [{ data: selectedData }] = useAlertQuery({
    variables: { id: alertId ?? '' },
    pause: !alertId,
    requestPolicy: 'network-only',
  })

  const [, acknowledgeAlert] = useAcknowledgeAlertMutation()
  const [, closeAlert] = useCloseAlertMutation()
  const [, promoteAlertToIncident] = usePromoteAlertToIncidentMutation()
  const [, snoozeAlert] = useSnoozeAlertMutation()
  const [, reEscalateAlert] = useReEscalateAlertMutation()
  const [ackLoading, setAckLoading] = useState(false)
  const [closeLoading, setCloseLoading] = useState(false)
  const [promoteLoading, setPromoteLoading] = useState(false)
  const [snoozeLoading, setSnoozeLoading] = useState(false)
  const [reEscalateLoading, setReEscalateLoading] = useState(false)
  const [snoozeOpen, setSnoozeOpen] = useState(false)

  useEffect(() => {
    if (data?.alerts) {
      setAlerts(sortAlertsByUpdatedAt(data.alerts))
    }
  }, [data?.alerts])

  const applyAlertUpdate = useCallback((updated: AlertFieldsFragment) => {
    setAlerts((current) => sortAlertsByUpdatedAt(mergeAlertUpdate(current, updated)))
  }, [])

  useAlertUpdatedSubscription(
    {
      variables: { orgId: organizationId },
      pause: !organizationId,
    },
    (_previous, response) => {
      const updated = response.alertUpdated
      if (updated) {
        applyAlertUpdate(updated)
      }
      return response
    },
  )

  const selectedAlert = useMemo(() => {
    if (!alertId) {
      return undefined
    }
    return (
      alerts.find((alert) => alert.id === alertId) ??
      selectedData?.alert ??
      undefined
    )
  }, [alertId, alerts, selectedData?.alert])

  const statusCounts = useMemo(() => {
    const counts: Record<StatusFilter, number> = {
      ALL: alerts.length,
      TRIGGERED: 0,
      ACKNOWLEDGED: 0,
      CLOSED: 0,
    }
    for (const alert of alerts) {
      counts[alert.status] += 1
    }
    return counts
  }, [alerts])

  const visibleAlerts = useMemo(
    () => alerts.filter((alert) => matchesStatusFilter(alert, statusFilter)),
    [alerts, statusFilter],
  )

  async function handleAcknowledge(): Promise<void> {
    if (!selectedAlert || selectedAlert.status !== AlertStatus.Triggered) {
      return
    }
    setActionError(null)
    setAckLoading(true)
    const result = await acknowledgeAlert({ id: selectedAlert.id })
    setAckLoading(false)
    if (result.error) {
      setActionError(t('alerts.error.action'))
      return
    }
    if (result.data?.acknowledgeAlert) {
      applyAlertUpdate(result.data.acknowledgeAlert)
    }
  }

  async function handleClose(): Promise<void> {
    if (!selectedAlert || selectedAlert.status === AlertStatus.Closed) {
      return
    }
    setActionError(null)
    setCloseLoading(true)
    const result = await closeAlert({ id: selectedAlert.id })
    setCloseLoading(false)
    if (result.error) {
      setActionError(t('alerts.error.action'))
      return
    }
    if (result.data?.closeAlert) {
      applyAlertUpdate(result.data.closeAlert)
    }
  }

  async function handlePromote(): Promise<void> {
    if (!selectedAlert || selectedAlert.incidentId) {
      return
    }
    setActionError(null)
    setPromoteLoading(true)
    const result = await promoteAlertToIncident({
      input: { alertId: selectedAlert.id },
    })
    setPromoteLoading(false)
    if (result.error) {
      setActionError(t('alerts.error.action'))
      return
    }
    if (result.data?.promoteAlertToIncident) {
      applyAlertUpdate(result.data.promoteAlertToIncident)
    }
  }

  async function handleSnooze(durationMinutes: number): Promise<void> {
    if (!selectedAlert || !canSnoozeAlert(selectedAlert.status)) {
      return
    }
    setActionError(null)
    setSnoozeLoading(true)
    const result = await snoozeAlert({
      id: selectedAlert.id,
      durationMinutes,
    })
    setSnoozeLoading(false)
    if (result.error) {
      toastManager.add({
        type: 'error',
        title: t('alerts.error.action'),
        description: formatGraphQLError(result.error.message),
      })
      return
    }
    if (result.data?.snoozeAlert) {
      applyAlertUpdate(result.data.snoozeAlert)
      setSnoozeOpen(false)
      toastManager.add({
        type: 'success',
        title: t('alerts.toast.snooze.success'),
      })
    }
  }

  async function handleReEscalate(): Promise<void> {
    if (!selectedAlert || !canReEscalateAlert(selectedAlert.status)) {
      return
    }
    setActionError(null)
    setReEscalateLoading(true)
    const result = await reEscalateAlert({ id: selectedAlert.id })
    setReEscalateLoading(false)
    if (result.error) {
      toastManager.add({
        type: 'error',
        title: t('alerts.error.action'),
        description: formatGraphQLError(result.error.message),
      })
      return
    }
    if (result.data?.reEscalateAlert) {
      applyAlertUpdate(result.data.reEscalateAlert)
      toastManager.add({
        type: 'success',
        title: t('alerts.toast.reEscalate.success'),
      })
    }
  }

  return (
    <AppShell title={t('alerts.title')}>
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
        <section className="rounded-xl border border-border bg-card shadow-xs/5">
          <div className="border-b border-border p-4">
            <Tabs
              onValueChange={(value) => {
                setStatusFilter(value as StatusFilter)
              }}
              value={statusFilter}
            >
              <TabsList>
                {STATUS_FILTERS.map((filter) => (
                  <TabsTab key={filter} value={filter}>
                    {statusFilterLabel(filter)}
                    <Badge className="ms-1" size="sm" variant="secondary">
                      {filter === 'ALL' ? statusCounts.ALL : statusCounts[filter]}
                    </Badge>
                  </TabsTab>
                ))}
              </TabsList>
              {STATUS_FILTERS.map((filter) => (
                <TabsPanel key={filter} value={filter} />
              ))}
            </Tabs>
          </div>

          <ScrollArea className="h-[32rem]">
            <Table variant="card">
              <TableHeader>
                <TableRow>
                  <TableHead>{t('alerts.column.summary')}</TableHead>
                  <TableHead>{t('alerts.column.status')}</TableHead>
                  <TableHead>{t('alerts.column.priority')}</TableHead>
                  <TableHead>{t('alerts.column.events')}</TableHead>
                  <TableHead>{t('alerts.column.updated')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {fetching && visibleAlerts.length === 0 ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={5}>
                      Loading…
                    </TableCell>
                  </TableRow>
                ) : null}
                {!fetching && visibleAlerts.length === 0 ? (
                  <TableRow>
                    <TableCell className="text-muted-foreground" colSpan={5}>
                      {t('alerts.empty')}
                    </TableCell>
                  </TableRow>
                ) : null}
                {visibleAlerts.map((alert) => (
                  <TableRow
                    key={alert.id}
                    className="cursor-pointer"
                    data-state={alert.id === alertId ? 'selected' : undefined}
                    onClick={() => {
                      navigate(`/alerts/${alert.id}`)
                    }}
                  >
                    <TableCell className="max-w-xs truncate font-medium text-foreground">
                      {alert.summary}
                    </TableCell>
                    <TableCell>
                      <AlertStatusBadge status={alert.status} />
                    </TableCell>
                    <TableCell>
                      <AlertPriorityBadge priority={alert.priority} />
                    </TableCell>
                    <TableCell>
                      {alert.eventCount > 1 ? (
                        <Badge variant="outline">{alert.eventCount}</Badge>
                      ) : (
                        <span className="text-muted-foreground">1</span>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatAlertTimestamp(alert.updatedAt)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
        </section>

        <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
          {!selectedAlert ? (
            <p className="text-sm text-muted-foreground">{t('alerts.detail.select')}</p>
          ) : (
            <div className="space-y-6">
              <div className="space-y-2">
                <div className="flex flex-wrap items-center gap-2">
                  <AlertStatusBadge status={selectedAlert.status} />
                  <AlertPriorityBadge priority={selectedAlert.priority} />
                  {selectedAlert.eventCount > 1 ? (
                    <Badge variant="outline">
                      {t('alerts.column.events')}: {selectedAlert.eventCount}
                    </Badge>
                  ) : null}
                </div>
                <h2 className="text-lg font-semibold text-foreground">{selectedAlert.summary}</h2>
                {selectedAlert.description ? (
                  <p className="text-sm text-muted-foreground">{selectedAlert.description}</p>
                ) : null}
              </div>

              {actionError ? (
                <Alert variant="error">
                  <AlertCircleIcon />
                  <AlertTitle>{t('alerts.error.action')}</AlertTitle>
                  <AlertDescription>{actionError}</AlertDescription>
                </Alert>
              ) : null}

              <Group>
                <Button
                  disabled={selectedAlert.status !== AlertStatus.Triggered}
                  loading={ackLoading}
                  onClick={() => {
                    void handleAcknowledge()
                  }}
                  variant="outline"
                >
                  {ackLoading ? t('alerts.action.acknowledge.loading') : t('alerts.action.acknowledge')}
                </Button>
                <GroupSeparator />
                {canSnoozeAlert(selectedAlert.status) ? (
                  <>
                    <Dialog
                      onOpenChange={(open) => {
                        setSnoozeOpen(open)
                        if (!open) {
                          setActionError(null)
                        }
                      }}
                      open={snoozeOpen}
                    >
                      <DialogTrigger
                        disabled={snoozeLoading}
                        render={<Button type="button" variant="outline" />}
                      >
                        {snoozeLoading ? t('alerts.action.snooze.loading') : t('alerts.action.snooze')}
                      </DialogTrigger>
                      <DialogPopup>
                        <DialogHeader>
                          <DialogTitle>{t('alerts.snooze.title')}</DialogTitle>
                          <DialogDescription>{t('alerts.snooze.description')}</DialogDescription>
                        </DialogHeader>
                        <DialogPanel className="grid gap-2">
                          {SNOOZE_PRESETS.map((preset) => (
                            <Button
                              key={preset.id}
                              disabled={snoozeLoading}
                              loading={snoozeLoading}
                              onClick={() => {
                                void handleSnooze(preset.durationMinutes)
                              }}
                              type="button"
                              variant="outline"
                            >
                              {t(preset.labelKey)}
                            </Button>
                          ))}
                        </DialogPanel>
                        <DialogFooter>
                          <DialogClose render={<Button type="button" variant="ghost" />}>
                            {t('alerts.action.cancel')}
                          </DialogClose>
                        </DialogFooter>
                      </DialogPopup>
                    </Dialog>
                    <GroupSeparator />
                  </>
                ) : null}
                {canReEscalateAlert(selectedAlert.status) ? (
                  <>
                    <Button
                      disabled={reEscalateLoading}
                      loading={reEscalateLoading}
                      onClick={() => {
                        void handleReEscalate()
                      }}
                      variant="outline"
                    >
                      {reEscalateLoading
                        ? t('alerts.action.reEscalate.loading')
                        : t('alerts.action.reEscalate')}
                    </Button>
                    <GroupSeparator />
                  </>
                ) : null}
                <Button
                  disabled={Boolean(selectedAlert.incidentId) || selectedAlert.status === AlertStatus.Closed}
                  loading={promoteLoading}
                  onClick={() => {
                    void handlePromote()
                  }}
                  variant="outline"
                >
                  {promoteLoading ? t('alerts.action.promote.loading') : t('alerts.action.promote')}
                </Button>
                <GroupSeparator />
                <Button
                  disabled={selectedAlert.status === AlertStatus.Closed}
                  loading={closeLoading}
                  onClick={() => {
                    void handleClose()
                  }}
                  variant="destructive"
                >
                  {closeLoading ? t('alerts.action.close.loading') : t('alerts.action.close')}
                </Button>
              </Group>

              <div className="space-y-3">
                <h3 className="text-sm font-medium text-foreground">{t('alerts.detail.timeline')}</h3>
                <dl className="space-y-2 text-sm">
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">{t('alerts.detail.created')}</dt>
                    <dd>{formatAlertTimestamp(selectedAlert.createdAt)}</dd>
                  </div>
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">{t('alerts.detail.acknowledged')}</dt>
                    <dd>
                      {selectedAlert.acknowledgedAt
                        ? formatAlertTimestamp(selectedAlert.acknowledgedAt)
                        : '—'}
                    </dd>
                  </div>
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">{t('alerts.detail.closed')}</dt>
                    <dd>
                      {selectedAlert.closedAt
                        ? formatAlertTimestamp(selectedAlert.closedAt)
                        : '—'}
                    </dd>
                  </div>
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">{t('alerts.detail.dedupKey')}</dt>
                    <dd className="font-mono text-xs">{selectedAlert.dedupKey}</dd>
                  </div>
                  <div className="flex justify-between gap-4">
                    <dt className="text-muted-foreground">{t('alerts.detail.service')}</dt>
                    <dd>
                      <Link
                        className="text-primary hover:underline"
                        to={`/services/${selectedAlert.service.id}`}
                      >
                        {selectedAlert.service.name}
                      </Link>
                    </dd>
                  </div>
                  {selectedAlert.incidentId ? (
                    <div className="flex justify-between gap-4">
                      <dt className="text-muted-foreground">{t('alerts.detail.incident')}</dt>
                      <dd>
                        <Link
                          className="font-mono text-xs text-primary hover:underline"
                          to={`/incidents/${selectedAlert.incidentId}`}
                        >
                          {t('alerts.detail.viewIncident')}
                        </Link>
                      </dd>
                    </div>
                  ) : null}
                </dl>
              </div>

              {selectedAlert.escalationState ? (
                <div className="space-y-3">
                  <h3 className="text-sm font-medium text-foreground">{t('alerts.detail.escalation')}</h3>
                  <dl className="space-y-2 text-sm">
                    <div className="flex justify-between gap-4">
                      <dt className="text-muted-foreground">{t('alerts.detail.currentStep')}</dt>
                      <dd>{selectedAlert.escalationState.currentStep}</dd>
                    </div>
                    {selectedAlert.escalationState.nextEscalationAt ? (
                      <div className="flex justify-between gap-4">
                        <dt className="text-muted-foreground">{t('alerts.detail.nextEscalationAt')}</dt>
                        <dd>{formatAlertTimestamp(selectedAlert.escalationState.nextEscalationAt)}</dd>
                      </div>
                    ) : null}
                    {selectedAlert.escalationState.escalatedExhausted ? (
                      <div className="flex justify-between gap-4">
                        <dt className="text-muted-foreground">{t('alerts.detail.escalationExhausted')}</dt>
                        <dd>Yes</dd>
                      </div>
                    ) : null}
                  </dl>
                </div>
              ) : null}

              {selectedAlert.notificationAttempts.length > 0 ? (
                <div className="space-y-3">
                  <h3 className="text-sm font-medium text-foreground">{t('alerts.detail.notifications')}</h3>
                  <Table variant="card">
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('alerts.detail.notificationChannel')}</TableHead>
                        <TableHead>{t('alerts.detail.notificationStatus')}</TableHead>
                        <TableHead>{t('alerts.detail.notificationSent')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {selectedAlert.notificationAttempts.map((attempt) => (
                        <TableRow key={attempt.id}>
                          <TableCell>{attempt.channel}</TableCell>
                          <TableCell>{attempt.status}</TableCell>
                          <TableCell className="text-muted-foreground">
                            {formatAlertTimestamp(attempt.sentAt ?? attempt.createdAt)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : null}

              {selectedAlert.status === AlertStatus.Triggered ? (
                <Alert variant="warning">
                  <AlertCircleIcon />
                  <AlertTitle>{t('alerts.status.triggered')}</AlertTitle>
                  <AlertDescription>
                    {t('alerts.action.acknowledge')} {t('alerts.action.close')}
                  </AlertDescription>
                  <AlertAction>
                    <Button
                      loading={ackLoading}
                      onClick={() => {
                        void handleAcknowledge()
                      }}
                      size="xs"
                    >
                      {t('alerts.action.acknowledge')}
                    </Button>
                  </AlertAction>
                </Alert>
              ) : null}
            </div>
          )}
        </section>
      </div>
    </AppShell>
  )
}
