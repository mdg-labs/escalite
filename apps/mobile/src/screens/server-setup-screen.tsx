import { useState } from 'react'
import { Image } from 'react-native'
import { Button, Input, Paragraph, Text, YStack } from 'tamagui'

import { escaliteLogo } from '@/assets/images'
import { useServerConfig } from '@/server/context'

export function ServerSetupScreen() {
  const { setServerOrigin } = useServerConfig()
  const [serverUrl, setServerUrl] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSave(): Promise<void> {
    setSubmitting(true)
    setError(null)

    try {
      await setServerOrigin(serverUrl)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to connect to server')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <YStack flex={1} backgroundColor="$background" padding="$4" gap="$4" justifyContent="center">
      <YStack alignItems="center" gap="$3">
        <Image
          accessibilityLabel="Escalite"
          source={escaliteLogo}
          style={{ width: 72, height: 72 }}
        />
        <Text color="$color" fontSize="$8" fontWeight="700" textAlign="center">
          Connect to Escalite
        </Text>
      </YStack>
      <Paragraph color="$color" textAlign="center">
        Enter the public URL of your Escalite instance, such as https://escalite.example.com.
      </Paragraph>
      <Input
        autoCapitalize="none"
        autoCorrect={false}
        keyboardType="url"
        placeholder="https://escalite.example.com"
        value={serverUrl}
        onChangeText={setServerUrl}
      />
      {error ? (
        <Paragraph color="$red10" textAlign="center" role="alert">
          {error}
        </Paragraph>
      ) : null}
      <Button disabled={submitting || serverUrl.trim().length === 0} onPress={() => void handleSave()}>
        {submitting ? 'Connecting…' : 'Save and continue'}
      </Button>
    </YStack>
  )
}
