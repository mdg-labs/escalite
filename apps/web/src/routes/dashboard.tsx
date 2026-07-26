import type { ReactElement } from 'react'
import { Link } from 'react-router'
import { useMeQuery } from '@escalite/ts-types'

import { AppShell } from '../components/app-shell'
import { t } from '../lib/i18n'

export function DashboardPage(): ReactElement {
  const [{ data }] = useMeQuery({ requestPolicy: 'cache-first' })
  const userEmail = data?.me?.email ?? ''

  return (
    <AppShell title={t('nav.dashboard')}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <h1 className="text-xl font-semibold text-foreground">Welcome back</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Your on-call workspace shell is ready. Incident and alerting features arrive in later
          phases.
        </p>
        {userEmail ? (
          <p className="mt-2 text-sm text-muted-foreground">
            Signed in as <span className="font-medium text-foreground">{userEmail}</span>
          </p>
        ) : null}
        <p className="mt-4 flex flex-wrap gap-4 text-sm">
          <Link className="font-medium text-foreground underline-offset-4 hover:underline" to="/alerts">
            View alerts
          </Link>
          <Link className="font-medium text-foreground underline-offset-4 hover:underline" to="/services">
            Configure services
          </Link>
          <Link className="font-medium text-foreground underline-offset-4 hover:underline" to="/integrations">
            Add an integration
          </Link>
        </p>
      </section>
    </AppShell>
  )
}
