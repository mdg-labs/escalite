import type { ReactElement } from 'react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
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
} from '@escalite/ts-types'
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
  Badge,
  Button,
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
} from '@escalite/ui'
import { AlertCircleIcon } from 'lucide-react'

import { AlertPriorityBadge } from '../components/alert-priority-badge'
import { AlertStatusBadge } from '../components/alert-status-badge'
import { AppShell } from '../components/app-shell'
import {
  formatAlertTimestamp,
  mergeAlertUpdate,
  sortAlertsByUpdatedAt,
} from '../lib/alerts'
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
  const [ackLoading, setAckLoading] = useState(false)
  const [closeLoading, setCloseLoading] = useState(false)
  const [promoteLoading, setPromoteLoading] = useState(false)

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
                    <dt className="text-muted-foreground">{t('alerts.detail.serviceId')}</dt>
                    <dd className="font-mono text-xs">{selectedAlert.serviceId}</dd>
                  </div>
                  {selectedAlert.incidentId ? (
                    <div className="flex justify-between gap-4">
                      <dt className="text-muted-foreground">{t('alerts.detail.incident')}</dt>
                      <dd className="font-mono text-xs">{selectedAlert.incidentId}</dd>
                    </div>
                  ) : null}
                </dl>
              </div>

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
