import * as Notifications from 'expo-notifications'
import { router } from 'expo-router'

import { acknowledgeAlert } from '@/api/alerts'
import {
  NOTIFICATION_ACTION_ACK,
  NOTIFICATION_ACTION_ESCALATE,
} from '@/push/categories'
import { extractAlertIdFromNotification } from '@/push/notifications'

export type NotificationActionResult = 'handled' | 'navigate' | 'ignored'

export async function handleNotificationActionResponse(
  response: Notifications.NotificationResponse,
): Promise<NotificationActionResult> {
  const actionId = response.actionIdentifier
  if (actionId === Notifications.DEFAULT_ACTION_IDENTIFIER) {
    return 'navigate'
  }

  const alertId = extractAlertIdFromNotification(response.notification)
  if (!alertId) {
    return 'ignored'
  }

  if (actionId === NOTIFICATION_ACTION_ACK) {
    try {
      await acknowledgeAlert(alertId)
    } catch {
      return 'ignored'
    }
    return 'handled'
  }

  if (actionId === NOTIFICATION_ACTION_ESCALATE) {
    router.push(`/alerts/${alertId}?escalate=1`)
    return 'handled'
  }

  return 'ignored'
}
