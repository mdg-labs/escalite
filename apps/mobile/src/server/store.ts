import * as SecureStore from 'expo-secure-store'

const serverOriginStorageKey = 'escalite.mobile.server_origin'

export async function getStoredServerOrigin(): Promise<string | null> {
  return SecureStore.getItemAsync(serverOriginStorageKey)
}

export async function setStoredServerOrigin(origin: string): Promise<void> {
  await SecureStore.setItemAsync(serverOriginStorageKey, origin, {
    keychainAccessible: SecureStore.WHEN_UNLOCKED_THIS_DEVICE_ONLY,
  })
}

export async function clearStoredServerOrigin(): Promise<void> {
  await SecureStore.deleteItemAsync(serverOriginStorageKey)
}
