import { useEffect, useState, type FormEvent, type ReactElement } from 'react'
import { useSearchParams } from 'react-router'
import {
  useSaveSlackSettingsMutation,
  useSlackSettingsQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  Badge,
  Button,
  Input,
} from '@escalite/ui'
import { CircleCheckIcon, HashIcon, LinkIcon } from 'lucide-react'

import { appConfig } from '../lib/config'
import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

function slackInstallHref(oauthInstallUrl: string | null | undefined): string | null {
  if (!oauthInstallUrl) {
    return null
  }
  if (oauthInstallUrl.startsWith('http://') || oauthInstallUrl.startsWith('https://')) {
    return oauthInstallUrl
  }
  const base = appConfig.apiPublicUrl.replace(/\/$/, '')
  return `${base}${oauthInstallUrl}`
}

export function SlackSettingsPanel(): ReactElement {
  const [searchParams, setSearchParams] = useSearchParams()
  const [{ data, fetching, error }, reexecuteQuery] = useSlackSettingsQuery({
    requestPolicy: 'network-only',
  })
  const [, saveSlackSettings] = useSaveSlackSettingsMutation()
  const [manualToken, setManualToken] = useState('')
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [oauthStatus, setOauthStatus] = useState<'connected' | 'error' | null>(null)

  const settings = data?.slackSettings
  const installHref = slackInstallHref(settings?.oauthInstallUrl)

  useEffect(() => {
    const status = searchParams.get('slack')
    if (status === 'connected' || status === 'error') {
      setOauthStatus(status)
      const nextParams = new URLSearchParams(searchParams)
      nextParams.delete('slack')
      setSearchParams(nextParams, { replace: true })
      if (status === 'connected') {
        reexecuteQuery({ requestPolicy: 'network-only' })
      }
    }
  }, [reexecuteQuery, searchParams, setSearchParams])

  async function handleManualSave(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    setActionError(null)
    setSaving(true)

    const result = await saveSlackSettings({ input: { botToken: manualToken.trim() } })
    setSaving(false)

    if (result.error) {
      setActionError(formatGraphQLError(result.error.message))
      return
    }

    setManualToken('')
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex items-start gap-3">
        <HashIcon className="mt-0.5 size-5 text-muted-foreground" />
        <div>
          <h2 className="text-xl font-semibold text-foreground">{t('settings.slack.title')}</h2>
          <p className="mt-2 text-sm text-muted-foreground">{t('settings.slack.description')}</p>
        </div>
      </div>

      {oauthStatus === 'connected' && (
        <Alert className="mt-6" variant="success">
          <AlertDescription>{t('settings.slack.oauth.connected')}</AlertDescription>
        </Alert>
      )}

      {oauthStatus === 'error' && (
        <Alert className="mt-6" variant="error">
          <AlertDescription>{t('settings.slack.oauth.error')}</AlertDescription>
        </Alert>
      )}

      {(error || actionError) && (
        <Alert className="mt-6" variant="error">
          <AlertDescription>
            {actionError ?? (error ? formatGraphQLError(error.message) : null)}
          </AlertDescription>
        </Alert>
      )}

      <div className="mt-6 space-y-6">
        {fetching && !settings ? (
          <p className="text-sm text-muted-foreground">{t('settings.slack.loading')}</p>
        ) : settings?.configured ? (
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="success">
                <CircleCheckIcon />
                {t('settings.slack.status.connected')}
              </Badge>
              {settings.workspaceName ? (
                <span className="text-sm text-foreground">{settings.workspaceName}</span>
              ) : null}
            </div>
            {settings.tokenHint ? (
              <p className="text-sm text-muted-foreground">
                {t('settings.slack.tokenHint', { hint: settings.tokenHint })}
              </p>
            ) : null}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">{t('settings.slack.status.notConnected')}</p>
        )}

        {installHref ? (
          <div>
            <Button render={<a href={installHref} />} type="button" variant="default">
              <LinkIcon />
              {settings?.configured
                ? t('settings.slack.action.reconnect')
                : t('settings.slack.action.addToSlack')}
            </Button>
          </div>
        ) : null}

        <form className="space-y-4" onSubmit={(event) => void handleManualSave(event)}>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="slack-bot-token">
              {t('settings.slack.manual.label')}
            </label>
            <Input
              autoComplete="off"
              id="slack-bot-token"
              onChange={(event) => setManualToken(event.target.value)}
              placeholder={t('settings.slack.manual.placeholder')}
              type="password"
              value={manualToken}
            />
            <p className="text-xs text-muted-foreground">
              {installHref ? t('settings.slack.manual.helpDev') : t('settings.slack.manual.help')}
            </p>
          </div>
          <Button disabled={saving || manualToken.trim() === ''} type="submit">
            {saving ? t('settings.slack.action.saving') : t('settings.slack.action.saveToken')}
          </Button>
        </form>
      </div>
    </section>
  )
}
