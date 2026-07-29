import { Card, H2, ListItem, Paragraph, Separator, Spinner, Text, YStack } from 'tamagui'

import { isOnCall, type MyOnCallAssignment } from '@/api/on-call'

type OnCallStatusProps = {
  assignments: MyOnCallAssignment[]
  loading: boolean
  error: string | null
}

function formatUntil(value: string): string {
  return new Date(value).toLocaleString()
}

function OnCallAssignmentList({ assignments }: { assignments: MyOnCallAssignment[] }) {
  return (
    <YStack gap="$1">
      {assignments.map((assignment, index) => (
        <YStack key={assignment.scheduleId} gap="$1">
          {index > 0 ? <Separator /> : null}
          <ListItem
            title={assignment.scheduleName}
            subTitle={`${assignment.teamName} · Layer ${assignment.layer}`}
            backgroundColor="transparent"
          />
          <Paragraph color="$colorMuted" paddingHorizontal="$4">
            On call until {formatUntil(assignment.until)}
          </Paragraph>
        </YStack>
      ))}
    </YStack>
  )
}

export function OnCallStatus({ assignments, loading, error }: OnCallStatusProps) {
  if (loading) {
    return (
      <YStack alignItems="center" gap="$3" paddingVertical="$6">
        <Spinner size="large" color="$color" />
        <Paragraph color="$color">Loading on-call status…</Paragraph>
      </YStack>
    )
  }

  if (error) {
    return (
      <YStack alignItems="center" gap="$3" paddingVertical="$6">
        <Paragraph color="$red10" textAlign="center" role="alert">
          {error}
        </Paragraph>
      </YStack>
    )
  }

  const onCall = isOnCall(assignments)

  return (
    <YStack gap="$4">
      <YStack alignItems="center" gap="$2">
        <H2
          color={onCall ? '$green10' : '$colorMuted'}
          fontSize="$9"
          fontWeight="700"
          textAlign="center"
        >
          {onCall ? 'On call' : 'Not on call'}
        </H2>
        {onCall ? (
          <Text color="$colorMuted" textAlign="center">
            You are currently on call for {assignments.length}{' '}
            {assignments.length === 1 ? 'schedule' : 'schedules'}.
          </Text>
        ) : (
          <Text color="$colorMuted" textAlign="center">
            You are not on call right now.
          </Text>
        )}
      </YStack>

      {onCall ? (
        <Card backgroundColor="$backgroundHover" borderColor="$borderColor" borderWidth={1} padding="$4">
          <OnCallAssignmentList assignments={assignments} />
        </Card>
      ) : null}
    </YStack>
  )
}
