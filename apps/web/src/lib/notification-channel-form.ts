/**
 * Build form field descriptors from a notification channel JSON Schema.
 * Supports the subset used by engine channel ConfigSchema() definitions.
 */
export type NotificationChannelFormField = {
  name: string
  label: string
  type: 'string'
  required: boolean
  format?: string
}

type JsonSchema = {
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

export function buildFormFieldsFromConfigSchema(
  schema: JsonSchema,
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
