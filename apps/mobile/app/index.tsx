import { Button, Paragraph, Spinner, Text, YStack } from 'tamagui'

import { useAuth } from '@/auth/context'

export default function HomeScreen() {
  const { status, user, error, signIn, signOut } = useAuth()

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

  if (status === 'authenticated' && user) {
    return (
      <YStack flex={1} alignItems="center" justifyContent="center" backgroundColor="$background" padding="$4" gap="$4">
        <Text color="$color" fontSize="$6" fontWeight="600" textAlign="center">
          Signed in
        </Text>
        <Paragraph color="$color" textAlign="center">
          {user.email}
        </Paragraph>
        <Button onPress={() => void signOut()}>Sign out</Button>
      </YStack>
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
      {error ? (
        <Paragraph color="$red10" textAlign="center" role="alert">
          {error}
        </Paragraph>
      ) : null}
      <Button onPress={() => void signIn()}>Continue in browser</Button>
    </YStack>
  )
}
