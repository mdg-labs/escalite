import { useEffect, useState, type FormEvent, type ReactElement } from 'react'
import {
  useSaveSamlSettingsMutation,
  useSamlSettingsQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  Badge,
  Button,
  Textarea,
} from '@escalite/ui'
import { CircleCheckIcon, ShieldIcon } from 'lucide-react'

import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

export function SamlSettingsPanel(): ReactElement {
  const [{ data, fetching, error }, reexecuteQuery] = useSamlSettingsQuery({
    requestPolicy: 'network-only',
  })
  const [, saveSamlSettings] = useSaveSamlSettingsMutation()
  const [metadataXml, setMetadataXml] = useState('')
  const [enabled, setEnabled] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const settings = data?.samlSettings

  useEffect(() => {
    if (settings) {
      setEnabled(settings.enabled)
    }
  }, [settings])

  async function handleSave(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    setActionError(null)
    setSaving(true)

    const result = await saveSamlSettings({
      input: {
        metadataXml: metadataXml.trim(),
        enabled,
      },
    })
    setSaving(false)

    if (result.error) {
      showMutationError(result.error, 'saml.error.action')
      return
    }

    notifyMutationSuccess('saml.toast.saved')
    setMetadataXml('')
    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex items-start gap-3">
        <ShieldIcon className="mt-0.5 size-5 text-muted-foreground" />
        <div>
          <h2 className="text-xl font-semibold text-foreground">{t('settings.saml.title')}</h2>
          <p className="mt-2 text-sm text-muted-foreground">{t('settings.saml.description')}</p>
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
          <p className="text-sm text-muted-foreground">{t('settings.saml.loading')}</p>
        ) : settings?.configured ? (
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={settings.enabled ? 'success' : 'secondary'}>
                <CircleCheckIcon />
                {settings.enabled
                  ? t('settings.saml.status.enabled')
                  : t('settings.saml.status.disabled')}
              </Badge>
              {settings.idpEntityId ? (
                <span className="text-sm text-foreground">{settings.idpEntityId}</span>
              ) : null}
            </div>
            {settings.certificateHint ? (
              <p className="text-sm text-muted-foreground">
                {t('settings.saml.certificateHint', { hint: settings.certificateHint })}
              </p>
            ) : null}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">{t('settings.saml.status.notConfigured')}</p>
        )}

        <form className="space-y-4" onSubmit={(event) => void handleSave(event)}>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="saml-metadata">
              {t('settings.saml.metadata.label')}
            </label>
            <Textarea
              id="saml-metadata"
              onChange={(event) => setMetadataXml(event.target.value)}
              placeholder={t('settings.saml.metadata.placeholder')}
              rows={10}
              value={metadataXml}
            />
            <p className="text-xs text-muted-foreground">{t('settings.saml.metadata.help')}</p>
          </div>
          <label className="flex items-center gap-2 text-sm text-foreground">
            <input
              checked={enabled}
              className="size-4 rounded border border-input"
              onChange={(event) => setEnabled(event.target.checked)}
              type="checkbox"
            />
            {t('settings.saml.enabled.label')}
          </label>
          <Button
            disabled={saving || (metadataXml.trim() === '' && !settings?.configured)}
            type="submit"
          >
            {saving ? t('settings.saml.action.saving') : t('settings.saml.action.save')}
          </Button>
        </form>
      </div>
    </section>
  )
}
