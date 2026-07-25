import assert from 'node:assert/strict'
import test from 'node:test'

import {
  createDefaultEditorPolicy,
  mapApiPolicyToEditor,
} from '../src/lib/escalation-policy.ts'

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

test('createDefaultEditorPolicy includes one valid step scaffold', () => {
  const editor = createDefaultEditorPolicy('Primary')

  assert.equal(editor.name, 'Primary')
  assert.equal(editor.steps.length, 1)
  assert.equal(editor.steps[0]?.targets.length, 1)
})
