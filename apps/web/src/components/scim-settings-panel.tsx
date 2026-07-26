import { useEffect, useState, type ReactElement } from 'react'
import {
  useRotateScimTokenMutation,
  useScimSettingsQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  Badge,
  Button,
} from '@escalite/ui'
import { CircleCheckIcon, UsersIcon } from 'lucide-react'

import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

export function ScimSettingsPanel(): ReactElement {
  const [{ data, fetching, error }, reexecuteQuery] = useScimSettingsQuery({
    requestPolicy: 'network-only',
  })
  const [, rotateScimToken] = useRotateScimTokenMutation()
  const [actionError, setActionError] = useState<string | null>(null)
  const [rotating, setRotating] = useState(false)
  const [revealedToken, setRevealedToken] = useState<string | null>(null)

  const settings = data?.scimSettings

  useEffect(() => {
    if (!settings?.configured) {
      setRevealedToken(null)
    }
  }, [settings?.configured])

  async function handleRotate(): Promise<void> {
    setActionError(null)
    setRotating(true)

    const result = await rotateScimToken({})
    setRotating(false)

    if (result.error) {
      setActionError(formatGraphQLError(result.error.message))
      return
    }

    setRevealedToken(result.data?.rotateScimToken.token ?? null)
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex items-start gap-3">
        <UsersIcon className="mt-0.5 size-5 text-muted-foreground" />
        <div>
          <h2 className="text-xl font-semibold text-foreground">{t('settings.scim.title')}</h2>
          <p className="mt-2 text-sm text-muted-foreground">{t('settings.scim.description')}</p>
        </div>
      </div>

      {(error || actionError) && (
        <Alert className="mt-6" variant="error">
          <AlertDescription>
            {actionError ?? (error ? formatGraphQLError(error.message) : null)}
          </AlertDescription>
        </Alert>
      )}

      <div className="mt-6 space-y-6">
        {fetching && !settings ? (
          <p className="text-sm text-muted-foreground">{t('settings.scim.loading')}</p>
        ) : settings?.configured ? (
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="success">
                <CircleCheckIcon />
                {t('settings.scim.status.configured')}
              </Badge>
              {settings.tokenHint ? (
                <span className="text-sm text-muted-foreground">
                  {t('settings.scim.tokenHint', { hint: settings.tokenHint })}
                </span>
              ) : null}
            </div>
            {settings.scimBaseUrl ? (
              <p className="text-sm text-foreground">
                {t('settings.scim.baseUrl', { url: settings.scimBaseUrl })}
              </p>
            ) : null}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">{t('settings.scim.status.notConfigured')}</p>
        )}

        {revealedToken ? (
          <Alert variant="warning">
            <AlertDescription>
              <p className="font-medium text-foreground">{t('settings.scim.tokenReveal.title')}</p>
              <p className="mt-2 break-all font-mono text-sm">{revealedToken}</p>
              <p className="mt-2 text-sm text-muted-foreground">
                {t('settings.scim.tokenReveal.help')}
              </p>
            </AlertDescription>
          </Alert>
        ) : null}

        <Button disabled={rotating} onClick={() => void handleRotate()} type="button">
          {rotating
            ? t('settings.scim.action.rotating')
            : settings?.configured
              ? t('settings.scim.action.rotate')
              : t('settings.scim.action.generate')}
        </Button>
      </div>
    </section>
  )
}
