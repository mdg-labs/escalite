import type { ReactElement } from 'react'
import { Link } from 'react-router'
import { Button } from '@escalite/ui'

import { AppShell } from '../components/app-shell'
import { t } from '../lib/i18n'

export function NotFoundPage(): ReactElement {
  return (
    <AppShell title={t('notFound.title')}>
      <section className="mx-auto flex max-w-lg flex-col items-center py-16 text-center">
        <p className="text-6xl font-semibold tabular-nums text-muted-foreground">404</p>
        <h1 className="mt-4 text-xl font-semibold text-foreground">{t('notFound.heading')}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t('notFound.description')}</p>
        <Button className="mt-6" render={<Link to="/dashboard" />}>
          {t('notFound.action')}
        </Button>
      </section>
    </AppShell>
  )
}
