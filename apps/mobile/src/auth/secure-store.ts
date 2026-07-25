import * as SecureStore from 'expo-secure-store'

import { mobileAuthConfig } from '@/auth/config'

export async function getStoredRefreshToken(): Promise<string | null> {
  return SecureStore.getItemAsync(mobileAuthConfig.refreshTokenStorageKey)
}

export async function setStoredRefreshToken(token: string): Promise<void> {
  await SecureStore.setItemAsync(mobileAuthConfig.refreshTokenStorageKey, token, {
    keychainAccessible: SecureStore.WHEN_UNLOCKED_THIS_DEVICE_ONLY,
  })
}

export async function clearStoredRefreshToken(): Promise<void> {
  await SecureStore.deleteItemAsync(mobileAuthConfig.refreshTokenStorageKey)
}
