import { useEffect, useMemo, useState, type FormEvent, type ReactElement } from 'react'
import {
  useNotificationChannelsQuery,
  useSaveUserContactMethodMutation,
  type NotificationChannelsQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  Button,
  Input,
} from '@escalite/ui'
import { BellIcon } from 'lucide-react'

import { formatGraphQLError } from '../lib/format'
import { t, type MessageKey } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'
import {
  buildConfigFromFormValues,
  buildFormFieldsFromConfigSchema,
  mergeConfigInitialValues,
  validateConfigFormValues,
} from '../lib/notification-channel-form'

type ChannelDefinition = NotificationChannelsQuery['notificationChannels'][number]

const CHANNEL_LABEL_KEYS: Record<string, MessageKey> = {
  email: 'settings.contactMethods.channel.email',
  push: 'settings.contactMethods.channel.push',
  'slack-dm': 'settings.contactMethods.channel.slackDm',
  webhook: 'settings.contactMethods.channel.webhook',
}

const CHANNEL_DESCRIPTION_KEYS: Record<string, MessageKey> = {
  email: 'settings.contactMethods.channel.description.email',
  push: 'settings.contactMethods.channel.description.push',
  'slack-dm': 'settings.contactMethods.channel.description.slackDm',
  webhook: 'settings.contactMethods.channel.description.webhook',
}

function channelLabel(name: string): string {
  const key = CHANNEL_LABEL_KEYS[name]
  return key ? t(key) : name
}

function channelDescription(name: string): string | null {
  const key = CHANNEL_DESCRIPTION_KEYS[name]
  return key ? t(key) : null
}

function ContactMethodChannelForm({
  channel,
}: {
  channel: ChannelDefinition
}): ReactElement {
  const [, saveContactMethod] = useSaveUserContactMethodMutation()
  const [values, setValues] = useState<Record<string, string>>({})
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const formFields = useMemo(
    () => buildFormFieldsFromConfigSchema(channel.configSchema),
    [channel.configSchema],
  )

  useEffect(() => {
    setValues(mergeConfigInitialValues(formFields, {}))
    setActionError(null)
  }, [formFields, channel.name])

  function handleConfigChange(name: string, value: string): void {
    setValues((current) => ({ ...current, [name]: value }))
  }

  async function handleSave(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    setActionError(null)

    const validationError = validateConfigFormValues(formFields, values)
    if (validationError) {
      setActionError(validationError)
      return
    }

    setSaving(true)
    const config = buildConfigFromFormValues(formFields, values)
    const result = await saveContactMethod({
      input: {
        channel: channel.name,
        config,
      },
    })
    setSaving(false)

    if (result.error) {
      showMutationError(result.error, 'contactMethods.error.action')
      return
    }

    notifyMutationSuccess('contactMethods.toast.saved')
  }

  const description = channelDescription(channel.name)

  return (
    <form
      className="space-y-4 rounded-lg border border-border bg-muted/30 p-4"
      onSubmit={(event) => void handleSave(event)}
    >
      <div>
        <h3 className="text-sm font-medium text-foreground">{channelLabel(channel.name)}</h3>
        {description ? (
          <p className="mt-1 text-sm text-muted-foreground">{description}</p>
        ) : null}
      </div>

      {formFields.length > 0 ? (
        <fieldset className="space-y-4">
          {formFields.map((field) => (
            <div className="space-y-2" key={field.name}>
              <label className="text-sm font-medium text-foreground" htmlFor={`${channel.name}-${field.name}`}>
                {field.label}
                {field.required ? <span className="text-destructive"> *</span> : null}
              </label>
              <Input
                autoComplete="off"
                id={`${channel.name}-${field.name}`}
                onChange={(event) => handleConfigChange(field.name, event.target.value)}
                placeholder={field.format === 'uri' ? 'https://example.com/hook' : field.name}
                required={field.required}
                type={field.format === 'uri' ? 'url' : 'text'}
                value={values[field.name] ?? ''}
              />
            </div>
          ))}
        </fieldset>
      ) : (
        <p className="text-sm text-muted-foreground">{t('settings.contactMethods.noFields')}</p>
      )}

      {actionError ? (
        <Alert variant="error">
          <AlertDescription>{actionError}</AlertDescription>
        </Alert>
      ) : null}

      <Button disabled={saving} type="submit">
        {saving ? t('settings.contactMethods.action.saving') : t('settings.contactMethods.action.save')}
      </Button>
    </form>
  )
}

export function ContactMethodsPanel(): ReactElement {
  const [{ data, fetching, error }] = useNotificationChannelsQuery({
    requestPolicy: 'cache-first',
  })

  const channels = data?.notificationChannels ?? []

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex items-start gap-3">
        <BellIcon className="mt-0.5 size-5 text-muted-foreground" />
        <div>
          <h2 className="text-xl font-semibold text-foreground">{t('settings.contactMethods.title')}</h2>
          <p className="mt-2 text-sm text-muted-foreground">{t('settings.contactMethods.description')}</p>
        </div>
      </div>

      {error ? (
        <Alert className="mt-6" variant="error">
          <AlertDescription>{formatGraphQLError(error.message)}</AlertDescription>
        </Alert>
      ) : null}

      <div className="mt-6 space-y-4">
        {fetching && channels.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('settings.contactMethods.loading')}</p>
        ) : channels.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('settings.contactMethods.empty')}</p>
        ) : (
          channels.map((channel) => (
            <ContactMethodChannelForm channel={channel} key={channel.name} />
          ))
        )}
      </div>
    </section>
  )
}
