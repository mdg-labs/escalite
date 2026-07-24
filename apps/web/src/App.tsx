import type { ReactElement, ReactNode } from 'react'
import { Navigate, Route, Routes } from 'react-router'
import { useMeQuery } from '@escalite/ts-types'

import { DashboardPage } from './routes/dashboard'
import { LoginPage } from './routes/login'
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

  if (fetching) {
    return <AuthLoading />
  }

  if (data?.me) {
    return <Navigate replace to="/dashboard" />
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
      <Route
        path="/setup"
        element={
          <GuestRoute>
            <SetupPage />
          </GuestRoute>
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
      <Route path="*" element={<Navigate replace to="/dashboard" />} />
    </Routes>
  )
}
