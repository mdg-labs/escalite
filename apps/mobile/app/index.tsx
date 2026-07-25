import { Paragraph, Spinner, YStack } from 'tamagui'

export default function HomeScreen() {
  return (
    <YStack flex={1} alignItems="center" justifyContent="center" backgroundColor="$background" padding="$4" gap="$3">
      <Spinner size="large" color="$color" />
      <Paragraph color="$color" textAlign="center">
        Escalite mobile — on-call alerts, ack, and escalate.
      </Paragraph>
    </YStack>
  )
}
