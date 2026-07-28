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
  pluginName: 'generic-webhook' | 'prometheus-alertmanager' | 'uptime-kuma'
  /** Pre-filled JSON path mapping for generic-webhook presets only. */
  mapping?: GenericWebhookMapping
  /** Short hint shown in the integration picker before key creation. */
  pickerHint?: string
  /** Setup instructions with the inbound webhook URL filled in. */
  buildDocsSnippet: (webhookURL: string) => string
}

/** Default generic-webhook mapping for custom JSON alert sources. */
export const GENERIC_WEBHOOK_PRESET: IntegrationPreset = {
  id: 'generic-webhook',
  label: 'Generic webhook',
  description:
    'Custom JSON webhook with configurable field mapping for any tool not covered by a dedicated preset.',
  pluginName: 'generic-webhook',
  mapping: {
    title: 'title',
    body: 'message',
    dedup_key: 'title',
  },
  pickerHint: 'Adjust the JSON paths below to match your payload, then POST alerts to the webhook URL.',
  buildDocsSnippet: buildGenericWebhookDocsSnippet,
}

/**
 * Beszel sends Shoutrrr generic:// JSON with title and message fields.
 *
 * Shoutrrr URL format: generic://<webhook-url>?template=json
 * Produces payload: {"title":"...","message":"..."} (default keys).
 */
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
  pickerHint: 'Beszel Shoutrrr URL is available after the key is created.',
  buildDocsSnippet: buildBeszelDocsSnippet,
}

export const PROMETHEUS_ALERTMANAGER_PRESET: IntegrationPreset = {
  id: 'prometheus-alertmanager',
  label: 'Prometheus Alertmanager',
  description:
    'Prometheus Alertmanager v4 webhook receiver with auto-resolve on resolved alerts.',
  pluginName: 'prometheus-alertmanager',
  pickerHint: 'No field mapping needed — paste the webhook URL into your Alertmanager receiver.',
  buildDocsSnippet: buildAlertmanagerDocsSnippet,
}

export const UPTIME_KUMA_PRESET: IntegrationPreset = {
  id: 'uptime-kuma',
  label: 'Uptime Kuma',
  description:
    'Uptime Kuma monitor down/up notifications with auto-resolve when the monitor recovers.',
  pluginName: 'uptime-kuma',
  pickerHint: 'No field mapping needed — add the webhook URL as a notification in Uptime Kuma.',
  buildDocsSnippet: buildUptimeKumaDocsSnippet,
}

export const INTEGRATION_PRESETS: IntegrationPreset[] = [
  GENERIC_WEBHOOK_PRESET,
  PROMETHEUS_ALERTMANAGER_PRESET,
  UPTIME_KUMA_PRESET,
  BESZEL_PRESET,
]

export function findIntegrationPreset(id: string): IntegrationPreset | undefined {
  return INTEGRATION_PRESETS.find((preset) => preset.id === id)
}

export function buildIntegrationKeyConfigFromPreset(
  preset: IntegrationPreset,
): GenericWebhookMapping | Record<string, never> {
  if (!preset.mapping) {
    return {}
  }
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
 * Build the Shoutrrr generic:// URL for tools that send flat JSON (e.g. Beszel).
 *
 * Format: generic://<webhook-url>?template=json
 * Default payload keys: title, message (override with titlekey/messagekey query params).
 */
export function buildShoutrrrGenericURL(webhookURL: string): string {
  return `generic://${webhookURL}?template=json`
}

/**
 * Build the Shoutrrr generic:// URL for Beszel notification settings.
 * Beszel uses Shoutrrr with template=json, producing {"title":"...","message":"..."}.
 */
export function buildBeszelShoutrrrURL(webhookURL: string): string {
  return buildShoutrrrGenericURL(webhookURL)
}

export function buildGenericWebhookDocsSnippet(webhookURL: string): string {
  return [
    `POST JSON alerts to:`,
    webhookURL,
    '',
    'Example body (adjust field names to match your mapping):',
    '{"title":"Alert title","message":"Details"}',
  ].join('\n')
}

export function buildBeszelDocsSnippet(webhookURL: string): string {
  return [
    'Paste this Shoutrrr URL into Beszel Settings → Notifications:',
    buildBeszelShoutrrrURL(webhookURL),
  ].join('\n')
}

export function buildAlertmanagerDocsSnippet(webhookURL: string): string {
  return [
    'Add a webhook receiver in alertmanager.yml:',
    '',
    'receivers:',
    '  - name: escalite',
    '    webhook_configs:',
    `      - url: '${webhookURL}'`,
    '        send_resolved: true',
  ].join('\n')
}

export function buildUptimeKumaDocsSnippet(webhookURL: string): string {
  return [
    'In Uptime Kuma: Settings → Notifications → Setup Notification → Webhook',
    '',
    `Webhook URL: ${webhookURL}`,
  ].join('\n')
}
