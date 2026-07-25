import { useEffect, useState, type ReactElement } from 'react'
import { useLocation } from 'react-router'

import { appConfig } from '../lib/config'

const mobileDeepLinkScheme = 'escalite://auth'

export function LoginMobilePage(): ReactElement {
  const location = useLocation()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

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
          throw new Error(body?.error ?? 'Unable to start mobile sign-in')
        }

        const body = (await response.json()) as { code: string }
        if (!body.code) {
          throw new Error('Missing mobile auth code')
        }

        if (!cancelled) {
          window.location.replace(`${mobileDeepLinkScheme}?code=${encodeURIComponent(body.code)}`)
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Unable to start mobile sign-in')
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
      Returning to the Escalite app…
    </div>
  )
}
