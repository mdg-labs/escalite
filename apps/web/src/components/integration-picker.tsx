import { useState, type ReactElement } from 'react'
import { useCreateIntegrationKeyMutation } from '@escalite/ts-types'
import { Button, Input } from '@escalite/ui'

import { appConfig } from '../lib/config'
import {
  BESZEL_PRESET,
  INTEGRATION_PRESETS,
  buildBeszelShoutrrrURL,
  buildInboundWebhookURL,
  buildIntegrationKeyConfigFromPreset,
  type IntegrationPreset,
} from '../lib/integration-presets'

type CreatedKey = {
  preset: IntegrationPreset
  token: string
  tokenPrefix: string
  webhookURL: string
  shoutrrrURL: string
}

function formatGraphQLError(message: string): string {
  return message.replace(/^(\[GraphQL\]\s*)+/, '')
}

export function IntegrationPicker(): ReactElement {
  const [, createIntegrationKey] = useCreateIntegrationKeyMutation()
  const [serviceId, setServiceId] = useState('')
  const [selectedPresetId, setSelectedPresetId] = useState(BESZEL_PRESET.id)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [createdKey, setCreatedKey] = useState<CreatedKey | null>(null)

  async function handleCreate(): Promise<void> {
    const preset = INTEGRATION_PRESETS.find((item) => item.id === selectedPresetId)
    if (!preset) {
      setError('Select an integration preset.')
      return
    }

    const trimmedServiceId = serviceId.trim()
    if (!trimmedServiceId) {
      setError('Service ID is required.')
      return
    }

    setError(null)
    setLoading(true)

    const result = await createIntegrationKey({
      input: {
        serviceId: trimmedServiceId,
        pluginName: preset.pluginName,
        config: buildIntegrationKeyConfigFromPreset(preset),
      },
    })

    setLoading(false)

    if (result.error) {
      setError(formatGraphQLError(result.error.message))
      return
    }

    const key = result.data?.createIntegrationKey
    if (!key?.token) {
      setError('Integration key was created but the token was not returned.')
      return
    }

    const webhookURL = buildInboundWebhookURL(
      appConfig.apiPublicUrl,
      preset.pluginName,
      key.token,
    )

    setCreatedKey({
      preset,
      token: key.token,
      tokenPrefix: key.tokenPrefix,
      webhookURL,
      shoutrrrURL: buildBeszelShoutrrrURL(webhookURL),
    })
  }

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground" htmlFor="service-id">
          Service ID
        </label>
        <Input
          id="service-id"
          onChange={(event) => setServiceId(event.target.value)}
          placeholder="Service UUID"
          value={serviceId}
        />
      </div>

      <fieldset className="space-y-3">
        <legend className="text-sm font-medium text-foreground">Integration preset</legend>
        {INTEGRATION_PRESETS.map((preset) => (
          <label
            key={preset.id}
            className="flex cursor-pointer gap-3 rounded-lg border border-border bg-card p-4"
          >
            <input
              checked={selectedPresetId === preset.id}
              className="mt-1"
              name="integration-preset"
              onChange={() => setSelectedPresetId(preset.id)}
              type="radio"
              value={preset.id}
            />
            <span>
              <span className="block text-sm font-medium text-foreground">{preset.label}</span>
              <span className="mt-1 block text-sm text-muted-foreground">{preset.description}</span>
            </span>
          </label>
        ))}
      </fieldset>

      {error ? (
        <p className="text-sm text-destructive-foreground" role="alert">
          {error}
        </p>
      ) : null}

      <Button disabled={loading} onClick={() => void handleCreate()} type="button">
        {loading ? 'Creating…' : 'Create integration key'}
      </Button>

      {createdKey ? (
        <section className="space-y-4 rounded-lg border border-border bg-muted/30 p-4">
          <h2 className="text-sm font-semibold text-foreground">
            {createdKey.preset.label} integration key created
          </h2>
          <p className="text-sm text-muted-foreground">
            Token prefix <span className="font-mono text-foreground">{createdKey.tokenPrefix}</span>{' '}
            — save the webhook URL below; the full token is shown once.
          </p>
          <div className="space-y-2">
            <p className="text-sm font-medium text-foreground">Webhook URL</p>
            <code className="block overflow-x-auto rounded-md bg-background p-3 text-xs text-foreground">
              {createdKey.webhookURL}
            </code>
          </div>
          {createdKey.preset.id === BESZEL_PRESET.id ? (
            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">
                Beszel notification URL (Shoutrrr generic://)
              </p>
              <code className="block overflow-x-auto rounded-md bg-background p-3 text-xs text-foreground">
                {createdKey.shoutrrrURL}
              </code>
              <p className="text-xs text-muted-foreground">
                Paste this into Beszel Settings → Notifications. See docs/integrations/beszel.md
                for setup details.
              </p>
            </div>
          ) : null}
        </section>
      ) : null}
    </div>
  )
}
