import { useEffect, useMemo, useState, type ReactElement } from 'react'
import { Link, useLocation } from 'react-router'
import { UserRole, useMeQuery } from '@escalite/ts-types'
import {
  Button,
  Drawer,
  DrawerPopup,
  DrawerTitle,
  DrawerTrigger,
  ScrollArea,
  cn,
} from '@escalite/ui'
import { MenuIcon } from 'lucide-react'

import { type MessageKey, t } from '../lib/i18n'
import { OrgSwitcher } from './org-switcher'

export type NavSection = 'operations' | 'configuration' | 'admin'

export type SidebarNavItem = {
  path: string
  labelKey: MessageKey
  adminOnly?: boolean
}

export type SidebarNavGroup = {
  section: NavSection
  items: SidebarNavItem[]
}

/** Section labels — move to i18n in fe-shell-nav-config-routes. */
const SECTION_LABELS: Record<NavSection, string> = {
  operations: 'Operations',
  configuration: 'Configuration',
  admin: 'Admin',
}

const SIDEBAR_NAV_CONFIG: SidebarNavGroup[] = [
  {
    section: 'operations',
    items: [
      { path: '/dashboard', labelKey: 'nav.dashboard' },
      { path: '/alerts', labelKey: 'nav.alerts' },
      { path: '/incidents', labelKey: 'nav.incidents' },
    ],
  },
  {
    section: 'configuration',
    items: [
      { path: '/services', labelKey: 'nav.services' },
      { path: '/integrations', labelKey: 'nav.integrations' },
      { path: '/analytics', labelKey: 'nav.analytics' },
      { path: '/settings', labelKey: 'nav.settings' },
    ],
  },
  {
    section: 'admin',
    items: [{ path: '/audit-log', labelKey: 'nav.auditLog', adminOnly: true }],
  },
]

function isNavItemActive(pathname: string, path: string): boolean {
  if (path === '/dashboard') {
    return pathname === '/dashboard'
  }

  return pathname === path || pathname.startsWith(`${path}/`)
}

type SidebarNavProps = {
  /** EL-166: mobile drawer will pass onNavigate to close drawer after route change. */
  onNavigate?: () => void
  /** Hide footer org switcher when shell header already shows it (mobile drawer). */
  hideOrgSwitcher?: boolean
  className?: string
}

export function SidebarNav({
  onNavigate,
  hideOrgSwitcher = false,
  className,
}: SidebarNavProps): ReactElement {
  const location = useLocation()
  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const isAdmin = meData?.me?.role === UserRole.Admin

  const visibleGroups = useMemo(
    () =>
      SIDEBAR_NAV_CONFIG.map((group) => ({
        ...group,
        items: group.items.filter((item) => !item.adminOnly || isAdmin),
      })).filter((group) => group.items.length > 0),
    [isAdmin],
  )

  return (
    <div className={cn('flex h-full min-h-0 flex-col', className)} data-slot="sidebar-nav">
      <div className="flex h-14 shrink-0 items-center border-b border-sidebar-border px-4">
        <Link
          className="text-sm font-semibold tracking-wide text-sidebar-foreground"
          onClick={onNavigate}
          to="/dashboard"
        >
          Escalite
        </Link>
      </div>

      <ScrollArea className="min-h-0 flex-1" fill>
        <nav aria-label="Main" className="flex flex-col gap-4 p-3">
          {visibleGroups.map((group) => (
            <div key={group.section}>
              <p className="px-2 text-xs font-medium text-sidebar-foreground/70">
                {SECTION_LABELS[group.section]}
              </p>
              <ul className="mt-1 flex flex-col gap-0.5">
                {group.items.map((item) => {
                  const active = isNavItemActive(location.pathname, item.path)

                  return (
                    <li key={item.path}>
                      <Link
                        aria-current={active ? 'page' : undefined}
                        className={cn(
                          'block rounded-md px-2 py-1.5 text-sm transition-colors',
                          active
                            ? 'bg-sidebar-accent font-medium text-sidebar-accent-foreground'
                            : 'text-sidebar-foreground hover:bg-sidebar-accent/60',
                        )}
                        onClick={onNavigate}
                        to={item.path}
                      >
                        {t(item.labelKey)}
                      </Link>
                    </li>
                  )
                })}
              </ul>
            </div>
          ))}
        </nav>
      </ScrollArea>

      {!hideOrgSwitcher ? (
        <footer className="shrink-0 border-t border-sidebar-border p-3">
          <div className="[&>div]:items-stretch [&>div]:gap-2">
            <OrgSwitcher />
          </div>
        </footer>
      ) : null}
    </div>
  )
}

/** Mobile hamburger → left drawer (p-drawer-11) for viewports below md. */
export function MobileSidebarDrawer(): ReactElement {
  const [open, setOpen] = useState(false)
  const location = useLocation()

  useEffect(() => {
    setOpen(false)
  }, [location.pathname])

  const closeDrawer = (): void => {
    setOpen(false)
  }

  return (
    <Drawer onOpenChange={setOpen} open={open} position="left">
      <DrawerTrigger
        aria-label={t('nav.openMenu')}
        render={<Button size="icon" type="button" variant="ghost" />}
      >
        <MenuIcon />
      </DrawerTrigger>
      <DrawerPopup
        className="h-full max-h-none w-56 max-w-[85vw] border-sidebar-border bg-sidebar p-0 text-sidebar-foreground shadow-none"
        position="left"
        variant="straight"
      >
        <DrawerTitle className="sr-only">{t('nav.menu')}</DrawerTitle>
        <SidebarNav hideOrgSwitcher className="h-full" onNavigate={closeDrawer} />
      </DrawerPopup>
    </Drawer>
  )
}
