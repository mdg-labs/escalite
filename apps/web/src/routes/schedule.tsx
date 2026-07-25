import type { ReactElement } from 'react'
import { useMemo, useState } from 'react'
import { useParams } from 'react-router'
import {
  useCreateOverrideMutation,
  useDeleteOverrideMutation,
  useMeQuery,
  useOnCallNowQuery,
  useOverridesQuery,
  useScheduleQuery,
} from '@escalite/ts-types'
import { ScheduleCalendar } from '@escalite/ui/domain/ScheduleCalendar'

import { AppShell } from '../components/app-shell'
import {
  collectScheduleUsers,
  formatGraphQLError,
  scheduleCalendarLabels,
} from '../lib/schedule'
import { t } from '../lib/i18n'

export function SchedulePage(): ReactElement {
  const { scheduleId } = useParams()
  const labels = useMemo(() => scheduleCalendarLabels(), [])

  const [{ data: meData }] = useMeQuery()
  const viewerRole = meData?.me?.role ?? 'MEMBER'

  const [{ data, fetching, error }] = useScheduleQuery({
    pause: !scheduleId,
    variables: { id: scheduleId ?? '' },
  })

  const [{ data: onCallData }] = useOnCallNowQuery({
    pause: !scheduleId,
    variables: { scheduleId: scheduleId ?? '' },
  })

  const [{ data: overridesData }, reexecuteOverrides] = useOverridesQuery({
    pause: !scheduleId,
    variables: { scheduleId: scheduleId ?? '' },
  })

  const [, createOverride] = useCreateOverrideMutation()
  const [, deleteOverride] = useDeleteOverrideMutation()

  const [saveError, setSaveError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const schedule = data?.schedule
  const onCallLayers = useMemo(
    () => onCallData?.onCallNow?.layers ?? [],
    [onCallData?.onCallNow?.layers],
  )
  const overrides = useMemo(
    () => overridesData?.overrides ?? [],
    [overridesData?.overrides],
  )

  const users = useMemo(() => {
    if (!schedule) {
      return []
    }

    const participantIds = schedule.rotations.flatMap((rotation) => rotation.participantIds)
    const onCallUserIds = onCallLayers.map((layer) => layer.userId)
    const overrideUserIds = overrides.map((override) => override.userId)

    return collectScheduleUsers(participantIds, onCallUserIds, overrideUserIds)
  }, [onCallLayers, overrides, schedule])

  const hasTeamAccess =
    viewerRole === 'ADMIN' || (viewerRole === 'MEMBER' && Boolean(schedule) && !error)

  async function handleCreateOverride(payload: {
    rotationId: string
    userId: string
    startsAt: string
    endsAt: string
  }): Promise<void> {
    if (!scheduleId) {
      return
    }

    setSaveError(null)
    setSaving(true)

    const result = await createOverride({
      input: {
        scheduleId,
        rotationId: payload.rotationId,
        userId: payload.userId,
        startsAt: payload.startsAt,
        endsAt: payload.endsAt,
      },
    })

    setSaving(false)

    if (result.error) {
      setSaveError(formatGraphQLError(result.error.message))
      return
    }

    reexecuteOverrides({ requestPolicy: 'network-only' })
  }

  async function handleDeleteOverride(id: string): Promise<void> {
    setSaveError(null)
    setSaving(true)

    const result = await deleteOverride({ id })
    setSaving(false)

    if (result.error) {
      setSaveError(formatGraphQLError(result.error.message))
      return
    }

    reexecuteOverrides({ requestPolicy: 'network-only' })
  }

  return (
    <AppShell title={t('schedule.pageTitle')}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <h1 className="text-xl font-semibold text-foreground">{t('schedule.pageTitle')}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t('schedule.pageDescription')}</p>

        {fetching ? <p className="mt-6 text-sm text-muted-foreground">{labels.loading}</p> : null}

        {error ? (
          <p className="mt-6 text-sm text-destructive-foreground" role="alert">
            {formatGraphQLError(error.message)}
          </p>
        ) : null}

        {!fetching && !schedule ? (
          <p className="mt-6 text-sm text-destructive-foreground" role="alert">
            {t('schedule.notFound')}
          </p>
        ) : null}

        {schedule ? (
          <div className="mt-6">
            <ScheduleCalendar
              computedAt={onCallData?.onCallNow?.computedAt}
              hasTeamAccess={hasTeamAccess}
              labels={labels}
              onCallLayers={onCallLayers}
              onCreateOverride={handleCreateOverride}
              onDeleteOverride={handleDeleteOverride}
              overrides={overrides}
              saving={saving}
              schedule={schedule}
              users={users}
              viewerRole={viewerRole}
            />

            {saveError ? (
              <p className="mt-4 text-sm text-destructive-foreground" role="alert">
                {saveError}
              </p>
            ) : null}
          </div>
        ) : null}
      </section>
    </AppShell>
  )
}
