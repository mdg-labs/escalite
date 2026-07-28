import { useEffect, useMemo, useState, type ReactElement } from 'react'
import { useCreateIntegrationKeyMutation, useInboundIntegrationsQuery } from '@escalite/ts-types'
import { Alert, AlertDescription, AlertTitle, Button, Input } from '@escalite/ui'
import { AlertTriangleIcon } from 'lucide-react'

import {
  INTEGRATION_PRESETS,
  type IntegrationPreset,
} from '../lib/integration-presets'
import { formatGraphQLError } from '../lib/format'
import {
  buildConfigFromFormValues,
  buildFormFieldsFromConfigSchema,
  findPluginConfigSchema,
  mergeConfigInitialValues,
  validateConfigFormValues,
} from '../lib/notification-channel-form'

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
  const [{ data: inboundData, fetching: schemasFetching, error: schemasError }] =
    useInboundIntegrationsQuery({
      requestPolicy: 'cache-first',
    })

  const [selectedPresetId, setSelectedPresetId] = useState(INTEGRATION_PRESETS[0]?.id ?? '')
  const [configValues, setConfigValues] = useState<Record<string, string>>({})
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const selectedPreset = useMemo(
    () => INTEGRATION_PRESETS.find((item) => item.id === selectedPresetId),
    [selectedPresetId],
  )

  const inboundPlugins = useMemo(
    () =>
      (inboundData?.inboundIntegrations ?? []).map((plugin) => ({
        name: plugin.name,
        configSchema: plugin.configSchema,
      })),
    [inboundData?.inboundIntegrations],
  )

  const configSchema = useMemo(
    () =>
      selectedPreset
        ? findPluginConfigSchema(inboundPlugins, selectedPreset.pluginName)
        : undefined,
    [inboundPlugins, selectedPreset],
  )

  const formFields = useMemo(
    () => (configSchema ? buildFormFieldsFromConfigSchema(configSchema) : []),
    [configSchema],
  )

  useEffect(() => {
    if (!selectedPreset || formFields.length === 0) {
      setConfigValues({})
      return
    }

    setConfigValues(
      mergeConfigInitialValues(formFields, {
        title: selectedPreset.mapping?.title,
        body: selectedPreset.mapping?.body,
        dedup_key: selectedPreset.mapping?.dedup_key,
        priority: selectedPreset.mapping?.priority,
        event_type: selectedPreset.mapping?.event_type,
      }),
    )
  }, [formFields, selectedPreset])

  function handleConfigChange(name: string, value: string): void {
    setConfigValues((current) => ({ ...current, [name]: value }))
  }

  async function handleCreate(): Promise<void> {
    if (!selectedPreset) {
      setError('Select an integration preset.')
      return
    }

    const trimmedServiceId = serviceId.trim()
    if (!trimmedServiceId) {
      setError('Service ID is required.')
      return
    }

    if (!configSchema) {
      setError('Integration config schema is not available yet.')
      return
    }

    if (formFields.length > 0) {
      const validationError = validateConfigFormValues(formFields, configValues)
      if (validationError) {
        setError(validationError)
        return
      }
    }

    setError(null)
    setLoading(true)

    const config =
      formFields.length > 0 ? buildConfigFromFormValues(formFields, configValues) : {}

    const result = await createIntegrationKey({
      input: {
        serviceId: trimmedServiceId,
        pluginName: selectedPreset.pluginName,
        config,
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
      presetLabel: selectedPreset.label,
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

      {selectedPreset ? (
        <div className="space-y-2 rounded-lg border border-border bg-muted/40 p-4">
          <p className="text-sm font-medium text-foreground">Setup snippet</p>
          <pre className="overflow-x-auto whitespace-pre-wrap text-xs text-muted-foreground">
            {selectedPreset.buildDocsSnippet('https://your-escalite.example.com/webhook/<plugin>/<token>')}
          </pre>
        </div>
      ) : null}

      {schemasFetching ? (
        <p className="text-sm text-muted-foreground">Loading integration config schema…</p>
      ) : null}

      {schemasError ? (
        <Alert variant="error">
          <AlertTriangleIcon />
          <AlertTitle>Could not load config schema</AlertTitle>
          <AlertDescription>{formatGraphQLError(schemasError.message)}</AlertDescription>
        </Alert>
      ) : null}

      {formFields.length > 0 ? (
        <fieldset className="space-y-4">
          <legend className="text-sm font-medium text-foreground">Field mapping</legend>
          <p className="text-sm text-muted-foreground">
            JSON paths in the inbound webhook payload for each alert field.
          </p>
          {formFields.map((field) => (
            <div className="space-y-2" key={field.name}>
              <label className="text-sm font-medium text-foreground" htmlFor={field.name}>
                {field.label}
                {field.required ? <span className="text-destructive"> *</span> : null}
              </label>
              <Input
                id={field.name}
                onChange={(event) => handleConfigChange(field.name, event.target.value)}
                placeholder={field.format === 'uri' ? 'https://example.com/hook' : field.name}
                required={field.required}
                type={field.format === 'uri' ? 'url' : 'text'}
                value={configValues[field.name] ?? ''}
              />
            </div>
          ))}
        </fieldset>
      ) : null}

      {error ? (
        <Alert variant="error">
          <AlertTriangleIcon />
          <AlertTitle>Could not create key</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <Button
        disabled={loading || schemasFetching || !configSchema}
        onClick={() => void handleCreate()}
        type="button"
      >
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
        {preset.pickerHint ? (
          <span className="mt-2 block text-xs text-muted-foreground">{preset.pickerHint}</span>
        ) : null}
      </span>
    </button>
  )
}
