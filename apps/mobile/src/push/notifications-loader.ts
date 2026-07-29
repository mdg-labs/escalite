type NotificationsModule = typeof import('expo-notifications')

let notificationsModule: NotificationsModule | null = null

export async function loadNotificationsModule(): Promise<NotificationsModule> {
  if (!notificationsModule) {
    notificationsModule = await import('expo-notifications')
  }
  return notificationsModule
}
