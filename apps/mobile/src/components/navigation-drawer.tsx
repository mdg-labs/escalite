import { Button, Paragraph, Separator, Sheet, Text, YStack } from 'tamagui'

type NavigationDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  email: string | null
  serverOrigin: string | null
  onSignOut: () => Promise<void>
  onChangeServer: () => Promise<void>
}

export function NavigationDrawer({
  open,
  onOpenChange,
  email,
  serverOrigin,
  onSignOut,
  onChangeServer,
}: NavigationDrawerProps) {
  return (
    <Sheet
      modal
      open={open}
      onOpenChange={onOpenChange}
      snapPointsMode="fit"
      animation="medium"
      dismissOnOverlayPress
      zIndex={100_000}
    >
      <Sheet.Overlay
        animation="lazy"
        enterStyle={{ opacity: 0 }}
        exitStyle={{ opacity: 0 }}
      />
      <Sheet.Frame
        padding="$4"
        gap="$4"
        backgroundColor="$background"
        position="absolute"
        left={0}
        top={0}
        bottom={0}
        width={300}
        maxWidth="85%"
        borderTopRightRadius="$4"
        borderBottomRightRadius="$4"
      >
        <Sheet.ScrollView>
          <YStack gap="$4" paddingTop="$6">
            {email ? (
              <Text color="$color" fontSize="$6" fontWeight="700">
                {email}
              </Text>
            ) : null}
            {serverOrigin ? (
              <Paragraph color="$colorMuted">{serverOrigin}</Paragraph>
            ) : null}
            <Separator />
            <Button onPress={() => void onSignOut()}>Sign out</Button>
            <Button chromeless onPress={() => void onChangeServer()}>
              Change server
            </Button>
          </YStack>
        </Sheet.ScrollView>
      </Sheet.Frame>
    </Sheet>
  )
}
