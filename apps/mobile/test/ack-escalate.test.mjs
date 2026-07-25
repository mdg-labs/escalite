import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

test('alert api exposes acknowledge, snooze, and re-escalate mutations', () => {
  const source = readFileSync(join(root, 'src/api/alerts.ts'), 'utf8')

  assert.match(source, /acknowledgeAlert/)
  assert.match(source, /snoozeAlert/)
  assert.match(source, /reEscalateAlert/)
  assert.match(source, /mutation AcknowledgeAlert/)
  assert.match(source, /mutation SnoozeAlert/)
  assert.match(source, /mutation ReEscalateAlert/)
})

test('escalation targets provide a minimal selection list', () => {
  const source = readFileSync(join(root, 'src/alerts/escalation-targets.ts'), 'utf8')

  assert.match(source, /ESCALATION_TARGETS/)
  assert.match(source, /re_escalate/)
  assert.match(source, /snooze_15/)
  assert.match(source, /snooze_30/)
  assert.match(source, /snooze_60/)
})

test('notification categories register ack and escalate actions', () => {
  const source = readFileSync(join(root, 'src/push/categories.ts'), 'utf8')

  assert.match(source, /setNotificationCategoryAsync/)
  assert.match(source, /ALERT_TRIGGERED_TYPE/)
  assert.match(source, /ALERT_NOTIFICATION_CATEGORY/)
  assert.match(source, /opensAppToForeground:\s*false/)
  assert.match(source, /identifier:\s*NOTIFICATION_ACTION_ACK/)
  assert.match(source, /identifier:\s*NOTIFICATION_ACTION_ESCALATE/)
})

test('notification action handler acknowledges without navigation', () => {
  const source = readFileSync(join(root, 'src/push/action-handler.ts'), 'utf8')

  assert.match(source, /handleNotificationActionResponse/)
  assert.match(source, /NOTIFICATION_ACTION_ACK/)
  assert.match(source, /await acknowledgeAlert\(alertId\)/)
  assert.match(source, /NOTIFICATION_ACTION_ESCALATE/)
  assert.match(source, /router\.push\(`\/alerts\/\$\{alertId\}\?escalate=1`\)/)
})

test('push hook routes notification action responses', () => {
  const source = readFileSync(join(root, 'src/push/use-push-notifications.ts'), 'utf8')

  assert.match(source, /handleNotificationActionResponse/)
  assert.match(source, /actionIdentifier/)
})

test('alert detail screen renders in-app ack and escalate actions', () => {
  const source = readFileSync(join(root, 'app/alerts/[alertId].tsx'), 'utf8')

  assert.match(source, /AlertActionBar/)
  assert.match(source, /escalateOpen/)
})

test('alert action bar uses AlertDialog and Sheet for ack/escalate', () => {
  const source = readFileSync(join(root, 'src/components/alert-action-bar.tsx'), 'utf8')

  assert.match(source, /AlertDialog/)
  assert.match(source, /Sheet\.ScrollView/)
  assert.match(source, /ESCALATION_TARGETS/)
  assert.match(source, /Acknowledge alert\?/)
})
