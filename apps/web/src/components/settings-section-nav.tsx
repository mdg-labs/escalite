import type { ReactElement } from 'react'
import { Link, useLocation } from 'react-router'
import { cn } from '@escalite/ui'

import { t, type MessageKey } from '../lib/i18n'

export type SettingsSectionSlug = 'enterprise' | 'notifications' | 'devices' | 'roles'

export type SettingsSection = {
  slug: SettingsSectionSlug
  labelKey: MessageKey
  adminOnly?: boolean
}

export const SETTINGS_SECTIONS: SettingsSection[] = [
  { slug: 'enterprise', labelKey: 'settings.nav.enterprise' },
  { slug: 'notifications', labelKey: 'settings.nav.notifications' },
  { slug: 'devices', labelKey: 'settings.nav.devices' },
  { slug: 'roles', labelKey: 'settings.nav.roles', adminOnly: true },
]

type SettingsSectionNavProps = {
  isAdmin: boolean
}

export function SettingsSectionNav({ isAdmin }: SettingsSectionNavProps): ReactElement {
  const location = useLocation()
  const visibleSections = SETTINGS_SECTIONS.filter((section) => !section.adminOnly || isAdmin)

  return (
    <nav aria-label={t('settings.nav.label')}>
      <ul className="flex flex-row gap-1 overflow-x-auto pb-1 lg:flex-col lg:gap-0.5 lg:overflow-visible lg:pb-0">
        {visibleSections.map((section) => {
          const href = `/settings/${section.slug}`
          const active =
            location.pathname === href || location.pathname.startsWith(`${href}/`)

          return (
            <li key={section.slug} className="shrink-0">
              <Link
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'block rounded-md px-3 py-1.5 text-sm whitespace-nowrap transition-colors',
                  active
                    ? 'bg-accent font-medium text-foreground'
                    : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground',
                )}
                to={href}
              >
                {t(section.labelKey)}
              </Link>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
