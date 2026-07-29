import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

test('root stack screens disable back navigation and gestures', () => {
  const source = readFileSync(join(root, 'app/_layout.tsx'), 'utf8')
  assert.match(source, /headerBackVisible:\s*false/)
  assert.match(source, /gestureEnabled:\s*false/)
  assert.match(source, /name="index"/)
  assert.match(source, /name="auth"/)
  assert.match(source, /name="setup"/)
})

test('navigation shell wires hamburger header and drawer provider', () => {
  const layoutSource = readFileSync(join(root, 'app/_layout.tsx'), 'utf8')
  const shellSource = readFileSync(join(root, 'src/components/navigation-shell.tsx'), 'utf8')

  assert.match(layoutSource, /NavigationShellProvider/)
  assert.match(layoutSource, /HeaderMenuButton/)
  assert.match(shellSource, /openDrawer/)
  assert.match(shellSource, /NavigationDrawer/)
})

test('navigation drawer shows account and server actions', () => {
  const drawerSource = readFileSync(join(root, 'src/components/navigation-drawer.tsx'), 'utf8')
  const shellSource = readFileSync(join(root, 'src/components/navigation-shell.tsx'), 'utf8')

  assert.match(drawerSource, /Sign out/)
  assert.match(drawerSource, /Change server/)
  assert.match(drawerSource, /serverOrigin/)
  assert.match(shellSource, /clearServer/)
  assert.match(shellSource, /signOut/)
})

test('authenticated home removes inline sign out actions', () => {
  const source = readFileSync(join(root, 'app/index.tsx'), 'utf8')
  assert.doesNotMatch(source, /<Button[^>]*>\s*Sign out\s*<\/Button>/)
  assert.doesNotMatch(source, /onPress=\{\(\) => void signOut\(\)\}/)
})
