import type * as Notifications from 'expo-notifications'
import { router } from 'expo-router'
import { Platform } from 'react-native'

import { ensureAlertNotificationCategories } from '@/push/categories'
import { hasCriticalAlertsPermission, resolveInterruptionLevel } from '@/push/critical-alerts'
import { loadNotificationsModule } from '@/push/notifications-loader'
import { parseAlertTriggeredPushData } from '@/push/payload'

export const ALERTS_CHANNEL_ID = 'alerts'
export const ALERTS_CRITICAL_CHANNEL_ID = 'alerts-critical'

let notificationHandlerInitialized = false

export async function initNotificationHandler(): Promise<void> {
  if (notificationHandlerInitialized) {
    return
  }

  const Notifications = await loadNotificationsModule()

  Notifications.setNotificationHandler({
    handleNotification: async (notification) => {
      const data = parseAlertTriggeredPushData(notification.request.content.data)
      const hasDisplayContent =
        Boolean(notification.request.content.title) ||
        Boolean(notification.request.content.body) ||
        Boolean(data?.title) ||
        Boolean(data?.body)

      const hasCriticalPermission = await hasCriticalAlertsPermission()
      const interruptionLevel = resolveInterruptionLevel(data?.critical === true, hasCriticalPermission)

      return {
        shouldShowAlert: hasDisplayContent,
        shouldPlaySound: true,
        shouldSetBadge: true,
        shouldShowBanner: hasDisplayContent,
        shouldShowList: hasDisplayContent,
        priority: interruptionLevel === 'critical'
          ? Notifications.AndroidNotificationPriority.MAX
          : Notifications.AndroidNotificationPriority.HIGH,
      }
    },
  })

  notificationHandlerInitialized = true
}

export async function ensureAndroidNotificationChannel(): Promise<void> {
  await ensureAlertNotificationCategories()

  if (Platform.OS !== 'android') {
    return
  }

  const Notifications = await loadNotificationsModule()

  await Notifications.setNotificationChannelAsync(ALERTS_CHANNEL_ID, {
    name: 'Alerts',
    importance: Notifications.AndroidImportance.HIGH,
    vibrationPattern: [0, 250, 250, 250],
    lightColor: '#f5c842',
  })

  await Notifications.setNotificationChannelAsync(ALERTS_CRITICAL_CHANNEL_ID, {
    name: 'Critical alerts',
    importance: Notifications.AndroidImportance.MAX,
    bypassDnd: true,
    vibrationPattern: [0, 500, 250, 500],
    lightColor: '#f5c842',
    lockscreenVisibility: Notifications.AndroidNotificationVisibility.PUBLIC,
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
