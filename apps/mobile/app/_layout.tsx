import { StatusBar } from 'expo-status-bar'
import { Spinner, TamaguiProvider, Theme, YStack } from 'tamagui'

import { HeaderMenuButton, NavigationShellProvider } from '@/components/navigation-shell'
import { ServerSetupScreen } from '@/screens/server-setup-screen'
import { PushNotificationBootstrap } from '@/push/push-notification-bootstrap'
import { AppProviders } from '@/providers/app-providers'
import { ServerConfigProvider, useServerConfig } from '@/server/context'
import { DEFAULT_THEME, tamaguiConfig } from '@/theme/tamagui.config'
import { Stack } from 'expo-router'

const rootScreenOptions = {
  headerBackVisible: false,
  gestureEnabled: false,
} as const

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
      <>
        <StatusBar style="light" />
        <ServerSetupScreen />
      </>
    )
  }

  return (
    <AppProviders>
      <PushNotificationBootstrap>
        <NavigationShellProvider>
          <StatusBar style="light" />
          <Stack
            screenOptions={{
              headerStyle: { backgroundColor: '#0a0a0b' },
              headerTintColor: '#f5f5f5',
              contentStyle: { backgroundColor: '#0a0a0b' },
            }}
          >
            <Stack.Screen
              name="index"
              options={{
                ...rootScreenOptions,
                title: 'Escalite',
                headerLeft: () => <HeaderMenuButton />,
              }}
            />
            <Stack.Screen
              name="auth"
              options={{
                ...rootScreenOptions,
                headerShown: false,
                title: 'Signing in',
              }}
            />
            <Stack.Screen
              name="setup"
              options={{
                ...rootScreenOptions,
                headerShown: false,
              }}
            />
            <Stack.Screen name="alerts/[alertId]" options={{ title: 'Alert' }} />
          </Stack>
        </NavigationShellProvider>
      </PushNotificationBootstrap>
    </AppProviders>
  )
}

export default function RootLayout() {
  return (
    <TamaguiProvider config={tamaguiConfig} defaultTheme={DEFAULT_THEME}>
      <Theme name={DEFAULT_THEME}>
        <ServerConfigProvider>
          <RootNavigation />
        </ServerConfigProvider>
      </Theme>
    </TamaguiProvider>
  )
}
