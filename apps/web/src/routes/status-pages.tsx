import type { ReactElement } from 'react'
import { Navigate } from 'react-router'
import { UserRole, useMeQuery } from '@escalite/ts-types'
import { Tabs, TabsList, TabsPanel, TabsTab } from '@escalite/ui'

import { AppShell } from '../components/app-shell'
import { StatusPageComponentsPanel } from '../components/status-page-components-panel'
import { StatusPageSettingsPanel } from '../components/status-page-settings-panel'
import { t, type MessageKey } from '../lib/i18n'

type StatusPagePlaceholderKey = Extract<
  MessageKey,
  'statusPages.placeholder.incidents' | 'statusPages.placeholder.subscriptions'
>

function StatusPagePlaceholderPanel({ messageKey }: { messageKey: StatusPagePlaceholderKey }): ReactElement {
  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <p className="text-sm text-muted-foreground">{t(messageKey)}</p>
    </section>
  )
}

export function StatusPagesPage(): ReactElement {
  const [{ data: meData, fetching: meFetching }] = useMeQuery({ requestPolicy: 'cache-first' })
  const isAdmin = meData?.me?.role === UserRole.Admin

  if (!meFetching && !isAdmin) {
    return <Navigate replace to="/dashboard" />
  }

  return (
    <AppShell title={t('statusPages.title')}>
      <div className="space-y-6">
        <p className="text-sm text-muted-foreground">{t('statusPages.description')}</p>

        <Tabs defaultValue="settings">
          <TabsList variant="underline">
            <TabsTab value="settings">{t('statusPages.tabs.settings')}</TabsTab>
            <TabsTab value="components">{t('statusPages.tabs.components')}</TabsTab>
            <TabsTab value="incidents">{t('statusPages.tabs.incidents')}</TabsTab>
            <TabsTab value="subscriptions">{t('statusPages.tabs.subscriptions')}</TabsTab>
          </TabsList>

          <TabsPanel className="mt-6" value="settings">
            <StatusPageSettingsPanel />
          </TabsPanel>

          <TabsPanel className="mt-6" value="components">
            <StatusPageComponentsPanel />
          </TabsPanel>

          <TabsPanel className="mt-6" value="incidents">
            <StatusPagePlaceholderPanel messageKey="statusPages.placeholder.incidents" />
          </TabsPanel>

          <TabsPanel className="mt-6" value="subscriptions">
            <StatusPagePlaceholderPanel messageKey="statusPages.placeholder.subscriptions" />
          </TabsPanel>
        </Tabs>
      </div>
    </AppShell>
  )
}
