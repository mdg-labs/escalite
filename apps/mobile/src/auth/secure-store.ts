import * as SecureStore from 'expo-secure-store'

import { mobileAuthConstants } from '@/auth/config'

export async function getStoredRefreshToken(): Promise<string | null> {
  return SecureStore.getItemAsync(mobileAuthConstants.refreshTokenStorageKey)
}

export async function setStoredRefreshToken(token: string): Promise<void> {
  await SecureStore.setItemAsync(mobileAuthConstants.refreshTokenStorageKey, token, {
    keychainAccessible: SecureStore.WHEN_UNLOCKED_THIS_DEVICE_ONLY,
  })
}

export async function clearStoredRefreshToken(): Promise<void> {
  await SecureStore.deleteItemAsync(mobileAuthConstants.refreshTokenStorageKey)
}
