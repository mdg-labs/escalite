import { useState, type ReactElement, type ReactNode } from 'react'
import { Navigate, Route, Routes, useLocation } from 'react-router'
import {
  UserRole,
  useMeQuery,
  useMobileDevicesQuery,
  useMyOrganizationsQuery,
  useRevokeMobileDeviceMutation,
  type MobileDevicesQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  AlertDialog,
  AlertDialogClose,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogPopup,
  AlertDialogTitle,
  AlertDialogTrigger,
  Badge,
  Button,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@escalite/ui'
import { AlertTriangleIcon, CircleCheckIcon, SmartphoneIcon } from 'lucide-react'

import { AppShell } from '../components/app-shell'
import { ContactMethodsPanel } from '../components/contact-methods-panel'
import { IncidentRolesPanel } from '../components/incident-roles-panel'
import { OrgProfilePanel } from '../components/org-profile-panel'
import { NotificationRulesPanel } from '../components/notification-rules-panel'
import { SamlSettingsPanel } from '../components/saml-settings-panel'
import { ScimSettingsPanel } from '../components/scim-settings-panel'
import { SettingsSectionNav } from '../components/settings-section-nav'
import { SlackSettingsPanel } from '../components/slack-settings-panel'
import { formatDateTime, formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'
import { showMutationError } from '../lib/toast'

type MobileDeviceRow = MobileDevicesQuery['mobileDevices'][number]

function deviceStatusBadge(device: MobileDeviceRow): ReactElement {
  if (device.revokedAt) {
    return (
      <Badge variant="error">
        <AlertTriangleIcon />
        {t('settings.devices.status.revoked')}
      </Badge>
    )
  }

  return (
    <Badge variant="success">
      <CircleCheckIcon />
      {t('settings.devices.status.active')}
    </Badge>
  )
}

function deviceLabel(device: MobileDeviceRow): string {
  if (device.deviceLabel?.trim()) {
    return device.deviceLabel
  }
  if (device.platform?.trim()) {
    return device.platform
  }
  return t('settings.devices.unknownDevice')
}

function AdminOnlySection({ children }: { children: ReactNode }): ReactElement {
  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const isAdmin = meData?.me?.role === UserRole.Admin

  if (!isAdmin) {
    return <Navigate replace to="/settings/enterprise" />
  }

  return <>{children}</>
}

function SettingsIndexRedirect(): ReactElement {
  const location = useLocation()
  return <Navigate replace to={{ pathname: 'enterprise', search: location.search }} />
}

function SettingsEnterpriseSection(): ReactElement {
  return (
    <div className="space-y-6">
      <OrgProfilePanel />
      <SamlSettingsPanel />
      <ScimSettingsPanel />
      <SlackSettingsPanel />
    </div>
  )
}

function SettingsNotificationsSection(): ReactElement {
  return (
    <div className="space-y-6">
      <ContactMethodsPanel />
      <NotificationRulesPanel />
    </div>
  )
}

function SettingsDevicesSection(): ReactElement {
  const [{ data, fetching, error }, reexecuteQuery] = useMobileDevicesQuery({
    requestPolicy: 'network-only',
  })
  const [, revokeMobileDevice] = useRevokeMobileDeviceMutation()
  const [actionError, setActionError] = useState<string | null>(null)
  const [actionDeviceId, setActionDeviceId] = useState<string | null>(null)

  const devices = data?.mobileDevices ?? []

  async function handleRevoke(device: MobileDeviceRow): Promise<void> {
    setActionError(null)
    setActionDeviceId(device.id)

    const result = await revokeMobileDevice({ id: device.id })
    setActionDeviceId(null)

    if (result.error) {
      showMutationError(result.error, 'settings.devices.error.action')
      return
    }

    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex items-start gap-3">
        <SmartphoneIcon className="mt-0.5 size-5 text-muted-foreground" />
        <div>
          <h1 className="text-xl font-semibold text-foreground">{t('settings.devices.title')}</h1>
          <p className="mt-2 text-sm text-muted-foreground">{t('settings.devices.description')}</p>
        </div>
      </div>

      {(error || actionError) && (
        <Alert className="mt-6" variant="error">
          <AlertDescription>
            {actionError ?? (error ? formatGraphQLError(error.message) : null)}
          </AlertDescription>
        </Alert>
      )}

      <div className="mt-6">
        {fetching && devices.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('settings.devices.loading')}</p>
        ) : devices.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('settings.devices.empty')}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('settings.devices.column.device')}</TableHead>
                <TableHead>{t('settings.devices.column.token')}</TableHead>
                <TableHead>{t('settings.devices.column.lastSeen')}</TableHead>
                <TableHead>{t('settings.devices.column.status')}</TableHead>
                <TableHead className="text-right">{t('settings.devices.column.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {devices.map((device) => (
                <TableRow key={device.id}>
                  <TableCell>
                    <div className="font-medium text-foreground">{deviceLabel(device)}</div>
                    {device.platform ? (
                      <div className="text-xs text-muted-foreground">{device.platform}</div>
                    ) : null}
                  </TableCell>
                  <TableCell className="font-mono text-sm">{device.pushTokenPrefix}…</TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {formatDateTime(device.lastRegisteredAt)}
                  </TableCell>
                  <TableCell>{deviceStatusBadge(device)}</TableCell>
                  <TableCell className="text-right">
                    {!device.revokedAt ? (
                      <AlertDialog>
                        <AlertDialogTrigger
                          render={
                            <Button
                              disabled={actionDeviceId === device.id}
                              size="sm"
                              type="button"
                              variant="destructive-outline"
                            />
                          }
                        >
                          {t('settings.devices.action.revoke')}
                        </AlertDialogTrigger>
                        <AlertDialogPopup>
                          <AlertDialogHeader>
                            <AlertDialogTitle>{t('settings.devices.revoke.title')}</AlertDialogTitle>
                            <AlertDialogDescription>
                              {t('settings.devices.revoke.description', {
                                device: deviceLabel(device),
                              })}
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogClose render={<Button type="button" variant="ghost" />}>
                              {t('settings.devices.action.cancel')}
                            </AlertDialogClose>
                            <AlertDialogClose
                              onClick={() => void handleRevoke(device)}
                              render={<Button type="button" variant="destructive" />}
                            >
                              {t('settings.devices.action.revoke')}
                            </AlertDialogClose>
                          </AlertDialogFooter>
                        </AlertDialogPopup>
                      </AlertDialog>
                    ) : (
                      <span className="text-sm text-muted-foreground">—</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </section>
  )
}

function SettingsRolesSection(): ReactElement {
  return <IncidentRolesPanel />
}

export function SettingsPage(): ReactElement {
  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const [{ data: orgData }] = useMyOrganizationsQuery({ requestPolicy: 'cache-first' })

  const isAdmin = meData?.me?.role === UserRole.Admin
  const activeOrganizationId = meData?.me?.organizationId ?? ''
  const activeOrganizationName =
    orgData?.myOrganizations.find(
      (membership) => membership.organization.id === activeOrganizationId,
    )?.organization.name ?? ''

  return (
    <AppShell title={t('settings.title')}>
      <div className="space-y-6">
        {activeOrganizationName ? (
          <p className="text-sm text-muted-foreground">
            {t('settings.org.description', { organization: activeOrganizationName })}
          </p>
        ) : null}

        <div className="flex flex-col gap-6 lg:flex-row lg:gap-8">
          <aside className="shrink-0 lg:w-48">
            <SettingsSectionNav isAdmin={isAdmin} />
          </aside>

          <div className="min-w-0 flex-1">
            <Routes>
              <Route element={<SettingsIndexRedirect />} index />
              <Route element={<SettingsEnterpriseSection />} path="enterprise" />
              <Route element={<SettingsNotificationsSection />} path="notifications" />
              <Route element={<SettingsDevicesSection />} path="devices" />
              <Route
                element={
                  <AdminOnlySection>
                    <SettingsRolesSection />
                  </AdminOnlySection>
                }
                path="roles"
              />
              <Route element={<Navigate replace to="roles" />} path="incident-roles" />
              <Route element={<Navigate replace to="enterprise" />} path="*" />
            </Routes>
          </div>
        </div>
      </div>
    </AppShell>
  )
}
