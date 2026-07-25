import { Stack } from 'expo-router'
import { StatusBar } from 'expo-status-bar'

import { AppProviders } from '@/providers/app-providers'

export default function RootLayout() {
  return (
    <AppProviders>
      <StatusBar style="light" />
      <Stack
        screenOptions={{
          headerStyle: { backgroundColor: '#0a0a0b' },
          headerTintColor: '#f5f5f5',
          contentStyle: { backgroundColor: '#0a0a0b' },
        }}
      />
    </AppProviders>
  )
}
