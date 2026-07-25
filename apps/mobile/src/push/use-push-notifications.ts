import * as Notifications from 'expo-notifications'
import { useEffect, useRef } from 'react'

import { handleNotificationActionResponse } from '@/push/action-handler'
import {
  ensureAndroidNotificationChannel,
  navigateToAlertDetailFromNotification,
} from '@/push/notifications'

export function usePushNotifications(): void {
  const handledNotificationIds = useRef(new Set<string>())

  useEffect(() => {
    void ensureAndroidNotificationChannel()

    const handleResponse = (response: Notifications.NotificationResponse) => {
      const responseKey = `${response.notification.request.identifier}:${response.actionIdentifier}`
      if (handledNotificationIds.current.has(responseKey)) {
        return
      }

      void handleNotificationActionResponse(response).then((result) => {
        if (result === 'ignored') {
          return
        }

        handledNotificationIds.current.add(responseKey)

        if (result === 'navigate' && navigateToAlertDetailFromNotification(response)) {
          return
        }
      })
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
