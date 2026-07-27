import { useState, type ReactElement } from 'react'
import { useNavigate } from 'react-router'
import { useMeQuery } from '@escalite/ts-types'
import { Button } from '@escalite/ui'
import { LogOutIcon } from 'lucide-react'

import { appConfig } from '../lib/config'
import { t } from '../lib/i18n'

type UserMenuProps = {
  /** EL-166: close mobile drawer after logout navigation. */
  onNavigate?: () => void
}

async function postLogout(): Promise<void> {
  const url = new URL('/api/v1/logout', appConfig.apiPublicUrl)
  const response = await fetch(url, { method: 'POST', credentials: 'include' })
  if (!response.ok && response.status !== 204) {
    throw new Error(`logout failed (${response.status})`)
  }
}

export function UserMenu({ onNavigate }: UserMenuProps): ReactElement | null {
  const navigate = useNavigate()
  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const email = meData?.me?.email
  if (!email) {
    return null
  }

  async function handleLogout(): Promise<void> {
    setError(null)
    setLoading(true)

    try {
      await postLogout()
      onNavigate?.()
      navigate('/login', { replace: true })
    } catch {
      setLoading(false)
      setError(t('user.menu.logoutError'))
    }
  }

  return (
    <div className="flex flex-col gap-2" data-slot="user-menu">
      <p className="truncate px-1 text-xs text-sidebar-foreground/80" title={email}>
        {email}
      </p>
      <Button
        className="w-full justify-start gap-2 text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
        loading={loading}
        onClick={() => void handleLogout()}
        size="sm"
        type="button"
        variant="ghost"
      >
        <LogOutIcon className="size-4 shrink-0" />
        {t('user.menu.logout')}
      </Button>
      {error ? (
        <p className="px-1 text-xs text-destructive" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  )
}
