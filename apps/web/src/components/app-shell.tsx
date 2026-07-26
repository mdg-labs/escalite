import type { ReactElement, ReactNode } from 'react'
import { Link } from 'react-router'
import { UserRole, useMeQuery } from '@escalite/ts-types'

import { t } from '../lib/i18n'
import { OrgSwitcher } from './org-switcher'

type AppShellProps = {
  title: string
  children: ReactNode
}

export function AppShell({ title, children }: AppShellProps): ReactElement {
  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const isAdmin = meData?.me?.role === UserRole.Admin

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border bg-card">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between gap-4 px-4">
          <div className="flex items-center gap-4">
            <Link className="text-sm font-semibold tracking-wide text-foreground" to="/dashboard">
              Escalite
            </Link>
            <nav className="flex items-center gap-3 text-sm text-muted-foreground">
              <Link className="hover:text-foreground" to="/dashboard">
                {t('nav.dashboard')}
              </Link>
              <Link className="hover:text-foreground" to="/alerts">
                {t('nav.alerts')}
              </Link>
              <Link className="hover:text-foreground" to="/incidents">
                {t('nav.incidents')}
              </Link>
              <Link className="hover:text-foreground" to="/services">
                {t('nav.services')}
              </Link>
              <Link className="hover:text-foreground" to="/integrations">
                {t('nav.integrations')}
              </Link>
              <Link className="hover:text-foreground" to="/analytics">
                {t('nav.analytics')}
              </Link>
              {isAdmin ? (
                <Link className="hover:text-foreground" to="/audit-log">
                  {t('nav.auditLog')}
                </Link>
              ) : null}
              <Link className="hover:text-foreground" to="/settings">
                {t('nav.settings')}
              </Link>
            </nav>
          </div>
          <div className="flex items-center gap-4">
            <span className="hidden text-sm text-muted-foreground sm:inline">{title}</span>
            <OrgSwitcher />
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-8">{children}</main>
    </div>
  )
}
