import assert from 'node:assert/strict'
import test from 'node:test'

import { buildFormFieldsFromConfigSchema } from '../src/lib/notification-channel-form.ts'

test('buildFormFieldsFromConfigSchema maps JSON Schema properties to form fields', () => {
  const fields = buildFormFieldsFromConfigSchema({
    type: 'object',
    properties: {
      expo_push_token: {
        type: 'string',
        title: 'Expo push token',
      },
      url: {
        type: 'string',
        format: 'uri',
        title: 'Webhook URL',
      },
    },
    required: ['expo_push_token'],
  })

  assert.equal(fields.length, 2)
  assert.deepEqual(fields[0], {
    name: 'expo_push_token',
    label: 'Expo push token',
    type: 'string',
    required: true,
    format: undefined,
  })
  assert.deepEqual(fields[1], {
    name: 'url',
    label: 'Webhook URL',
    type: 'string',
    required: false,
    format: 'uri',
  })
})

test('buildFormFieldsFromConfigSchema returns empty list for schema without properties', () => {
  const fields = buildFormFieldsFromConfigSchema({ type: 'object', additionalProperties: false })
  assert.deepEqual(fields, [])
})
