import * as Notifications from 'expo-notifications'
import { useEffect, useRef } from 'react'

import {
  ensureAndroidNotificationChannel,
  navigateToAlertDetailFromNotification,
} from '@/push/notifications'

export function usePushNotifications(): void {
  const handledNotificationIds = useRef(new Set<string>())

  useEffect(() => {
    void ensureAndroidNotificationChannel()

    const handleResponse = (response: Notifications.NotificationResponse) => {
      const notificationId = response.notification.request.identifier
      if (handledNotificationIds.current.has(notificationId)) {
        return
      }

      if (navigateToAlertDetailFromNotification(response)) {
        handledNotificationIds.current.add(notificationId)
      }
    }

    void Notifications.getLastNotificationResponseAsync().then((response) => {
      if (response) {
        handleResponse(response)
      }
    })

    const subscription = Notifications.addNotificationResponseReceivedListener(handleResponse)
    return () => {
      subscription.remove()
    }
  }, [])
}
