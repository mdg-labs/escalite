import { useLocalSearchParams } from 'expo-router'
import { useEffect, useState } from 'react'
import { Card, H3, ListItem, Paragraph, ScrollView, Separator, Spinner, Text, YStack } from 'tamagui'

import { fetchAlert, type MobileAlert } from '@/api/alerts'
import { useAuth } from '@/auth/context'
import { severityColors } from '@escalite/tokens'

function priorityColor(priority: string): string {
  switch (priority.toLowerCase()) {
    case 'critical':
      return severityColors.severityCritical
    case 'high':
      return severityColors.severityHigh
    case 'medium':
      return severityColors.severityMedium
    case 'low':
      return severityColors.severityLow
    default:
      return severityColors.severityInfo
  }
}

function formatTimestamp(value: string | null): string {
  if (!value) {
    return '—'
  }

  return new Date(value).toLocaleString()
}

function AlertMetadata({ alert }: { alert: MobileAlert }) {
  return (
    <YStack gap="$1">
      <ListItem title="Service" subTitle={alert.serviceId} backgroundColor="transparent" />
      <Separator />
      <ListItem title="Fired at" subTitle={formatTimestamp(alert.createdAt)} backgroundColor="transparent" />
      <Separator />
      <ListItem
        title="Severity"
        subTitle={alert.priority}
        backgroundColor="transparent"
        iconAfter={
          <Text color={priorityColor(alert.priority)} fontWeight="600">
            ●
          </Text>
        }
      />
      <Separator />
      <ListItem title="Status" subTitle={alert.status} backgroundColor="transparent" />
    </YStack>
  )
}

export default function AlertDetailScreen() {
  const { alertId } = useLocalSearchParams<{ alertId: string }>()
  const { status } = useAuth()
  const [alert, setAlert] = useState<MobileAlert | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!alertId || status !== 'authenticated') {
      setLoading(status === 'loading')
      return
    }

    let cancelled = false
    setLoading(true)
    setError(null)

    void fetchAlert(alertId)
      .then((nextAlert) => {
        if (cancelled) {
          return
        }
        setAlert(nextAlert)
        if (!nextAlert) {
          setError('Alert not found.')
        }
      })
      .catch((err: unknown) => {
        if (cancelled) {
          return
        }
        setError(err instanceof Error ? err.message : 'Unable to load alert')
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [alertId, status])

  if (status === 'loading' || loading) {
    return (
      <YStack flex={1} alignItems="center" justifyContent="center" backgroundColor="$background" padding="$4" gap="$3">
        <Spinner size="large" color="$color" />
        <Paragraph color="$color">Loading alert…</Paragraph>
      </YStack>
    )
  }

  if (status !== 'authenticated') {
    return (
      <YStack flex={1} alignItems="center" justifyContent="center" backgroundColor="$background" padding="$4" gap="$3">
        <Paragraph color="$color" textAlign="center">
          Sign in to view alert details.
        </Paragraph>
      </YStack>
    )
  }

  if (error || !alert) {
    return (
      <YStack flex={1} alignItems="center" justifyContent="center" backgroundColor="$background" padding="$4" gap="$3">
        <Paragraph color="$red10" textAlign="center" role="alert">
          {error ?? 'Alert not found.'}
        </Paragraph>
      </YStack>
    )
  }

  return (
    <ScrollView flex={1} backgroundColor="$background" contentContainerStyle={{ padding: 16 }}>
      <YStack gap="$4">
        <Card backgroundColor="$backgroundHover" borderColor="$borderColor" borderWidth={1} padding="$4" gap="$3">
          <H3 color="$color">{alert.summary}</H3>
          <Paragraph color="$colorMuted">{alert.description}</Paragraph>
        </Card>

        <Card backgroundColor="$backgroundHover" borderColor="$borderColor" borderWidth={1} padding="$4">
          <AlertMetadata alert={alert} />
        </Card>
      </YStack>
    </ScrollView>
  )
}
