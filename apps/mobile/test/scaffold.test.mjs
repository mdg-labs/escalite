import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

test('app.json configures escalite deep link scheme', () => {
  const appJson = JSON.parse(readFileSync(join(root, 'app.json'), 'utf8'))
  assert.equal(appJson.expo.scheme, 'escalite')
  assert.equal(appJson.expo.userInterfaceStyle, 'dark')
})

test('app.json declares iOS Critical Alerts entitlement pending Apple approval', () => {
  const appJson = JSON.parse(readFileSync(join(root, 'app.json'), 'utf8'))
  assert.equal(appJson.expo.ios.bundleIdentifier, 'dev.mdglabs.escalite')
  assert.equal(
    appJson.expo.ios.entitlements['com.apple.developer.usernotifications.critical-alerts'],
    true,
  )

  const notificationsPlugin = appJson.expo.plugins.find(
    (plugin) => Array.isArray(plugin) && plugin[0] === 'expo-notifications',
  )
  assert.ok(notificationsPlugin)
  assert.equal(notificationsPlugin[1].enableBackgroundRemoteNotifications, true)
})

test('eas.json stub defines build profiles', () => {
  const easJson = JSON.parse(readFileSync(join(root, 'eas.json'), 'utf8'))
  assert.ok(easJson.build)
  assert.ok(easJson.build.development)
  assert.ok(easJson.build.preview)
  assert.ok(easJson.build.production)
})

test('tamagui config defaults to dark theme from tokens', () => {
  const configSource = readFileSync(join(root, 'src/theme/tamagui.config.ts'), 'utf8')
  const tokensSource = readFileSync(join(root, 'src/theme/tokens/index.ts'), 'utf8')

  assert.match(configSource, /defaultTheme:\s*DEFAULT_THEME/)
  assert.match(configSource, /@escalite\/tokens/)
  assert.match(tokensSource, /DEFAULT_THEME\s*=\s*'dark'/)
  assert.match(tokensSource, /severityCritical/)
})
