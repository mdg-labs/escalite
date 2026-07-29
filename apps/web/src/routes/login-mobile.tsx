import { useEffect, useState, type ReactElement } from 'react'
import { useLocation } from 'react-router'

import { appConfig } from '../lib/config'
import { t } from '../lib/i18n'
import { buildMobileAuthRedirectUrl, resolveMobileAuthRedirectUri } from '../lib/mobile-auth-redirect'

export function LoginMobilePage(): ReactElement {
  const location = useLocation()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const redirectUri = resolveMobileAuthRedirectUri(new URLSearchParams(location.search).get('redirect_uri'))

    async function issueCodeAndRedirect(): Promise<void> {
      try {
        const response = await fetch(`${appConfig.apiPublicUrl}/api/v1/mobile/auth/code`, {
          method: 'POST',
          credentials: 'include',
        })

        if (response.status === 401) {
          if (!cancelled) {
            const redirect = encodeURIComponent(`${location.pathname}${location.search}`)
            window.location.replace(`/login?redirect=${redirect}`)
          }
          return
        }

        if (!response.ok) {
          const body = (await response.json().catch(() => null)) as { error?: string } | null
          throw new Error(body?.error ?? t('loginMobile.error.start'))
        }

        const body = (await response.json()) as { code: string }
        if (!body.code) {
          throw new Error(t('loginMobile.error.missingCode'))
        }

        if (!cancelled) {
          window.location.replace(buildMobileAuthRedirectUrl(redirectUri, body.code))
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : t('loginMobile.error.start'))
        }
      }
    }

    void issueCodeAndRedirect()

    return () => {
      cancelled = true
    }
  }, [location.pathname, location.search])

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4 text-center text-sm text-destructive-foreground">
        {error}
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 text-center text-sm text-muted-foreground">
      {t('loginMobile.returning')}
    </div>
  )
}
