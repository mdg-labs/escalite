import type { ReactElement } from 'react'

import { AppShell } from '../components/app-shell'
import { IntegrationKeysPanel } from '../components/integration-keys-panel'
import { t } from '../lib/i18n'

export function IntegrationsPage(): ReactElement {
  return (
    <AppShell title={t('nav.integrations')}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <h1 className="text-xl font-semibold text-foreground">Integration keys</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Create, rotate, and revoke inbound integration keys. Webhook URLs and full tokens are
          shown once at creation or rotation; only the prefix remains visible afterward.
        </p>
        <div className="mt-6">
          <IntegrationKeysPanel />
        </div>
      </section>
    </AppShell>
  )
}
