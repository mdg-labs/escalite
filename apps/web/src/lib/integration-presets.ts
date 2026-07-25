/** Generic-webhook field mapping config (matches services/integrations/genericwebhook). */
export type GenericWebhookMapping = {
  title: string
  body?: string
  dedup_key: string
  priority?: string
  event_type?: string
}

export type IntegrationPreset = {
  id: string
  label: string
  description: string
  pluginName: 'generic-webhook'
  mapping: GenericWebhookMapping
}

/** Beszel sends Shoutrrr generic:// JSON with title and message fields. */
export const BESZEL_PRESET: IntegrationPreset = {
  id: 'beszel',
  label: 'Beszel',
  description:
    'Self-hosted server monitor using Shoutrrr generic:// webhooks (title and message fields).',
  pluginName: 'generic-webhook',
  mapping: {
    title: 'title',
    body: 'message',
    dedup_key: 'title',
  },
}

export const INTEGRATION_PRESETS: IntegrationPreset[] = [BESZEL_PRESET]

export function findIntegrationPreset(id: string): IntegrationPreset | undefined {
  return INTEGRATION_PRESETS.find((preset) => preset.id === id)
}

export function buildIntegrationKeyConfigFromPreset(
  preset: IntegrationPreset,
): GenericWebhookMapping {
  return { ...preset.mapping }
}

/** Build the inbound webhook URL for an integration key token. */
export function buildInboundWebhookURL(
  apiBaseUrl: string,
  pluginName: string,
  token: string,
): string {
  const base = apiBaseUrl.replace(/\/+$/, '')
  return `${base}/webhook/${pluginName}/${token}`
}

/**
 * Build the Shoutrrr generic:// URL for Beszel notification settings.
 * Beszel uses Shoutrrr with template=json, producing {"title":"...","message":"..."}.
 */
export function buildBeszelShoutrrrURL(webhookURL: string): string {
  return `generic://${webhookURL}?template=json`
}
