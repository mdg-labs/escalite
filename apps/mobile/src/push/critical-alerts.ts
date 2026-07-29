import { Platform } from 'react-native'

import { loadNotificationsModule } from '@/push/notifications-loader'

export type AlertInterruptionLevel = 'critical' | 'timeSensitive'

export async function hasCriticalAlertsPermission(): Promise<boolean> {
  if (Platform.OS !== 'ios') {
    return true
  }

  const Notifications = await loadNotificationsModule()
  const permissions = await Notifications.getPermissionsAsync()
  return permissions.ios?.allowsCriticalAlerts === true
}

export function resolveInterruptionLevel(
  critical: boolean,
  hasCriticalPermission: boolean,
): AlertInterruptionLevel {
  if (critical && hasCriticalPermission) {
    return 'critical'
  }

  return 'timeSensitive'
}
