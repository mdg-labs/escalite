import { useMemo, useState, useEffect, type ReactElement } from 'react'
import { Link, useLocation } from 'react-router'
import { UserRole, useMeQuery } from '@escalite/ts-types'
import {
  Button,
  Drawer,
  DrawerPopup,
  DrawerTitle,
  DrawerTrigger,
  EscaliteLogo,
  ScrollArea,
  cn,
} from '@escalite/ui'
import { MenuIcon } from 'lucide-react'

import { t } from '../lib/i18n'
import {
  SECTION_LABEL_KEYS,
  filterNavGroupsForRole,
  isNavItemActive,
} from '../lib/nav-config'
import { OrgSwitcher } from './org-switcher'
import { UserMenu } from './user-menu'

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

  const visibleGroups = useMemo(() => filterNavGroupsForRole(isAdmin), [isAdmin])

  return (
    <div className={cn('flex h-full min-h-0 flex-col', className)} data-slot="sidebar-nav">
      <div className="flex h-14 shrink-0 items-center border-b border-sidebar-border px-4">
        <Link
          className="flex items-center gap-2 text-sm font-semibold tracking-wide text-sidebar-foreground"
          onClick={onNavigate}
          to="/dashboard"
        >
          <EscaliteLogo variant="mark" />
          <span className="text-brand">Escalite</span>
        </Link>
      </div>

      <ScrollArea className="min-h-0 flex-1" fill>
        <nav aria-label="Main" className="flex flex-col gap-4 p-3">
          {visibleGroups.map((group) => (
            <div key={group.section}>
              <p className="px-2 text-xs font-medium text-sidebar-foreground/70">
                {t(SECTION_LABEL_KEYS[group.section])}
              </p>
              <ul className="mt-1 flex flex-col gap-0.5">
                {group.items.map((item) => {
                  const active = isNavItemActive(location.pathname, item)

                  return (
                    <li key={item.path}>
                      <Link
                        aria-current={active ? 'page' : undefined}
                        className={cn(
                          'block rounded-md border-s-2 px-2 py-1.5 text-sm transition-colors',
                          active
                            ? 'border-brand bg-brand-muted font-medium text-sidebar-accent-foreground'
                            : 'border-transparent text-sidebar-foreground hover:bg-sidebar-accent/60',
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

      <footer className="shrink-0 space-y-3 border-t border-sidebar-border p-3">
        {!hideOrgSwitcher ? (
          <div className="[&>div]:items-stretch [&>div]:gap-2">
            <OrgSwitcher />
          </div>
        ) : null}
        <UserMenu onNavigate={onNavigate} />
      </footer>
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
