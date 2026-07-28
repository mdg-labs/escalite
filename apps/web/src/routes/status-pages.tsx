import type { ReactElement } from 'react'
import { Navigate } from 'react-router'
import { UserRole, useMeQuery } from '@escalite/ts-types'
import { Tabs, TabsList, TabsPanel, TabsTab } from '@escalite/ui'

import { AppShell } from '../components/app-shell'
import { StatusPageComponentsPanel } from '../components/status-page-components-panel'
import { StatusPageIncidentsPanel } from '../components/status-page-incidents-panel'
import { StatusPageSettingsPanel } from '../components/status-page-settings-panel'
import { StatusPageSubscriptionsPanel } from '../components/status-page-subscriptions-panel'
import { t } from '../lib/i18n'

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
            <StatusPageIncidentsPanel />
          </TabsPanel>

          <TabsPanel className="mt-6" value="subscriptions">
            <StatusPageSubscriptionsPanel />
          </TabsPanel>
        </Tabs>
      </div>
    </AppShell>
  )
}
