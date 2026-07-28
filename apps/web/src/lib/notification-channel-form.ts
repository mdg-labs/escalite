/**
 * Build form field descriptors from a notification channel or integration plugin JSON Schema.
 * Supports the subset used by engine channel and inbound plugin ConfigSchema() definitions.
 */
export type NotificationChannelFormField = {
  name: string
  label: string
  type: 'string'
  required: boolean
  format?: string
}

export type ConfigSchema = {
  properties?: Record<
    string,
    {
      type?: string
      title?: string
      format?: string
    }
  >
  required?: string[]
}

export type PluginConfigDefinition = {
  name: string
  configSchema: ConfigSchema
}

export function buildFormFieldsFromConfigSchema(
  schema: ConfigSchema,
): NotificationChannelFormField[] {
  const properties = schema.properties ?? {}
  const required = new Set(schema.required ?? [])

  return Object.entries(properties)
    .filter(([, property]) => property.type === 'string' || property.type === undefined)
    .map(([name, property]) => ({
      name,
      label: property.title ?? name,
      type: 'string' as const,
      required: required.has(name),
      format: property.format,
    }))
}

export function findPluginConfigSchema(
  plugins: PluginConfigDefinition[],
  pluginName: string,
): ConfigSchema | undefined {
  return plugins.find((plugin) => plugin.name === pluginName)?.configSchema
}

export function buildConfigFromFormValues(
  fields: NotificationChannelFormField[],
  values: Record<string, string>,
): Record<string, string> {
  const config: Record<string, string> = {}

  for (const field of fields) {
    const value = values[field.name]?.trim() ?? ''
    if (value) {
      config[field.name] = value
    }
  }

  return config
}

export function validateConfigFormValues(
  fields: NotificationChannelFormField[],
  values: Record<string, string>,
): string | null {
  for (const field of fields) {
    if (field.required && !values[field.name]?.trim()) {
      return `${field.label} is required.`
    }
  }

  return null
}

export function mergeConfigInitialValues(
  fields: NotificationChannelFormField[],
  defaults: Record<string, string | undefined>,
): Record<string, string> {
  const values: Record<string, string> = {}

  for (const field of fields) {
    values[field.name] = defaults[field.name] ?? ''
  }

  return values
}
