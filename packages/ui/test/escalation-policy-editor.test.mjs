import assert from 'node:assert/strict'
import test from 'node:test'

import {
  addEscalationStep,
  normalizeStepOrders,
  removeEscalationStep,
  reorderEscalationSteps,
} from '../domain/EscalationPolicyEditor/reorder.ts'
import { createEmptyStep, createEmptyTarget } from '../domain/EscalationPolicyEditor/types.ts'
import {
  canSaveEscalationPolicy,
  toEscalationPolicySavePayload,
  validateEscalationPolicySteps,
} from '../domain/EscalationPolicyEditor/validation.ts'

function makeStep(overrides = {}) {
  return {
    id: overrides.id ?? 'step-1',
    stepOrder: overrides.stepOrder ?? 1,
    delayMinutes: overrides.delayMinutes ?? 0,
    repeatLastStep: false,
    maxRepeats: null,
    targets: overrides.targets ?? [
      {
        id: 'target-1',
        targetType: 'user',
        userId: 'user-1',
      },
    ],
    ...overrides,
  }
}

test('reorderEscalationSteps updates contiguous step orders', () => {
  const steps = [
    makeStep({ id: 'a', stepOrder: 1 }),
    makeStep({ id: 'b', stepOrder: 2, delayMinutes: 10 }),
    makeStep({ id: 'c', stepOrder: 3, delayMinutes: 20 }),
  ]

  const reordered = reorderEscalationSteps(steps, 'c', 'a')

  assert.deepEqual(
    reordered.map((step) => step.id),
    ['c', 'a', 'b'],
  )
  assert.deepEqual(
    reordered.map((step) => step.stepOrder),
    [1, 2, 3],
  )
})

test('validateEscalationPolicySteps rejects steps without targets', () => {
  const issues = validateEscalationPolicySteps([
    makeStep({ targets: [] }),
    makeStep({
      id: 'step-2',
      stepOrder: 2,
      targets: [{ id: 'target-2', targetType: 'webhook' }],
    }),
  ])

  assert.equal(issues.length, 2)
  assert.match(issues[0]?.message ?? '', /at least one target/i)
  assert.match(issues[1]?.message ?? '', /incomplete targets/i)
})

test('canSaveEscalationPolicy returns false when any step is invalid', () => {
  assert.equal(canSaveEscalationPolicy([makeStep({ targets: [] })]), false)
  assert.equal(canSaveEscalationPolicy([makeStep()]), true)
})

test('toEscalationPolicySavePayload normalizes order after reorder', () => {
  const payload = toEscalationPolicySavePayload('On-call', [
    makeStep({ id: 'b', stepOrder: 2, delayMinutes: 15 }),
    makeStep({ id: 'a', stepOrder: 1, delayMinutes: 5 }),
  ])

  assert.deepEqual(payload, {
    name: 'On-call',
    steps: [
      {
        stepOrder: 1,
        delayMinutes: 0,
        repeatLastStep: false,
        maxRepeats: undefined,
        targets: [{ targetType: 'user', userId: 'user-1' }],
      },
      {
        stepOrder: 2,
        delayMinutes: 5,
        repeatLastStep: false,
        maxRepeats: undefined,
        targets: [{ targetType: 'user', userId: 'user-1' }],
      },
    ],
  })
})

test('addEscalationStep and removeEscalationStep keep at least one step', () => {
  const single = [makeStep()]
  assert.deepEqual(removeEscalationStep(single, 'step-1'), single)

  const added = addEscalationStep(single)
  assert.equal(added.length, 2)

  const removed = removeEscalationStep(added, added[1]?.id ?? '')
  assert.equal(removed.length, 1)
})

test('createEmptyTarget and createEmptyStep provide editor defaults', () => {
  const target = createEmptyTarget()
  assert.equal(target.targetType, 'user')

  const step = createEmptyStep(2)
  assert.equal(step.stepOrder, 2)
  assert.equal(step.targets.length, 1)
})

test('normalizeStepOrders forces first step delay to zero', () => {
  const normalized = normalizeStepOrders([
    makeStep({ id: 'a', stepOrder: 1, delayMinutes: 12 }),
    makeStep({ id: 'b', stepOrder: 2, delayMinutes: 4 }),
  ])

  assert.equal(normalized[0]?.delayMinutes, 0)
  assert.equal(normalized[1]?.delayMinutes, 4)
})
