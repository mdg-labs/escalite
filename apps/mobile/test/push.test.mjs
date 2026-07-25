import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

test('push registration uses bearer GraphQL mutation', () => {
  const source = readFileSync(join(root, 'src/push/register-device.ts'), 'utf8')
  assert.match(source, /registerMobileDevice/)
  assert.match(source, /Authorization: `Bearer \$\{refreshToken\}`/)
  assert.match(source, /expo-notifications/)
})

test('auth context syncs push token after sign-in', () => {
  const source = readFileSync(join(root, 'src/auth/context.tsx'), 'utf8')
  assert.match(source, /syncMobileDevicePushToken/)
})

test('push payload parser handles alert.triggered contract', () => {
  const source = readFileSync(join(root, 'src/push/payload.ts'), 'utf8')
  assert.match(source, /alert\.triggered/)
  assert.match(source, /alertId/)
  assert.match(source, /parseAlertTriggeredPushData/)
})

test('push registration requests critical alert permission on iOS', () => {
  const source = readFileSync(join(root, 'src/push/register-device.ts'), 'utf8')
  assert.match(source, /allowCriticalAlerts/)
})

test('push payload parser handles critical flag', () => {
  const source = readFileSync(join(root, 'src/push/payload.ts'), 'utf8')
  assert.match(source, /critical\?: boolean/)
})

test('critical alerts helper falls back to time-sensitive', () => {
  const source = readFileSync(join(root, 'src/push/critical-alerts.ts'), 'utf8')
  assert.match(source, /resolveInterruptionLevel/)
  assert.match(source, /hasCriticalAlertsPermission/)
  assert.match(source, /timeSensitive/)
})

test('android notification channels include critical alerts channel', () => {
  const source = readFileSync(join(root, 'src/push/notifications.ts'), 'utf8')
  assert.match(source, /ALERTS_CRITICAL_CHANNEL_ID/)
  assert.match(source, /AndroidImportance\.MAX/)
  assert.match(source, /bypassDnd: true/)
})

test('app.json declares Android full-screen intent permission', () => {
  const appJson = JSON.parse(readFileSync(join(root, 'app.json'), 'utf8'))
  assert.ok(appJson.expo.android.permissions.includes('android.permission.USE_FULL_SCREEN_INTENT'))
})

test('foreground notification handler shows title and body', () => {
  const source = readFileSync(join(root, 'src/push/notifications.ts'), 'utf8')
  assert.match(source, /setNotificationHandler/)
  assert.match(source, /shouldShowAlert/)
  assert.match(source, /shouldShowBanner/)
  assert.match(source, /navigateToAlertDetailFromNotification/)
  assert.match(source, /router\.push\(`\/alerts\/\$\{alertId\}`\)/)
})

test('notification tap navigates to alert detail route', () => {
  const layoutSource = readFileSync(join(root, 'app/_layout.tsx'), 'utf8')
  const hookSource = readFileSync(join(root, 'src/push/use-push-notifications.ts'), 'utf8')
  const detailSource = readFileSync(join(root, 'app/alerts/[alertId].tsx'), 'utf8')
  const notificationsSource = readFileSync(join(root, 'src/push/notifications.ts'), 'utf8')

  assert.match(layoutSource, /PushNotificationBootstrap/)
  assert.match(hookSource, /addNotificationResponseReceivedListener/)
  assert.match(hookSource, /getLastNotificationResponseAsync/)
  assert.match(hookSource, /handleNotificationActionResponse/)
  assert.match(detailSource, /fetchAlert/)
  assert.match(notificationsSource, /ensureAlertNotificationCategories/)
})
