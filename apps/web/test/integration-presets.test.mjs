import assert from 'node:assert/strict'
import test from 'node:test'

import {
  BESZEL_PRESET,
  GENERIC_WEBHOOK_PRESET,
  INTEGRATION_PRESETS,
  PROMETHEUS_ALERTMANAGER_PRESET,
  UPTIME_KUMA_PRESET,
  buildAlertmanagerDocsSnippet,
  buildBeszelDocsSnippet,
  buildBeszelShoutrrrURL,
  buildGenericWebhookDocsSnippet,
  buildInboundWebhookURL,
  buildIntegrationKeyConfigFromPreset,
  buildShoutrrrGenericURL,
  buildUptimeKumaDocsSnippet,
  findIntegrationPreset,
} from '../src/lib/integration-presets.ts'

test('BESZEL_PRESET maps title and message to generic-webhook fields', () => {
  assert.equal(BESZEL_PRESET.pluginName, 'generic-webhook')
  assert.deepEqual(buildIntegrationKeyConfigFromPreset(BESZEL_PRESET), {
    title: 'title',
    body: 'message',
    dedup_key: 'title',
  })
})

test('GENERIC_WEBHOOK_PRESET provides default field mapping', () => {
  assert.equal(GENERIC_WEBHOOK_PRESET.pluginName, 'generic-webhook')
  assert.deepEqual(buildIntegrationKeyConfigFromPreset(GENERIC_WEBHOOK_PRESET), {
    title: 'title',
    body: 'message',
    dedup_key: 'title',
  })
})

test('PROMETHEUS_ALERTMANAGER_PRESET has no field mapping', () => {
  assert.equal(PROMETHEUS_ALERTMANAGER_PRESET.pluginName, 'prometheus-alertmanager')
  assert.deepEqual(buildIntegrationKeyConfigFromPreset(PROMETHEUS_ALERTMANAGER_PRESET), {})
})

test('UPTIME_KUMA_PRESET has no field mapping', () => {
  assert.equal(UPTIME_KUMA_PRESET.pluginName, 'uptime-kuma')
  assert.deepEqual(buildIntegrationKeyConfigFromPreset(UPTIME_KUMA_PRESET), {})
})

test('INTEGRATION_PRESETS includes all day-1 presets', () => {
  const ids = INTEGRATION_PRESETS.map((preset) => preset.id)
  assert.deepEqual(ids, [
    'generic-webhook',
    'prometheus-alertmanager',
    'uptime-kuma',
    'beszel',
  ])
})

test('findIntegrationPreset returns presets by id', () => {
  assert.equal(findIntegrationPreset('beszel')?.label, 'Beszel')
  assert.equal(findIntegrationPreset('generic-webhook')?.label, 'Generic webhook')
  assert.equal(findIntegrationPreset('prometheus-alertmanager')?.label, 'Prometheus Alertmanager')
  assert.equal(findIntegrationPreset('uptime-kuma')?.label, 'Uptime Kuma')
  assert.equal(findIntegrationPreset('unknown'), undefined)
})

test('buildInboundWebhookURL joins base URL, plugin, and token', () => {
  const url = buildInboundWebhookURL('https://alerts.example.com/', 'generic-webhook', 'tok123')
  assert.equal(url, 'https://alerts.example.com/webhook/generic-webhook/tok123')
})

test('buildShoutrrrGenericURL uses exact generic:// format with template=json', () => {
  const webhookURL = 'https://alerts.example.com/webhook/generic-webhook/tok123'
  assert.equal(
    buildShoutrrrGenericURL(webhookURL),
    'generic://https://alerts.example.com/webhook/generic-webhook/tok123?template=json',
  )
})

test('buildBeszelShoutrrrURL delegates to buildShoutrrrGenericURL', () => {
  const webhookURL = 'https://alerts.example.com/webhook/generic-webhook/tok123'
  assert.equal(buildBeszelShoutrrrURL(webhookURL), buildShoutrrrGenericURL(webhookURL))
})

test('buildGenericWebhookDocsSnippet includes webhook URL', () => {
  const snippet = buildGenericWebhookDocsSnippet('https://alerts.example.com/webhook/generic-webhook/tok')
  assert.match(snippet, /https:\/\/alerts\.example\.com\/webhook\/generic-webhook\/tok/)
  assert.match(snippet, /POST JSON alerts/)
})

test('buildBeszelDocsSnippet includes Shoutrrr URL', () => {
  const webhookURL = 'https://alerts.example.com/webhook/generic-webhook/tok123'
  const snippet = buildBeszelDocsSnippet(webhookURL)
  assert.match(snippet, /generic:\/\//)
  assert.match(snippet, /template=json/)
})

test('buildAlertmanagerDocsSnippet includes webhook URL and send_resolved', () => {
  const webhookURL = 'https://alerts.example.com/webhook/prometheus-alertmanager/tok'
  const snippet = buildAlertmanagerDocsSnippet(webhookURL)
  assert.match(snippet, /send_resolved: true/)
  assert.match(snippet, new RegExp(webhookURL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
})

test('buildUptimeKumaDocsSnippet includes webhook URL', () => {
  const webhookURL = 'https://alerts.example.com/webhook/uptime-kuma/tok'
  const snippet = buildUptimeKumaDocsSnippet(webhookURL)
  assert.match(snippet, /Uptime Kuma/)
  assert.match(snippet, new RegExp(webhookURL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
})
