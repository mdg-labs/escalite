import Constants from 'expo-constants'

export function isExpoGo(): boolean {
  return Constants.appOwnership === 'expo'
}

export function logExpoGoPushSkipped(feature: string): void {
  if (__DEV__) {
    console.log(
      `[push] Skipping ${feature} in Expo Go — use a development build (eas build / expo run:android) for push notifications`,
    )
  }
}
