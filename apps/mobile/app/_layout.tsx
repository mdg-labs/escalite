import { StatusBar } from 'expo-status-bar'
import { Spinner, TamaguiProvider, Theme, YStack } from 'tamagui'

import { ServerSetupScreen } from '@/screens/server-setup-screen'
import { PushNotificationBootstrap } from '@/push/push-notification-bootstrap'
import { AppProviders } from '@/providers/app-providers'
import { ServerConfigProvider, useServerConfig } from '@/server/context'
import { DEFAULT_THEME, tamaguiConfig } from '@/theme/tamagui.config'
import { Stack } from 'expo-router'

function RootNavigation() {
  const { status } = useServerConfig()

  if (status === 'loading') {
    return (
      <YStack flex={1} alignItems="center" justifyContent="center" backgroundColor="$background">
        <Spinner size="large" color="$color" />
      </YStack>
    )
  }

  if (status === 'unconfigured') {
    return (
      <TamaguiProvider config={tamaguiConfig} defaultTheme={DEFAULT_THEME}>
        <Theme name={DEFAULT_THEME}>
          <StatusBar style="light" />
          <ServerSetupScreen />
        </Theme>
      </TamaguiProvider>
    )
  }

  return (
    <AppProviders>
      <PushNotificationBootstrap>
        <StatusBar style="light" />
        <Stack
          screenOptions={{
            headerStyle: { backgroundColor: '#0a0a0b' },
            headerTintColor: '#f5f5f5',
            contentStyle: { backgroundColor: '#0a0a0b' },
          }}
        >
          <Stack.Screen name="index" options={{ title: 'Escalite' }} />
          <Stack.Screen name="alerts/[alertId]" options={{ title: 'Alert' }} />
        </Stack>
      </PushNotificationBootstrap>
    </AppProviders>
  )
}

export default function RootLayout() {
  return (
    <ServerConfigProvider>
      <RootNavigation />
    </ServerConfigProvider>
  )
}
