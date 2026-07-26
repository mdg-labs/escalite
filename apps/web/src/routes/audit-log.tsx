import type { ReactElement } from 'react'
import { useMemo, useState } from 'react'
import { Navigate } from 'react-router'
import {
  UserRole,
  useAuditEventActionsQuery,
  useAuditEventsQuery,
  useMeQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  Button,
  Group,
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
import { AlertTriangleIcon, DownloadIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import {
  AUDIT_LOG_PAGE_SIZE,
  dateInputToEndOfDayUTC,
  dateInputToStartOfDayUTC,
  downloadAuditLogCsv,
  formatAuditActionLabel,
} from '../lib/audit-log'
import { formatDateTime, formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

const ALL_ACTIONS_VALUE = '__all__'

export function AuditLogPage(): ReactElement {
  const [{ data: meData, fetching: meFetching }] = useMeQuery({ requestPolicy: 'cache-first' })
  const isAdmin = meData?.me?.role === UserRole.Admin

  const [action, setAction] = useState(ALL_ACTIONS_VALUE)
  const [fromDate, setFromDate] = useState('')
  const [toDate, setToDate] = useState('')
  const [page, setPage] = useState(1)
  const [exporting, setExporting] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)

  const actionFilter = action === ALL_ACTIONS_VALUE ? undefined : action
  const from = dateInputToStartOfDayUTC(fromDate)
  const to = dateInputToEndOfDayUTC(toDate)
  const offset = (page - 1) * AUDIT_LOG_PAGE_SIZE

  const [{ data: actionsData }] = useAuditEventActionsQuery({
    pause: !isAdmin,
    requestPolicy: 'cache-first',
  })

  const [{ data, fetching, error }] = useAuditEventsQuery({
    pause: !isAdmin,
    variables: {
      action: actionFilter,
      from,
      to,
      limit: AUDIT_LOG_PAGE_SIZE,
      offset,
    },
    requestPolicy: 'network-only',
  })

  const actions = useMemo(() => actionsData?.auditEventActions ?? [], [actionsData?.auditEventActions])
  const items = data?.auditEvents.items ?? []
  const totalCount = data?.auditEvents.totalCount ?? 0
  const totalPages = Math.max(1, Math.ceil(totalCount / AUDIT_LOG_PAGE_SIZE))
  const showingFrom = totalCount === 0 ? 0 : offset + 1
  const showingTo = Math.min(offset + items.length, totalCount)

  if (!meFetching && !isAdmin) {
    return <Navigate replace to="/dashboard" />
  }

  function resetPage(): void {
    setPage(1)
  }

  async function handleExport(): Promise<void> {
    setActionError(null)
    setExporting(true)
    try {
      await downloadAuditLogCsv({
        action: actionFilter,
        from: fromDate.trim() || undefined,
        to: toDate.trim() || undefined,
      })
    } catch {
      setActionError(t('auditLog.error.export'))
    } finally {
      setExporting(false)
    }
  }

  return (
    <AppShell title={t('auditLog.title')}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold text-foreground">{t('auditLog.title')}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{t('auditLog.description')}</p>
          </div>
          <Button disabled={exporting} onClick={() => void handleExport()} type="button">
            <DownloadIcon />
            {exporting ? t('auditLog.action.exporting') : t('auditLog.action.export')}
          </Button>
        </div>

        <div className="mt-6 grid gap-4 md:grid-cols-3">
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground" htmlFor="audit-action">
              {t('auditLog.filter.action')}
            </label>
            <Select
              onValueChange={(value) => {
                setAction(value ?? ALL_ACTIONS_VALUE)
                resetPage()
              }}
              value={action}
            >
              <SelectTrigger id="audit-action">
                <SelectValue placeholder={t('auditLog.filter.actionPlaceholder')} />
              </SelectTrigger>
              <SelectPopup>
                <SelectItem value={ALL_ACTIONS_VALUE}>{t('auditLog.filter.allActions')}</SelectItem>
                {actions.map((actionOption) => (
                  <SelectItem key={actionOption} value={actionOption}>
                    {formatAuditActionLabel(actionOption)}
                  </SelectItem>
                ))}
              </SelectPopup>
            </Select>
          </div>

          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground" htmlFor="audit-from">
              {t('auditLog.filter.from')}
            </label>
            <Input
              id="audit-from"
              nativeInput
              onChange={(event) => {
                setFromDate(event.target.value)
                resetPage()
              }}
              type="date"
              value={fromDate}
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground" htmlFor="audit-to">
              {t('auditLog.filter.to')}
            </label>
            <Input
              id="audit-to"
              nativeInput
              onChange={(event) => {
                setToDate(event.target.value)
                resetPage()
              }}
              type="date"
              value={toDate}
            />
          </div>
        </div>

        {(error || actionError) && (
          <Alert className="mt-6" variant="error">
            <AlertTriangleIcon />
            <AlertDescription>
              {actionError ?? (error ? formatGraphQLError(error.message) : null)}
            </AlertDescription>
          </Alert>
        )}

        <div className="mt-6">
          {fetching && items.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('auditLog.loading')}</p>
          ) : items.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('auditLog.empty')}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('auditLog.column.timestamp')}</TableHead>
                  <TableHead>{t('auditLog.column.action')}</TableHead>
                  <TableHead>{t('auditLog.column.actor')}</TableHead>
                  <TableHead>{t('auditLog.column.target')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((event) => (
                  <TableRow key={event.id}>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDateTime(event.createdAt)}
                    </TableCell>
                    <TableCell className="font-mono text-sm">{event.action}</TableCell>
                    <TableCell className="text-sm">
                      {event.actorEmail ?? event.actorId ?? t('auditLog.actor.system')}
                    </TableCell>
                    <TableCell className="font-mono text-sm text-muted-foreground">
                      {event.targetType
                        ? `${event.targetType}${event.targetId ? `:${event.targetId}` : ''}`
                        : '—'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>

        {totalCount > 0 ? (
          <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm text-muted-foreground">
              {t('auditLog.pagination.summary', {
                from: String(showingFrom),
                to: String(showingTo),
                total: String(totalCount),
              })}
            </p>
            <Group>
              <Button
                disabled={page <= 1 || fetching}
                onClick={() => setPage((current) => Math.max(1, current - 1))}
                type="button"
                variant="outline"
              >
                {t('auditLog.pagination.previous')}
              </Button>
              <Button
                disabled={page >= totalPages || fetching}
                onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
                type="button"
                variant="outline"
              >
                {t('auditLog.pagination.next')}
              </Button>
            </Group>
          </div>
        ) : null}
      </section>
    </AppShell>
  )
}
