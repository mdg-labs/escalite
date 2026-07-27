import type { ReactElement, ReactNode } from 'react'
import { Navigate, Route, Routes, useSearchParams } from 'react-router'
import { useMeQuery } from '@escalite/ts-types'

import { AnalyticsPage } from './routes/analytics'
import { AuditLogPage } from './routes/audit-log'
import { AlertsPage } from './routes/alerts'
import { DashboardPage } from './routes/dashboard'
import { EscalationPolicyPage } from './routes/escalation-policy'
import { IncidentsPage } from './routes/incidents'
import { IntegrationsPage } from './routes/integrations'
import { LoginPage } from './routes/login'
import { LoginMobilePage } from './routes/login-mobile'
import { SchedulePage } from './routes/schedule'
import { ServicePage } from './routes/service'
import { ServicesPage } from './routes/services'
import { SettingsPage } from './routes/settings'
import { TeamsPage } from './routes/teams'
import { NotFoundPage } from './routes/not-found'
import { SetupPage } from './routes/setup'

function AuthLoading(): ReactElement {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background text-muted-foreground">
      Loading…
    </div>
  )
}

function ProtectedRoute({ children }: { children: ReactNode }): ReactElement {
  const [{ data, fetching }] = useMeQuery({ requestPolicy: 'network-only' })

  if (fetching) {
    return <AuthLoading />
  }

  if (!data?.me) {
    return <Navigate replace to="/login" />
  }

  return <>{children}</>
}

function GuestRoute({ children }: { children: ReactNode }): ReactElement {
  const [{ data, fetching }] = useMeQuery({ requestPolicy: 'network-only' })
  const [searchParams] = useSearchParams()
  const redirectTo = searchParams.get('redirect') ?? '/dashboard'

  if (fetching) {
    return <AuthLoading />
  }

  if (data?.me) {
    return <Navigate replace to={redirectTo} />
  }

  return <>{children}</>
}

export function App(): ReactElement {
  return (
    <Routes>
      <Route
        path="/login"
        element={
          <GuestRoute>
            <LoginPage />
          </GuestRoute>
        }
      />
      <Route path="/login/mobile" element={<LoginMobilePage />} />
      <Route
        path="/setup"
        element={
          <GuestRoute>
            <SetupPage />
          </GuestRoute>
        }
      />
      <Route
        path="/alerts/:alertId?"
        element={
          <ProtectedRoute>
            <AlertsPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/incidents/:incidentId?"
        element={
          <ProtectedRoute>
            <IncidentsPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/integrations"
        element={
          <ProtectedRoute>
            <IntegrationsPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/teams"
        element={
          <ProtectedRoute>
            <TeamsPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/services"
        element={
          <ProtectedRoute>
            <ServicesPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/services/:serviceId"
        element={
          <ProtectedRoute>
            <ServicePage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/schedules/:scheduleId"
        element={
          <ProtectedRoute>
            <SchedulePage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/services/:serviceId/escalation-policies/:policyId"
        element={
          <ProtectedRoute>
            <EscalationPolicyPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/settings"
        element={
          <ProtectedRoute>
            <SettingsPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/analytics"
        element={
          <ProtectedRoute>
            <AnalyticsPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/audit-log"
        element={
          <ProtectedRoute>
            <AuditLogPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/dashboard"
        element={
          <ProtectedRoute>
            <DashboardPage />
          </ProtectedRoute>
        }
      />
      <Route path="/" element={<Navigate replace to="/dashboard" />} />
      <Route
        path="*"
        element={
          <ProtectedRoute>
            <NotFoundPage />
          </ProtectedRoute>
        }
      />
    </Routes>
  )
}
