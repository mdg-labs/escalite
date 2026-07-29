import { Button, Paragraph, Spinner, Text, YStack } from 'tamagui'
import { RefreshControl } from 'react-native'
import { ScrollView } from 'tamagui'

import { useAuth } from '@/auth/context'
import { OnCallStatus } from '@/components/on-call-status'
import { useMyOnCallStatus } from '@/api/use-my-on-call-status'
import { useServerConfig } from '@/server/context'

export default function HomeScreen() {
  const { status, error, signIn, signOut } = useAuth()
  const { endpoints, clearServer } = useServerConfig()
  const { assignments, loading, refreshing, error: onCallError, refresh } = useMyOnCallStatus(
    status === 'authenticated',
  )

  async function handleChangeServer(): Promise<void> {
    await signOut()
    await clearServer()
  }

  if (status === 'loading') {
    return (
      <YStack flex={1} alignItems="center" justifyContent="center" backgroundColor="$background" padding="$4" gap="$3">
        <Spinner size="large" color="$color" />
        <Paragraph color="$color" textAlign="center">
          Restoring session…
        </Paragraph>
      </YStack>
    )
  }

  if (status === 'authenticated') {
    return (
      <ScrollView
        flex={1}
        backgroundColor="$background"
        contentContainerStyle={{ flexGrow: 1, padding: 16 }}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void refresh()} />}
      >
        <YStack flex={1} justifyContent="center" gap="$4">
          <OnCallStatus assignments={assignments} loading={loading} error={onCallError} />
        </YStack>
      </ScrollView>
    )
  }

  return (
    <YStack flex={1} alignItems="center" justifyContent="center" backgroundColor="$background" padding="$4" gap="$4">
      <Spinner size="large" color="$color" />
      <Text color="$color" fontSize="$6" fontWeight="600" textAlign="center">
        Sign in to Escalite
      </Text>
      <Paragraph color="$color" textAlign="center">
        Continue in your browser to authenticate, then return here automatically.
      </Paragraph>
      {endpoints ? (
        <Paragraph color="$color" textAlign="center">
          Server: {endpoints.origin}
        </Paragraph>
      ) : null}
      {error ? (
        <Paragraph color="$red10" textAlign="center" role="alert">
          {error}
        </Paragraph>
      ) : null}
      <Button onPress={() => void signIn()}>Continue in browser</Button>
      <Button chromeless onPress={() => void handleChangeServer()}>
        Change server
      </Button>
    </YStack>
  )
}
