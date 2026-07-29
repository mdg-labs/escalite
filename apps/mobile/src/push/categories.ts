import { ALERT_TRIGGERED_TYPE } from '@/push/payload'
import { loadNotificationsModule } from '@/push/notifications-loader'

export const ALERT_NOTIFICATION_CATEGORY = ALERT_TRIGGERED_TYPE

export const NOTIFICATION_ACTION_ACK = 'ack' as const
export const NOTIFICATION_ACTION_ESCALATE = 'escalate' as const

let categoriesRegistered = false

export async function ensureAlertNotificationCategories(): Promise<void> {
  if (categoriesRegistered) {
    return
  }

  const Notifications = await loadNotificationsModule()

  await Notifications.setNotificationCategoryAsync(ALERT_NOTIFICATION_CATEGORY, [
    {
      identifier: NOTIFICATION_ACTION_ACK,
      buttonTitle: 'Acknowledge',
      options: {
        opensAppToForeground: false,
      },
    },
    {
      identifier: NOTIFICATION_ACTION_ESCALATE,
      buttonTitle: 'Escalate',
      options: {
        opensAppToForeground: true,
      },
    },
  ])

  categoriesRegistered = true
}
