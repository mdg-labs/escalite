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
  useTeamsQuery,
} from '@escalite/ts-types'
import { Button } from '@escalite/ui'
import { PencilIcon } from 'lucide-react'
import { ScheduleCalendar } from '@escalite/ui/domain/ScheduleCalendar'

import { AppShell } from '../components/app-shell'
import { PageBreadcrumbs } from '../components/page-breadcrumbs'
import { ScheduleFormDialog } from '../components/schedule-form-dialog'
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

  const [{ data, fetching, error }, reexecuteSchedule] = useScheduleQuery({
    pause: !scheduleId,
    variables: { id: scheduleId ?? '' },
  })
  const [{ data: teamsData }] = useTeamsQuery({ requestPolicy: 'cache-first' })

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
  const [editOpen, setEditOpen] = useState(false)

  const schedule = data?.schedule
  const scheduleTitle = schedule?.name ?? t('schedule.pageTitle')
  const teamName = useMemo(() => {
    if (!schedule?.teamId) {
      return ''
    }

    return teamsData?.teams.find((team) => team.id === schedule.teamId)?.name ?? schedule.teamId
  }, [schedule?.teamId, teamsData?.teams])
  const breadcrumbItems = useMemo(
    () => [
      { label: t('nav.schedules'), href: '/schedules' },
      { label: scheduleTitle },
    ],
    [scheduleTitle],
  )
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
    <AppShell title={scheduleTitle}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
        <PageBreadcrumbs items={breadcrumbItems} />
        <div className="mt-2 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold text-foreground">{scheduleTitle}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{t('schedule.pageDescription')}</p>
            {schedule ? (
              <p className="mt-2 text-sm text-muted-foreground">
                {t('schedule.meta', { team: teamName, timezone: schedule.timezone })}
              </p>
            ) : null}
          </div>
          {schedule ? (
            <ScheduleFormDialog
              initialValues={{
                id: schedule.id,
                name: schedule.name,
                timezone: schedule.timezone,
                teamId: schedule.teamId,
                teamName,
              }}
              mode="edit"
              onOpenChange={setEditOpen}
              onSuccess={() => {
                reexecuteSchedule({ requestPolicy: 'network-only' })
              }}
              open={editOpen}
              teams={[]}
              trigger={
                <Button type="button" variant="outline">
                  <PencilIcon />
                  {t('schedule.action.edit')}
                </Button>
              }
            />
          ) : null}
        </div>

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
