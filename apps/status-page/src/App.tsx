import type { ReactElement } from 'react'
import { Navigate, Route, Routes } from 'react-router'

import { t } from './lib/i18n'
import { StatusPageRoute } from './routes/status-page'

function LandingPage(): ReactElement {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <div className="max-w-lg space-y-2 text-center">
        <h1 className="text-2xl font-semibold text-foreground">{t('statusPage.landing.title')}</h1>
        <p className="text-sm text-muted-foreground">{t('statusPage.landing.description')}</p>
      </div>
    </div>
  )
}

export function App(): ReactElement {
  return (
    <Routes>
      <Route element={<LandingPage />} path="/" />
      <Route element={<StatusPageRoute />} path="/:slug" />
      <Route element={<Navigate replace to="/" />} path="*" />
    </Routes>
  )
}
