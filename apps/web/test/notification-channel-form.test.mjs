import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildConfigFromFormValues,
  buildFormFieldsFromConfigSchema,
  findPluginConfigSchema,
  mergeConfigInitialValues,
  validateConfigFormValues,
} from '../src/lib/notification-channel-form.ts'

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

test('findPluginConfigSchema returns schema for matching plugin name', () => {
  const schema = {
    type: 'object',
    properties: {
      title: { type: 'string', title: 'Title JSON path' },
      dedup_key: { type: 'string', title: 'Dedup key JSON path' },
    },
    required: ['title', 'dedup_key'],
  }

  const found = findPluginConfigSchema(
    [{ name: 'generic-webhook', configSchema: schema }],
    'generic-webhook',
  )

  assert.deepEqual(found, schema)
  assert.equal(findPluginConfigSchema([], 'generic-webhook'), undefined)
})

test('buildConfigFromFormValues omits empty optional fields', () => {
  const fields = buildFormFieldsFromConfigSchema({
    type: 'object',
    properties: {
      title: { type: 'string', title: 'Title JSON path' },
      body: { type: 'string', title: 'Body JSON path' },
      dedup_key: { type: 'string', title: 'Dedup key JSON path' },
    },
    required: ['title', 'dedup_key'],
  })

  assert.deepEqual(
    buildConfigFromFormValues(fields, {
      title: 'title',
      body: '  ',
      dedup_key: 'title',
    }),
    {
      title: 'title',
      dedup_key: 'title',
    },
  )
})

test('validateConfigFormValues reports missing required fields', () => {
  const fields = buildFormFieldsFromConfigSchema({
    type: 'object',
    properties: {
      title: { type: 'string', title: 'Title JSON path' },
      dedup_key: { type: 'string', title: 'Dedup key JSON path' },
    },
    required: ['title', 'dedup_key'],
  })

  assert.equal(
    validateConfigFormValues(fields, { title: 'title', dedup_key: ' ' }),
    'Dedup key JSON path is required.',
  )
  assert.equal(validateConfigFormValues(fields, { title: 'title', dedup_key: 'id' }), null)
})

test('mergeConfigInitialValues applies preset defaults to schema fields', () => {
  const fields = buildFormFieldsFromConfigSchema({
    type: 'object',
    properties: {
      title: { type: 'string', title: 'Title JSON path' },
      body: { type: 'string', title: 'Body JSON path' },
      dedup_key: { type: 'string', title: 'Dedup key JSON path' },
    },
    required: ['title', 'dedup_key'],
  })

  assert.deepEqual(
    mergeConfigInitialValues(fields, {
      title: 'title',
      body: 'message',
      dedup_key: 'title',
    }),
    {
      title: 'title',
      body: 'message',
      dedup_key: 'title',
    },
  )
})
