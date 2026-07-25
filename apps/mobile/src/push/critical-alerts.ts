import * as Notifications from 'expo-notifications'
import { Platform } from 'react-native'

export type AlertInterruptionLevel = 'critical' | 'timeSensitive'

export async function hasCriticalAlertsPermission(): Promise<boolean> {
  if (Platform.OS !== 'ios') {
    return true
  }

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
