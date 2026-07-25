import type { ReactElement } from 'react'
import { Link } from 'react-router'

import { IntegrationPicker } from '../components/integration-picker'

export function IntegrationsPage(): ReactElement {
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border bg-card">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <Link className="text-sm font-semibold tracking-wide text-foreground" to="/dashboard">
              Escalite
            </Link>
            <span className="text-sm text-muted-foreground">Integrations</span>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-2xl px-4 py-8">
        <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
          <h1 className="text-xl font-semibold text-foreground">Add integration</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Choose a preset to create an inbound integration key with pre-filled field mapping.
          </p>
          <div className="mt-6">
            <IntegrationPicker />
          </div>
        </section>
      </main>
    </div>
  )
}
