import type { ReactElement } from 'react'
import { useEffect, useState } from 'react'
import { useCreateScheduleMutation, useUpdateScheduleMutation } from '@escalite/ts-types'
import {
  Button,
  Dialog,
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogPanel,
  DialogPopup,
  DialogTitle,
  DialogTrigger,
  Input,
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
} from '@escalite/ui'

import { t } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'
import { defaultIanaTimezone, isValidIanaTimezone } from '../lib/timezones'
import { TimezoneCombobox } from './timezone-combobox'

type TeamOption = {
  id: string
  name: string
}

type ScheduleFormValues = {
  id: string
  name: string
  timezone: string
  teamId: string
  teamName: string
}

type ScheduleFormDialogProps = {
  mode: 'create' | 'edit'
  open: boolean
  onOpenChange: (open: boolean) => void
  teams: TeamOption[]
  defaultTeamId?: string
  initialValues?: ScheduleFormValues
  onSuccess?: (schedule: { id: string }) => void
  trigger?: ReactElement
}

export function ScheduleFormDialog({
  mode,
  open,
  onOpenChange,
  teams,
  defaultTeamId,
  initialValues,
  onSuccess,
  trigger,
}: ScheduleFormDialogProps): ReactElement {
  const [, createSchedule] = useCreateScheduleMutation()
  const [, updateSchedule] = useUpdateScheduleMutation()

  const [name, setName] = useState('')
  const [teamId, setTeamId] = useState('')
  const [timezone, setTimezone] = useState(defaultIanaTimezone())
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [showTimezoneError, setShowTimezoneError] = useState(false)

  useEffect(() => {
    if (!open) {
      return
    }

    if (mode === 'edit' && initialValues) {
      setName(initialValues.name)
      setTeamId(initialValues.teamId)
      setTimezone(initialValues.timezone)
    } else {
      setName('')
      setTeamId(defaultTeamId ?? teams[0]?.id ?? '')
      setTimezone(defaultIanaTimezone())
    }

    setActionError(null)
    setShowTimezoneError(false)
  }, [defaultTeamId, initialValues, mode, open, teams])

  async function handleSubmit(): Promise<void> {
    const trimmedName = name.trim()
    if (!trimmedName) {
      setActionError(t('schedule.form.error.requiredName'))
      return
    }

    if (!isValidIanaTimezone(timezone)) {
      setShowTimezoneError(true)
      setActionError(t('schedule.form.error.invalidTimezone'))
      return
    }

    if (mode === 'create' && !teamId) {
      setActionError(t('schedule.form.error.requiredTeam'))
      return
    }

    setActionError(null)
    setShowTimezoneError(false)
    setSaving(true)

    let scheduleId: string | undefined

    if (mode === 'create') {
      const result = await createSchedule({
        input: {
          teamId,
          name: trimmedName,
          timezone: timezone.trim(),
        },
      })

      if (result.error) {
        setSaving(false)
        showMutationError(result.error, 'schedule.form.error.save')
        return
      }

      scheduleId = result.data?.createSchedule.id
    } else {
      const result = await updateSchedule({
        input: {
          id: initialValues?.id ?? '',
          name: trimmedName,
          timezone: timezone.trim(),
        },
      })

      if (result.error) {
        setSaving(false)
        showMutationError(result.error, 'schedule.form.error.save')
        return
      }

      scheduleId = result.data?.updateSchedule.id
    }

    setSaving(false)

    if (!scheduleId) {
      showMutationError(t('schedule.form.error.save'), 'schedule.form.error.save')
      return
    }

    notifyMutationSuccess(
      mode === 'create' ? 'schedules.toast.created' : 'schedules.toast.updated',
    )
    onOpenChange(false)
    onSuccess?.({ id: scheduleId })
  }

  const title =
    mode === 'create' ? t('schedule.form.create.title') : t('schedule.form.edit.title')
  const description =
    mode === 'create'
      ? t('schedule.form.create.description')
      : t('schedule.form.edit.description')

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      {trigger ? <DialogTrigger render={trigger} /> : null}
      <DialogPopup>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogPanel className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="schedule-name">
              {t('schedule.form.field.name')}
            </label>
            <Input
              id="schedule-name"
              onChange={(event) => setName(event.target.value)}
              placeholder={t('schedule.form.field.namePlaceholder')}
              value={name}
            />
          </div>

          {mode === 'create' ? (
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground" htmlFor="schedule-team">
                {t('schedule.form.field.team')}
              </label>
              <Select
                disabled={teams.length === 0}
                onValueChange={(value) => setTeamId(value ?? '')}
                value={teamId}
              >
                <SelectTrigger id="schedule-team">
                  <SelectValue placeholder={t('schedule.form.field.teamPlaceholder')} />
                </SelectTrigger>
                <SelectPopup>
                  {teams.map((team) => (
                    <SelectItem key={team.id} value={team.id}>
                      {team.name}
                    </SelectItem>
                  ))}
                </SelectPopup>
              </Select>
            </div>
          ) : (
            <div className="space-y-2">
              <span className="text-sm font-medium text-foreground">{t('schedule.form.field.team')}</span>
              <p className="text-sm text-muted-foreground">{initialValues?.teamName}</p>
            </div>
          )}

          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="schedule-timezone">
              {t('schedule.form.field.timezone')}
            </label>
            <TimezoneCombobox
              id="schedule-timezone"
              invalid={showTimezoneError}
              onValueChange={setTimezone}
              value={timezone}
            />
          </div>

          {actionError ? (
            <p className="text-sm text-destructive-foreground" role="alert">
              {actionError}
            </p>
          ) : null}
        </DialogPanel>
        <DialogFooter>
          <DialogClose render={<Button type="button" variant="ghost" />}>
            {t('schedule.action.cancel')}
          </DialogClose>
          <Button disabled={saving} onClick={() => void handleSubmit()} type="button">
            {saving
              ? t('schedule.form.action.saving')
              : mode === 'create'
                ? t('schedule.form.action.create')
                : t('schedule.form.action.save')}
          </Button>
        </DialogFooter>
      </DialogPopup>
    </Dialog>
  )
}
