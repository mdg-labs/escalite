import { useEffect, useState } from 'react'
import {
  AlertDialog,
  Button,
  ListItem,
  Paragraph,
  Sheet,
  Spinner,
  Text,
  XStack,
  YStack,
} from 'tamagui'

import {
  acknowledgeAlert,
  canAcknowledgeAlert,
  canEscalateAlert,
  reEscalateAlert,
  snoozeAlert,
  type MobileAlert,
} from '@/api/alerts'
import {
  ESCALATION_TARGETS,
  type EscalationTarget,
} from '@/alerts/escalation-targets'

type AlertActionBarProps = {
  alert: MobileAlert
  escalateOpen?: boolean
  onAlertUpdated: (alert: MobileAlert) => void
  onEscalateOpenChange?: (open: boolean) => void
}

export function AlertActionBar({
  alert,
  escalateOpen = false,
  onAlertUpdated,
  onEscalateOpenChange,
}: AlertActionBarProps) {
  const [ackOpen, setAckOpen] = useState(false)
  const [escalateSheetOpen, setEscalateSheetOpen] = useState(escalateOpen)
  const [actionError, setActionError] = useState<string | null>(null)
  const [ackLoading, setAckLoading] = useState(false)
  const [escalateLoading, setEscalateLoading] = useState(false)

  useEffect(() => {
    setEscalateSheetOpen(escalateOpen)
  }, [escalateOpen])

  const setEscalateOpen = (open: boolean) => {
    setEscalateSheetOpen(open)
    onEscalateOpenChange?.(open)
  }

  async function runAcknowledge(): Promise<void> {
    setActionError(null)
    setAckLoading(true)
    try {
      const updated = await acknowledgeAlert(alert.id)
      onAlertUpdated(updated)
      setAckOpen(false)
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Unable to acknowledge alert')
    } finally {
      setAckLoading(false)
    }
  }

  async function runEscalationTarget(target: EscalationTarget): Promise<void> {
    setActionError(null)
    setEscalateLoading(true)
    try {
      const updated =
        target.kind === 're_escalate'
          ? await reEscalateAlert(alert.id)
          : await snoozeAlert(alert.id, target.durationMinutes ?? 0)
      onAlertUpdated(updated)
      setEscalateOpen(false)
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Unable to escalate alert')
    } finally {
      setEscalateLoading(false)
    }
  }

  const showAck = canAcknowledgeAlert(alert)
  const showEscalate = canEscalateAlert(alert)

  if (!showAck && !showEscalate) {
    return null
  }

  return (
    <>
      <XStack gap="$3" flexWrap="wrap">
        {showAck ? (
          <Button
            flex={1}
            minWidth={140}
            theme="active"
            onPress={() => {
              setActionError(null)
              setAckOpen(true)
            }}
          >
            Acknowledge
          </Button>
        ) : null}
        {showEscalate ? (
          <Button
            flex={1}
            minWidth={140}
            variant="outlined"
            onPress={() => {
              setActionError(null)
              setEscalateOpen(true)
            }}
          >
            Escalate
          </Button>
        ) : null}
      </XStack>

      {actionError ? (
        <Paragraph color="$red10" role="alert">
          {actionError}
        </Paragraph>
      ) : null}

      <AlertDialog open={ackOpen} onOpenChange={setAckOpen}>
        <AlertDialog.Portal>
          <AlertDialog.Overlay
            key="overlay"
            animation="quick"
            opacity={0.5}
            enterStyle={{ opacity: 0 }}
            exitStyle={{ opacity: 0 }}
          />
          <AlertDialog.Content
            bordered
            elevate
            key="content"
            animation={[
              'quick',
              {
                opacity: {
                  overshootClamping: true,
                },
              },
            ]}
            enterStyle={{ x: 0, y: -20, opacity: 0, scale: 0.9 }}
            exitStyle={{ x: 0, y: 10, opacity: 0, scale: 0.95 }}
            gap="$4"
          >
            <YStack gap="$2">
              <AlertDialog.Title>Acknowledge alert?</AlertDialog.Title>
              <AlertDialog.Description>
                Acknowledging stops escalation but does not close the alert.
              </AlertDialog.Description>
            </YStack>
            <XStack gap="$3" justifyContent="flex-end">
              <AlertDialog.Cancel asChild>
                <Button variant="outlined">Cancel</Button>
              </AlertDialog.Cancel>
              <AlertDialog.Action asChild>
                <Button theme="active" disabled={ackLoading} onPress={() => void runAcknowledge()}>
                  {ackLoading ? <Spinner size="small" color="$color" /> : 'Acknowledge'}
                </Button>
              </AlertDialog.Action>
            </XStack>
          </AlertDialog.Content>
        </AlertDialog.Portal>
      </AlertDialog>

      <Sheet
        modal
        open={escalateSheetOpen}
        onOpenChange={setEscalateOpen}
        snapPoints={[55]}
        dismissOnSnapToBottom
        zIndex={100_000}
      >
        <Sheet.Overlay
          animation="lazy"
          enterStyle={{ opacity: 0 }}
          exitStyle={{ opacity: 0 }}
        />
        <Sheet.Handle />
        <Sheet.Frame padding="$4" gap="$3">
          <Text fontSize="$6" fontWeight="700">
            Escalate alert
          </Text>
          <Paragraph color="$colorMuted">Choose how to escalate this alert.</Paragraph>
          <Sheet.ScrollView>
            <YStack gap="$1">
              {ESCALATION_TARGETS.map((target) => (
                <ListItem
                  key={target.id}
                  title={target.label}
                  backgroundColor="transparent"
                  pressTheme
                  disabled={escalateLoading}
                  onPress={() => void runEscalationTarget(target)}
                />
              ))}
            </YStack>
          </Sheet.ScrollView>
          {escalateLoading ? <Spinner size="small" color="$color" /> : null}
        </Sheet.Frame>
      </Sheet>
    </>
  )
}
