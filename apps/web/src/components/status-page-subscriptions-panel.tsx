import { useMemo, type ReactElement } from 'react'
import { useStatusPageQuery, type StatusPageQuery } from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  Badge,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@escalite/ui'
import { MailIcon, UsersIcon } from 'lucide-react'

import { formatDateTime, formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

type SubscriptionRow = NonNullable<StatusPageQuery['statusPage']>['subscriptions'][number]

export function StatusPageSubscriptionsPanel(): ReactElement {
  const [{ data, fetching, error }] = useStatusPageQuery({
    requestPolicy: 'network-only',
  })

  const statusPage = data?.statusPage
  const subscriptions = useMemo(() => {
    const rows = statusPage?.subscriptions ?? []
    return [...rows].sort(
      (left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime(),
    )
  }, [statusPage?.subscriptions])

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex items-start gap-3">
          <MailIcon className="mt-0.5 size-5 text-muted-foreground" />
          <div>
            <h2 className="text-xl font-semibold text-foreground">
              {t('statusPages.subscriptions.title')}
            </h2>
            <p className="mt-2 text-sm text-muted-foreground">
              {t('statusPages.subscriptions.description')}
            </p>
          </div>
        </div>

        {statusPage ? (
          <Badge variant="secondary">
            <UsersIcon />
            {t('statusPages.subscriptions.count', { count: String(subscriptions.length) })}
          </Badge>
        ) : null}
      </div>

      {error ? (
        <Alert className="mt-6" variant="error">
          <AlertDescription>{formatGraphQLError(error.message)}</AlertDescription>
        </Alert>
      ) : null}

      <div className="mt-6">
        {fetching && !statusPage ? (
          <p className="text-sm text-muted-foreground">{t('statusPages.subscriptions.loading')}</p>
        ) : !statusPage ? (
          <p className="text-sm text-muted-foreground">{t('statusPages.subscriptions.notConfigured')}</p>
        ) : subscriptions.length === 0 ? (
          <div className="flex items-start gap-3 rounded-lg border border-dashed border-border p-4">
            <UsersIcon className="mt-0.5 size-4 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">{t('statusPages.subscriptions.empty')}</p>
          </div>
        ) : (
          <Table variant="card">
            <TableHeader>
              <TableRow>
                <TableHead>{t('statusPages.subscriptions.column.email')}</TableHead>
                <TableHead>{t('statusPages.subscriptions.column.subscribed')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {subscriptions.map((subscription: SubscriptionRow) => (
                <TableRow key={subscription.id}>
                  <TableCell className="font-medium text-foreground">{subscription.email}</TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDateTime(subscription.createdAt)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </section>
  )
}
