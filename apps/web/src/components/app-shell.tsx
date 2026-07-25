import type { ReactElement, ReactNode } from 'react'
import { Link } from 'react-router'

import { t } from '../lib/i18n'

type AppShellProps = {
  title: string
  children: ReactNode
}

export function AppShell({ title, children }: AppShellProps): ReactElement {
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border bg-card">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
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
              <Link className="hover:text-foreground" to="/services">
                {t('nav.services')}
              </Link>
              <Link className="hover:text-foreground" to="/integrations">
                {t('nav.integrations')}
              </Link>
              <Link className="hover:text-foreground" to="/settings">
                {t('nav.settings')}
              </Link>
            </nav>
          </div>
          <span className="text-sm text-muted-foreground">{title}</span>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-8">{children}</main>
    </div>
  )
}
