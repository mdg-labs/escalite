import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

test('on-call api queries myOnCallStatus via graphqlRequest', () => {
  const source = readFileSync(join(root, 'src/api/on-call.ts'), 'utf8')

  assert.match(source, /graphqlRequest/)
  assert.match(source, /myOnCallStatus/)
  assert.match(source, /scheduleName/)
  assert.match(source, /teamName/)
  assert.match(source, /layer/)
  assert.match(source, /until/)
  assert.doesNotMatch(source, /onCallNow/)
})

test('on-call api maps empty and multiple schedule responses', () => {
  const source = readFileSync(join(root, 'src/api/on-call.ts'), 'utf8')

  assert.match(source, /export function parseMyOnCallAssignments/)
  assert.match(source, /if \(!assignments\?\.length\)/)
  assert.match(source, /return \[\]/)
  assert.match(source, /export function isOnCall/)
  assert.match(source, /assignments\.length > 0/)
})

test('home screen refreshes on focus and pull-to-refresh', () => {
  const homeSource = readFileSync(join(root, 'app/index.tsx'), 'utf8')
  const hookSource = readFileSync(join(root, 'src/api/use-my-on-call-status.ts'), 'utf8')

  assert.match(homeSource, /useMyOnCallStatus/)
  assert.match(homeSource, /OnCallStatus/)
  assert.match(homeSource, /RefreshControl/)
  assert.match(hookSource, /useFocusEffect/)
  assert.match(hookSource, /from 'expo-router'/)
  assert.match(hookSource, /fetchMyOnCallStatus/)
})

test('on-call status component shows on call and not on call states', () => {
  const source = readFileSync(join(root, 'src/components/on-call-status.tsx'), 'utf8')

  assert.match(source, /On call/)
  assert.match(source, /Not on call/)
  assert.match(source, /On call until/)
  assert.match(source, /scheduleName/)
  assert.match(source, /teamName/)
  assert.match(source, /layer/)
})
