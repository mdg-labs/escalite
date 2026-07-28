'use client'

import { useEffect, useMemo, useState, type ReactElement } from 'react'

import { Button } from '../../primitives/button'
import { Input } from '../../primitives/input'
import {
  NumberField,
  NumberFieldDecrement,
  NumberFieldGroup,
  NumberFieldIncrement,
  NumberFieldInput,
} from '../../primitives/number-field'
import { ScrollArea } from '../../primitives/scroll-area'
import {
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
} from '../../primitives/select'
import { buildRRule, parseRRule, type RotationFrequency } from './rrule'
import type {
  CreateRotationPayload,
  ScheduleCalendarLabels,
  ScheduleCalendarRotation,
  ScheduleCalendarUser,
  UpdateRotationPayload,
} from './types'

export type RotationFormMode = 'create' | 'edit'

export type RotationFormInitialValues = {
  id?: string
  name: string
  layer: number
  rrule: string
  participantIds: string[]
}

export type RotationFormProps = {
  mode: RotationFormMode
  labels: ScheduleCalendarLabels
  participantOptions: ScheduleCalendarUser[]
  usedLayers: number[]
  saving?: boolean
  initialValues?: RotationFormInitialValues
  onSubmit: (payload: CreateRotationPayload | UpdateRotationPayload) => void | Promise<void>
  onCancel: () => void
}

const LAYER_OPTIONS = [1, 2] as const

function rotationToInitialValues(rotation?: ScheduleCalendarRotation): RotationFormInitialValues {
  if (!rotation) {
    return {
      name: '',
      layer: 1,
      rrule: buildRRule('WEEKLY', 1),
      participantIds: [],
    }
  }

  return {
    id: rotation.id,
    name: rotation.name,
    layer: rotation.layer,
    rrule: rotation.rrule,
    participantIds: [...rotation.participantIds],
  }
}

