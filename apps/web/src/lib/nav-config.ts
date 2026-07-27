import type { MessageKey } from './i18n'

export type NavSection = 'operations' | 'configuration' | 'admin'

export type NavItem = {
  path: string
  labelKey: MessageKey
  adminOnly?: boolean
}

export type NavGroup = {
  section: NavSection
  items: NavItem[]
}

export const SECTION_LABEL_KEYS: Record<NavSection, MessageKey> = {
  operations: 'nav.section.operations',
  configuration: 'nav.section.configuration',
  admin: 'nav.section.admin',
}

export const SIDEBAR_NAV_CONFIG: NavGroup[] = [
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
      { path: '/teams', labelKey: 'nav.teams' },
      { path: '/services', labelKey: 'nav.services' },
      { path: '/schedules', labelKey: 'nav.schedules' },
      { path: '/integrations', labelKey: 'nav.integrations' },
      { path: '/analytics', labelKey: 'nav.analytics' },
      { path: '/settings', labelKey: 'nav.settings' },
    ],
  },
  {
    section: 'admin',
    items: [
      { path: '/users', labelKey: 'nav.users', adminOnly: true },
      { path: '/audit-log', labelKey: 'nav.auditLog', adminOnly: true },
      { path: '/status-pages', labelKey: 'nav.statusPages', adminOnly: true },
      { path: '/settings/incident-roles', labelKey: 'nav.roleDefinitions', adminOnly: true },
    ],
  },
]

const ALL_NAV_ITEMS = SIDEBAR_NAV_CONFIG.flatMap((group) => group.items)

export function filterNavGroupsForRole(isAdmin: boolean): NavGroup[] {
  return SIDEBAR_NAV_CONFIG.map((group) => ({
    ...group,
    items: group.items.filter((item) => !item.adminOnly || isAdmin),
  })).filter((group) => group.items.length > 0)
}

export function isNavItemActive(pathname: string, item: NavItem): boolean {
  if (item.path === '/dashboard') {
    return pathname === '/dashboard'
  }

  const matches = pathname === item.path || pathname.startsWith(`${item.path}/`)
  if (!matches) {
    return false
  }

  const hasMoreSpecificMatch = ALL_NAV_ITEMS.some(
    (other) =>
      other.path !== item.path &&
      other.path.startsWith(`${item.path}/`) &&
      (pathname === other.path || pathname.startsWith(`${other.path}/`)),
  )

  return !hasMoreSpecificMatch
}
