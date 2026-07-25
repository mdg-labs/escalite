import { useState, type ReactElement } from 'react'
import { useCreateIntegrationKeyMutation } from '@escalite/ts-types'
import { Alert, AlertDescription, AlertTitle, Button } from '@escalite/ui'
import { AlertTriangleIcon } from 'lucide-react'

import {
  BESZEL_PRESET,
  INTEGRATION_PRESETS,
  buildIntegrationKeyConfigFromPreset,
  type IntegrationPreset,
} from '../lib/integration-presets'
import { formatGraphQLError } from '../lib/format'

type IntegrationPickerProps = {
  serviceId: string
  onCreated: (payload: {
    pluginName: string
    token: string
    tokenPrefix: string
    presetLabel: string
  }) => void
}

export function IntegrationPicker({
  serviceId,
  onCreated,
}: IntegrationPickerProps): ReactElement {
  const [, createIntegrationKey] = useCreateIntegrationKeyMutation()
  const [selectedPresetId, setSelectedPresetId] = useState(BESZEL_PRESET.id)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

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

    onCreated({
      pluginName: key.pluginName,
      token: key.token,
      tokenPrefix: key.tokenPrefix,
      presetLabel: preset.label,
    })
  }

  return (
    <div className="space-y-6">
      <fieldset className="space-y-3">
        <legend className="text-sm font-medium text-foreground">Integration preset</legend>
        {INTEGRATION_PRESETS.map((preset) => (
          <PresetOption
            key={preset.id}
            onSelect={() => setSelectedPresetId(preset.id)}
            preset={preset}
            selected={selectedPresetId === preset.id}
          />
        ))}
      </fieldset>

      {error ? (
        <Alert variant="error">
          <AlertTriangleIcon />
          <AlertTitle>Could not create key</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <Button disabled={loading} onClick={() => void handleCreate()} type="button">
        {loading ? 'Creating…' : 'Create integration key'}
      </Button>
    </div>
  )
}

function PresetOption({
  preset,
  selected,
  onSelect,
}: {
  preset: IntegrationPreset
  selected: boolean
  onSelect: () => void
}): ReactElement {
  return (
    <button
      className={`flex w-full cursor-pointer gap-3 rounded-lg border bg-card p-4 text-left ${
        selected ? 'border-primary ring-2 ring-ring/24' : 'border-border'
      }`}
      onClick={onSelect}
      type="button"
    >
      <span>
        <span className="block text-sm font-medium text-foreground">{preset.label}</span>
        <span className="mt-1 block text-sm text-muted-foreground">{preset.description}</span>
        {preset.id === BESZEL_PRESET.id ? (
          <span className="mt-2 block text-xs text-muted-foreground">
            Beszel Shoutrrr URL is available after the key is created.
          </span>
        ) : null}
      </span>
    </button>
  )
}
