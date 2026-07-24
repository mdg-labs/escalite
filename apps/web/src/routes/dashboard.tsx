import type { ReactElement } from 'react'
import { useMeQuery } from '@escalite/ts-types'

export function DashboardPage(): ReactElement {
  const [{ data }] = useMeQuery({ requestPolicy: 'cache-first' })
  const userEmail = data?.me?.email ?? ''

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border bg-card">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <span className="text-sm font-semibold tracking-wide text-foreground">Escalite</span>
            <span className="text-sm text-muted-foreground">Dashboard</span>
          </div>
          <p className="text-sm text-foreground">
            Signed in as <span className="font-medium">{userEmail}</span>
          </p>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-8">
        <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
          <h1 className="text-xl font-semibold text-foreground">Welcome back</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Your on-call workspace shell is ready. Incident and alerting features arrive in later
            phases.
          </p>
        </section>
      </main>
    </div>
  )
}