export function RotationForm({
  mode,
  labels,
  participantOptions,
  usedLayers,
  saving = false,
  initialValues,
  onSubmit,
  onCancel,
}: RotationFormProps): ReactElement {
  const [name, setName] = useState('')
  const [layer, setLayer] = useState<number>(1)
  const [frequency, setFrequency] = useState<RotationFrequency>('WEEKLY')
  const [interval, setInterval] = useState(1)
  const [useCustomRrule, setUseCustomRrule] = useState(false)
  const [customRrule, setCustomRrule] = useState('')
  const [participantIds, setParticipantIds] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const values = initialValues ?? rotationToInitialValues()
    const parsed = parseRRule(values.rrule)

    setName(values.name)
    setLayer(values.layer)
    setParticipantIds(values.participantIds)

    if (parsed) {
      setUseCustomRrule(false)
      setFrequency(parsed.frequency)
      setInterval(parsed.interval)
      setCustomRrule(values.rrule)
    } else {
      setUseCustomRrule(true)
      setCustomRrule(values.rrule)
      setFrequency('WEEKLY')
      setInterval(1)
    }

    setError(null)
  }, [initialValues])

  const availableLayers = useMemo(() => {
    const currentLayer = initialValues?.layer
    return LAYER_OPTIONS.filter(
      (option) => option === currentLayer || !usedLayers.includes(option),
    )
  }, [initialValues?.layer, usedLayers])

  const rrulePreview = useMemo(() => {
    if (useCustomRrule) {
      return customRrule.trim()
    }

    return buildRRule(frequency, interval)
  }, [customRrule, frequency, interval, useCustomRrule])

  function toggleParticipant(userId: string): void {
    setParticipantIds((current) =>
      current.includes(userId)
        ? current.filter((id) => id !== userId)
        : [...current, userId],
    )
  }

  async function handleSubmit(): Promise<void> {
    const trimmedName = name.trim()
    if (!trimmedName) {
      setError(labels.rotationErrorName)
      return
    }

    if (!layer || layer < 1) {
      setError(labels.rotationErrorLayer)
      return
    }

    if (participantIds.length === 0) {
      setError(labels.rotationErrorParticipants)
      return
    }

    const rrule = rrulePreview.trim()
    if (!rrule) {
      setError(labels.rotationErrorRrule)
      return
    }

    setError(null)

    const payload: CreateRotationPayload = {
      name: trimmedName,
      layer,
      rrule,
      participantIds,
    }

    if (mode === 'edit' && initialValues?.id) {
      await onSubmit({ ...payload, id: initialValues.id })
      return
    }

    await onSubmit(payload)
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium text-foreground" htmlFor="rotation-name">
          {labels.rotationName}
        </label>
        <Input
          id="rotation-name"
          onChange={(event) => setName(event.target.value)}
          value={name}
        />
      </div>

      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium text-foreground" htmlFor="rotation-layer">
          {labels.rotationLayer}
        </label>
        <Select
          onValueChange={(value) => setLayer(Number(value ?? 1))}
          value={String(layer)}
        >
          <SelectTrigger id="rotation-layer">
            <SelectValue placeholder={labels.rotationLayer} />
          </SelectTrigger>
          <SelectPopup>
            {availableLayers.map((option) => (
              <SelectItem key={option} value={String(option)}>
                {option === 1 ? labels.rotationLayerPrimary : labels.rotationLayerSecondary}
              </SelectItem>
            ))}
          </SelectPopup>
        </Select>
      </div>

      <div className="flex flex-col gap-2">
        <span className="text-sm font-medium text-foreground">{labels.rotationFrequency}</span>
        <div className="flex flex-col gap-3 sm:flex-row">
          <Select
            disabled={useCustomRrule}
            onValueChange={(value) => setFrequency((value as RotationFrequency) ?? 'WEEKLY')}
            value={frequency}
          >
            <SelectTrigger>
              <SelectValue placeholder={labels.rotationFrequency} />
            </SelectTrigger>
            <SelectPopup>
              <SelectItem value="HOURLY">{labels.rotationFrequencyHourly}</SelectItem>
              <SelectItem value="DAILY">{labels.rotationFrequencyDaily}</SelectItem>
              <SelectItem value="WEEKLY">{labels.rotationFrequencyWeekly}</SelectItem>
            </SelectPopup>
          </Select>

          <div className="flex min-w-40 flex-col gap-1">
            <span className="text-xs text-muted-foreground">{labels.rotationInterval}</span>
            <NumberField
              disabled={useCustomRrule}
              min={1}
              onValueChange={(value) => setInterval(value ?? 1)}
              value={interval}
            >
              <NumberFieldGroup>
                <NumberFieldDecrement />
                <NumberFieldInput />
                <NumberFieldIncrement />
              </NumberFieldGroup>
            </NumberField>
          </div>
        </div>

        <label className="flex items-center gap-2 text-sm text-muted-foreground">
          <input
            checked={useCustomRrule}
            className="size-4 rounded border border-input"
            onChange={(event) => setUseCustomRrule(event.target.checked)}
            type="checkbox"
          />
          {labels.rotationCustomRrule}
        </label>

        {useCustomRrule ? (
          <Input
            onChange={(event) => setCustomRrule(event.target.value)}
            placeholder="FREQ=WEEKLY;INTERVAL=1"
            value={customRrule}
          />
        ) : null}

        <p className="text-xs text-muted-foreground">
          {labels.rotationRrulePreview}: {rrulePreview}
        </p>
      </div>

      <div className="flex flex-col gap-2">
        <span className="text-sm font-medium text-foreground">{labels.rotationParticipants}</span>
        {participantOptions.length === 0 ? (
          <p className="text-sm text-muted-foreground">{labels.rotationParticipantsEmpty}</p>
        ) : (
          <ScrollArea className="max-h-48 rounded-lg border border-border">
            <ul className="flex flex-col gap-1 p-2">
              {participantOptions.map((user) => {
                const checked = participantIds.includes(user.id)
                return (
                  <li key={user.id}>
                    <label className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 hover:bg-muted/50">
                      <input
                        checked={checked}
                        className="size-4 rounded border border-input"
                        onChange={() => toggleParticipant(user.id)}
                        type="checkbox"
                      />
                      <span className="text-sm text-foreground">{user.label}</span>
                    </label>
                  </li>
                )
              })}
            </ul>
          </ScrollArea>
        )}
      </div>

      {error ? (
        <p className="text-sm text-destructive-foreground" role="alert">
          {error}
        </p>
      ) : null}

      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <Button onClick={onCancel} type="button" variant="outline">
          {labels.cancel}
        </Button>
        <Button disabled={saving} onClick={() => void handleSubmit()} type="button">
          {labels.save}
        </Button>
      </div>
    </div>
  )
}
