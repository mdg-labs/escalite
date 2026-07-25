import * as Device from 'expo-device'
import * as Notifications from 'expo-notifications'

import { mobileAuthConfig } from '@/auth/config'

type RegisterMobileDeviceResponse = {
  data?: {
    registerMobileDevice?: {
      id: string
    }
  }
  errors?: Array<{ message: string }>
}

export async function obtainExpoPushToken(): Promise<string | null> {
  if (!Device.isDevice) {
    return null
  }

  const { status: existingStatus } = await Notifications.getPermissionsAsync()
  let finalStatus = existingStatus
  if (existingStatus !== 'granted') {
    const { status } = await Notifications.requestPermissionsAsync()
    finalStatus = status
  }

  if (finalStatus !== 'granted') {
    return null
  }

  const token = await Notifications.getExpoPushTokenAsync()
  return token.data
}

export async function registerMobileDevicePushToken(
  refreshToken: string,
  expoPushToken: string,
): Promise<void> {
  const platform = Device.osName?.toLowerCase() ?? undefined
  const deviceLabel = Device.modelName ?? undefined

  const response = await fetch(`${mobileAuthConfig.apiBaseUrl}/graphql`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${refreshToken}`,
    },
    body: JSON.stringify({
      query: `mutation RegisterMobileDevice($input: RegisterMobileDeviceInput!) {
        registerMobileDevice(input: $input) {
          id
        }
      }`,
      variables: {
        input: {
          expoPushToken,
          platform,
          deviceLabel,
        },
      },
    }),
  })

  if (!response.ok) {
    throw new Error('device registration request failed')
  }

  const body = (await response.json()) as RegisterMobileDeviceResponse
  if (body.errors?.length) {
    throw new Error(body.errors[0]?.message ?? 'device registration failed')
  }
}

export async function syncMobileDevicePushToken(refreshToken: string): Promise<void> {
  const expoPushToken = await obtainExpoPushToken()
  if (!expoPushToken) {
    return
  }

  await registerMobileDevicePushToken(refreshToken, expoPushToken)
}
