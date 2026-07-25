import * as Notifications from 'expo-notifications'
import { router } from 'expo-router'
import { Platform } from 'react-native'

import { parseAlertTriggeredPushData } from '@/push/payload'

Notifications.setNotificationHandler({
  handleNotification: async (notification) => {
    const data = parseAlertTriggeredPushData(notification.request.content.data)
    const hasDisplayContent =
      Boolean(notification.request.content.title) ||
      Boolean(notification.request.content.body) ||
      Boolean(data?.title) ||
      Boolean(data?.body)

    return {
      shouldShowAlert: hasDisplayContent,
      shouldPlaySound: true,
      shouldSetBadge: true,
      shouldShowBanner: hasDisplayContent,
      shouldShowList: hasDisplayContent,
    }
  },
})

export async function ensureAndroidNotificationChannel(): Promise<void> {
  if (Platform.OS !== 'android') {
    return
  }

  await Notifications.setNotificationChannelAsync('alerts', {
    name: 'Alerts',
    importance: Notifications.AndroidImportance.HIGH,
    vibrationPattern: [0, 250, 250, 250],
    lightColor: '#f5c842',
  })
}

export function extractAlertIdFromNotification(
  notification: Notifications.Notification,
): string | null {
  const data = parseAlertTriggeredPushData(notification.request.content.data)
  return data?.alertId ?? null
}

export function navigateToAlertDetailFromNotification(
  response: Notifications.NotificationResponse,
): boolean {
  const alertId = extractAlertIdFromNotification(response.notification)
  if (!alertId) {
    return false
  }

  router.push(`/alerts/${alertId}`)
  return true
}
