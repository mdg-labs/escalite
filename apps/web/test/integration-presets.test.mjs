import assert from 'node:assert/strict'
import test from 'node:test'

import {
  BESZEL_PRESET,
  buildBeszelShoutrrrURL,
  buildInboundWebhookURL,
  buildIntegrationKeyConfigFromPreset,
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

test('findIntegrationPreset returns Beszel by id', () => {
  assert.equal(findIntegrationPreset('beszel')?.label, 'Beszel')
  assert.equal(findIntegrationPreset('unknown'), undefined)
})

test('buildInboundWebhookURL joins base URL, plugin, and token', () => {
  const url = buildInboundWebhookURL('https://alerts.example.com/', 'generic-webhook', 'tok123')
  assert.equal(url, 'https://alerts.example.com/webhook/generic-webhook/tok123')
})

test('buildBeszelShoutrrrURL uses exact generic:// format with template=json', () => {
  const webhookURL = 'https://alerts.example.com/webhook/generic-webhook/tok123'
  assert.equal(
    buildBeszelShoutrrrURL(webhookURL),
    'generic://https://alerts.example.com/webhook/generic-webhook/tok123?template=json',
  )
})
