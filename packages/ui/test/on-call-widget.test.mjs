import assert from 'node:assert/strict'
import test from 'node:test'

import {
  initialsFromLabel,
  layerRoleLabel,
  sortOnCallLayers,
  userLabel,
} from '../domain/OnCallWidget/utils.ts'

const labels = {
  title: 'On call now',
  loading: 'Loading…',
  empty: 'No on-call assignments',
  layerLabel: (layer) => `Layer ${layer}`,
  primaryLayer: 'Primary',
  secondaryLayer: 'Secondary',
}

test('initialsFromLabel handles single and multi-word names', () => {
  assert.equal(initialsFromLabel('Ada Lovelace'), 'AL')
  assert.equal(initialsFromLabel('ops'), 'OP')
  assert.equal(initialsFromLabel(''), '?')
})

test('userLabel resolves display names from the user directory', () => {
  const users = [
    { id: 'user-1', label: 'Ada Lovelace' },
    { id: 'user-2', label: 'Grace Hopper' },
  ]

  assert.equal(userLabel(users, 'user-2'), 'Grace Hopper')
  assert.equal(userLabel(users, 'missing'), 'missing')
})

test('sortOnCallLayers orders primary before secondary', () => {
  const sorted = sortOnCallLayers([
    { layer: 2, rotationId: 'rotation-2', userId: 'user-2' },
    { layer: 1, rotationId: 'rotation-1', userId: 'user-1' },
  ])

  assert.deepEqual(
    sorted.map((layer) => layer.layer),
    [1, 2],
  )
})

test('layerRoleLabel maps layer numbers to primary and secondary labels', () => {
  assert.equal(layerRoleLabel(1, labels), 'Primary')
  assert.equal(layerRoleLabel(2, labels), 'Secondary')
  assert.equal(layerRoleLabel(3, labels), 'Layer 3')
})
