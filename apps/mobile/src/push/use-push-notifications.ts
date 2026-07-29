import type * as Notifications from 'expo-notifications'
import { useEffect, useRef } from 'react'

import { isExpoGo, logExpoGoPushSkipped } from '@/push/expo-go'

export function usePushNotifications(): void {
  const handledNotificationIds = useRef(new Set<string>())

  useEffect(() => {
    if (isExpoGo()) {
      logExpoGoPushSkipped('push notification bootstrap')
      return
    }

    let cancelled = false
    let cleanup: (() => void) | undefined

    void (async () => {
      const Notifications = await import('expo-notifications')
      const { handleNotificationActionResponse } = await import('@/push/action-handler')
      const {
        ensureAndroidNotificationChannel,
        initNotificationHandler,
        navigateToAlertDetailFromNotification,
      } = await import('@/push/notifications')

      if (cancelled) {
        return
      }

      await initNotificationHandler()
      await ensureAndroidNotificationChannel()

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
      cleanup = () => {
        subscription.remove()
      }
    })()

    return () => {
      cancelled = true
      cleanup?.()
    }
  }, [])
}
