import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildEscalationEditorOptions,
  createDefaultEditorPolicy,
  mapApiPolicyToEditor,
  mapOrganizationUsersToEditorOptions,
  mapTeamSchedulesToEditorOptions,
} from '../src/lib/escalation-policy.ts'

test('mapOrganizationUsersToEditorOptions sorts by email label', () => {
  const options = mapOrganizationUsersToEditorOptions([
    { id: 'user-2', email: 'zoe@example.com' },
    { id: 'user-1', email: 'amy@example.com' },
  ])

  assert.deepEqual(options, [
    { id: 'user-1', label: 'amy@example.com' },
    { id: 'user-2', label: 'zoe@example.com' },
  ])
})

test('mapTeamSchedulesToEditorOptions sorts by schedule name', () => {
  const options = mapTeamSchedulesToEditorOptions([
    { id: 'schedule-2', name: 'Weekend' },
    { id: 'schedule-1', name: 'Primary' },
  ])

  assert.deepEqual(options, [
    { id: 'schedule-1', label: 'Primary' },
    { id: 'schedule-2', label: 'Weekend' },
  ])
})

test('buildEscalationEditorOptions combines users and schedules', () => {
  const options = buildEscalationEditorOptions(
    [{ id: 'user-1', email: 'oncall@example.com' }],
    [{ id: 'schedule-1', name: 'Primary' }],
  )

  assert.deepEqual(options, {
    users: [{ id: 'user-1', label: 'oncall@example.com' }],
    schedules: [{ id: 'schedule-1', label: 'Primary' }],
  })
})

test('mapApiPolicyToEditor sorts steps and preserves targets map', () => {
  const editor = mapApiPolicyToEditor(
    {
      id: 'policy-1',
      name: 'Default',
      steps: [
        { id: 'step-2', stepOrder: 2, delayMinutes: 15, repeatLastStep: false },
        { id: 'step-1', stepOrder: 1, delayMinutes: 0, repeatLastStep: false },
      ],
    },
    {
      'step-1': [
        {
          id: 'target-1',
          targetType: 'user',
          userId: 'user-1',
        },
      ],
    },
  )

  assert.equal(editor.steps[0]?.id, 'step-1')
  assert.equal(editor.steps[1]?.id, 'step-2')
  assert.equal(editor.steps[0]?.targets[0]?.userId, 'user-1')
})

test('mapApiPolicyToEditor maps inline API targets', () => {
  const editor = mapApiPolicyToEditor({
    id: 'policy-1',
    name: 'Default',
    steps: [
      {
        id: 'step-1',
        stepOrder: 1,
        delayMinutes: 0,
        repeatLastStep: false,
        targets: [
          {
            id: 'target-1',
            targetType: 'user',
            userId: 'user-1',
          },
        ],
      },
    ],
  })

  assert.equal(editor.steps[0]?.targets[0]?.userId, 'user-1')
})

test('createDefaultEditorPolicy includes one valid step scaffold', () => {
  const editor = createDefaultEditorPolicy('Primary')

  assert.equal(editor.name, 'Primary')
  assert.equal(editor.steps.length, 1)
  assert.equal(editor.steps[0]?.targets.length, 1)
})
