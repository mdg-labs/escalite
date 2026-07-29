'use client'

import { addDays } from 'date-fns'
import { CalendarIcon, PencilIcon, PlusIcon, TrashIcon } from 'lucide-react'
import { useEffect, useMemo, useState, type ReactElement } from 'react'
import type { DateRange } from 'react-day-picker'

import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogPopup,
  AlertDialogTitle,
} from '../../primitives/alert-dialog'
import { Alert, AlertDescription } from '../../primitives/alert'
import { Avatar, AvatarFallback } from '../../primitives/avatar'
import { Badge } from '../../primitives/badge'
import { Button } from '../../primitives/button'
import { Calendar } from '../../primitives/calendar'
import {
  Combobox,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxPopup,
} from '../../primitives/combobox'
import {
  Dialog,
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogPanel,
  DialogPopup,
  DialogTitle,
} from '../../primitives/dialog'
import {
  Drawer,
  DrawerDescription,
  DrawerHeader,
  DrawerPanel,
  DrawerPopup,
  DrawerTitle,
} from '../../primitives/drawer'
import { Frame, FrameDescription, FramePanel, FrameTitle } from '../../primitives/frame'
import { Group } from '../../primitives/group'
import { Popover, PopoverPopup, PopoverTrigger } from '../../primitives/popover'
import {
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
} from '../../primitives/select'
import { canCreateScheduleOverride, canManageRotations } from './permissions'
import {
  describeRotationRrule,
  layerLabel,
} from './rotation-display'
import {
  RotationForm,
  type RotationFormInitialValues,
} from './RotationForm'
import { formatViewerLocalDateRange, formatViewerLocalTime } from './timezone'
import type {
  CreateOverridePayload,
  CreateRotationPayload,
  ScheduleCalendarData,
  ScheduleCalendarLabels,
  ScheduleCalendarOnCallLayer,
  ScheduleCalendarOverride,
  ScheduleCalendarUser,
  UpdateRotationPayload,
  ViewerRole,
} from './types'

function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window === 'undefined') {
      return false
    }

    return window.matchMedia(query).matches
  })

  useEffect(() => {
    const media = window.matchMedia(query)
    const onChange = (): void => setMatches(media.matches)
    onChange()
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [query])

  return matches
}

function initialsFromLabel(label: string): string {
  const parts = label.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) {
    return '?'
  }

  if (parts.length === 1) {
    return parts[0]?.slice(0, 2).toUpperCase() ?? '?'
  }

  return `${parts[0]?.[0] ?? ''}${parts[1]?.[0] ?? ''}`.toUpperCase()
}

function userLabel(users: ScheduleCalendarUser[], userId: string): string {
  return users.find((user) => user.id === userId)?.label ?? userId
}

function dateKey(date: Date): string {
  return date.toISOString().slice(0, 10)
}

function toIsoRange(range: DateRange | undefined): { startsAt: string; endsAt: string } | null {
  if (!range?.from || !range.to) {
    return null
  }

  const startsAt = new Date(range.from)
  startsAt.setHours(0, 0, 0, 0)
  const endsAt = new Date(range.to)
  endsAt.setHours(23, 59, 59, 999)

  return {
    startsAt: startsAt.toISOString(),
    endsAt: endsAt.toISOString(),
  }
}

export type ScheduleCalendarProps = {
  schedule: ScheduleCalendarData
  onCallLayers: ScheduleCalendarOnCallLayer[]
  overrides: ScheduleCalendarOverride[]
  users: ScheduleCalendarUser[]
  viewerRole: ViewerRole
  hasTeamAccess: boolean
  labels: ScheduleCalendarLabels
  computedAt?: string | null
  saving?: boolean
  onCreateOverride?: (payload: CreateOverridePayload) => void | Promise<void>
  onDeleteOverride?: (overrideId: string) => void | Promise<void>
  participantOptions?: ScheduleCalendarUser[]
  onCreateRotation?: (payload: CreateRotationPayload) => void | Promise<void>
  onUpdateRotation?: (payload: UpdateRotationPayload) => void | Promise<void>
  onDeleteRotation?: (rotationId: string) => void | Promise<void>
}

type OverrideFormProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  rotations: ScheduleCalendarData['rotations']
  users: ScheduleCalendarUser[]
  labels: ScheduleCalendarLabels
  saving?: boolean
  onSubmit: (payload: CreateOverridePayload) => void | Promise<void>
}

function OverrideFormPanel({
  rotations,
  users,
  labels,
  saving,
  onSubmit,
  onCancel,
}: Omit<OverrideFormProps, 'open' | 'onOpenChange'> & {
  onCancel: () => void
}): ReactElement {
  const [rotationId, setRotationId] = useState(rotations[0]?.id ?? '')
  const [userId, setUserId] = useState<string | null>(users[0]?.id ?? null)
  const [range, setRange] = useState<DateRange | undefined>()
  const [error, setError] = useState<string | null>(null)

  const rotationItems = useMemo(
    () =>
      rotations.map((rotation) => ({
        label: `${rotation.name} (L${rotation.layer})`,
        value: rotation.id,
      })),
    [rotations],
  )

  async function handleSubmit(): Promise<void> {
    const isoRange = toIsoRange(range)
    if (!rotationId.trim()) {
      setError(labels.rotation)
      return
    }

    if (!userId) {
      setError(labels.replacementUser)
      return
    }

    if (!isoRange) {
      setError(labels.dateRange)
      return
    }

    setError(null)
    await onSubmit({
      rotationId,
      userId,
      startsAt: isoRange.startsAt,
      endsAt: isoRange.endsAt,
    })
  }

  return (
    <>
      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          <label className="text-sm font-medium text-foreground" htmlFor="override-rotation">
            {labels.rotation}
          </label>
          <Select
            items={rotationItems}
            onValueChange={(value) => setRotationId(value ?? '')}
            value={rotationId || null}
          >
            <SelectTrigger id="override-rotation">
              <SelectValue placeholder={labels.rotation} />
            </SelectTrigger>
            <SelectPopup>
              {rotationItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectPopup>
          </Select>
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium text-foreground">{labels.replacementUser}</span>
          <Combobox
            onValueChange={(value) => setUserId(value)}
            value={userId}
          >
            <ComboboxInput placeholder={labels.replacementUser} showClear />
            <ComboboxPopup>
              <ComboboxList>
                <ComboboxEmpty>No users found</ComboboxEmpty>
                {users.map((user) => (
                  <ComboboxItem key={user.id} value={user.id}>
                    {user.label}
                  </ComboboxItem>
                ))}
              </ComboboxList>
            </ComboboxPopup>
          </Combobox>
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium text-foreground">{labels.dateRange}</span>
          <Popover>
            <PopoverTrigger>
              <Button className="w-full justify-start font-normal" type="button" variant="outline">
                <CalendarIcon />
                {range?.from && range.to
                  ? `${range.from.toLocaleDateString()} – ${range.to.toLocaleDateString()}`
                  : labels.dateRange}
              </Button>
            </PopoverTrigger>
            <PopoverPopup align="start" className="w-auto p-0">
              <Calendar
                mode="range"
                numberOfMonths={2}
                onSelect={setRange}
                pagedNavigation
                selected={range}
                showOutsideDays={false}
              />
            </PopoverPopup>
          </Popover>
        </div>

        {error ? (
          <p className="text-sm text-destructive-foreground" role="alert">
            {error}
          </p>
        ) : null}
      </div>

      <div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <Button onClick={onCancel} type="button" variant="outline">
          {labels.cancel}
        </Button>
        <Button disabled={saving} onClick={() => void handleSubmit()} type="button">
          {labels.save}
        </Button>
      </div>
    </>
  )
}

function OverrideForm({
  open,
  onOpenChange,
  rotations,
  users,
  labels,
  saving,
  onSubmit,
}: OverrideFormProps): ReactElement {
  const isMobile = useMediaQuery('(max-width: 767px)')

  if (isMobile) {
    return (
      <Drawer onOpenChange={onOpenChange} open={open}>
        <DrawerPopup showBar>
          <DrawerHeader>
            <DrawerTitle>{labels.createOverride}</DrawerTitle>
            <DrawerDescription>{labels.dateRange}</DrawerDescription>
          </DrawerHeader>
          <DrawerPanel>
            <OverrideFormPanel
              labels={labels}
              onCancel={() => onOpenChange(false)}
              onSubmit={async (payload) => {
                await onSubmit(payload)
                onOpenChange(false)
              }}
              rotations={rotations}
              saving={saving}
              users={users}
            />
          </DrawerPanel>
        </DrawerPopup>
      </Drawer>
    )
  }

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogPopup>
        <DialogHeader>
          <DialogTitle>{labels.createOverride}</DialogTitle>
          <DialogDescription>{labels.dateRange}</DialogDescription>
        </DialogHeader>
        <DialogPanel>
          <OverrideFormPanel
            labels={labels}
            onCancel={() => onOpenChange(false)}
            onSubmit={async (payload) => {
              await onSubmit(payload)
              onOpenChange(false)
            }}
            rotations={rotations}
            saving={saving}
            users={users}
          />
        </DialogPanel>
        <DialogFooter className="hidden" />
        <DialogClose />
      </DialogPopup>
    </Dialog>
  )
}

type RotationDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  mode: 'create' | 'edit'
  labels: ScheduleCalendarLabels
  participantOptions: ScheduleCalendarUser[]
  usedLayers: number[]
  initialValues?: RotationFormInitialValues
  saving?: boolean
  onSubmit: (payload: CreateRotationPayload | UpdateRotationPayload) => void | Promise<void>
}

function RotationDialog({
  open,
  onOpenChange,
  mode,
  labels,
  participantOptions,
  usedLayers,
  initialValues,
  saving,
  onSubmit,
}: RotationDialogProps): ReactElement {
  const isMobile = useMediaQuery('(max-width: 767px)')
  const title = mode === 'create' ? labels.createRotation : labels.editRotation

  const form = (
    <RotationForm
      initialValues={initialValues}
      labels={labels}
      mode={mode}
      onCancel={() => onOpenChange(false)}
      onSubmit={async (payload) => {
        await onSubmit(payload)
        onOpenChange(false)
      }}
      participantOptions={participantOptions}
      saving={saving}
      usedLayers={usedLayers}
    />
  )

  if (isMobile) {
    return (
      <Drawer onOpenChange={onOpenChange} open={open}>
        <DrawerPopup showBar>
          <DrawerHeader>
            <DrawerTitle>{title}</DrawerTitle>
          </DrawerHeader>
          <DrawerPanel>{form}</DrawerPanel>
        </DrawerPopup>
      </Drawer>
    )
  }

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogPopup>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <DialogPanel>{form}</DialogPanel>
        <DialogFooter className="hidden" />
        <DialogClose />
      </DialogPopup>
    </Dialog>
  )
}

export function ScheduleCalendar({
  schedule,
  onCallLayers,
  overrides,
  users,
  viewerRole,
  hasTeamAccess,
  labels,
  computedAt,
  saving = false,
  onCreateOverride,
  onDeleteOverride,
  participantOptions = [],
  onCreateRotation,
  onUpdateRotation,
  onDeleteRotation,
}: ScheduleCalendarProps): ReactElement {
  const today = useMemo(() => new Date(), [])
  const [month, setMonth] = useState<Date>(today)
  const [selectedRange, setSelectedRange] = useState<DateRange | undefined>({
    from: today,
    to: addDays(today, 25),
  })
  const [overrideFormOpen, setOverrideFormOpen] = useState(false)
  const [rotationFormOpen, setRotationFormOpen] = useState(false)
  const [rotationFormMode, setRotationFormMode] = useState<'create' | 'edit'>('create')
  const [editingRotation, setEditingRotation] = useState<RotationFormInitialValues | undefined>()
  const [rotationToDelete, setRotationToDelete] = useState<string | null>(null)

  const viewerTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const allowOverrideCreate = canCreateScheduleOverride(viewerRole, hasTeamAccess)
  const allowRotationManage = canManageRotations(viewerRole)

  const usedLayers = useMemo(
    () =>
      schedule.rotations
        .filter((rotation) => rotation.id !== editingRotation?.id)
        .map((rotation) => rotation.layer),
    [editingRotation?.id, schedule.rotations],
  )

  function openCreateRotation(): void {
    setRotationFormMode('create')
    setEditingRotation(undefined)
    setRotationFormOpen(true)
  }

  function openEditRotation(rotationId: string): void {
    const rotation = schedule.rotations.find((item) => item.id === rotationId)
    if (!rotation) {
      return
    }

    setRotationFormMode('edit')
    setEditingRotation({
      id: rotation.id,
      name: rotation.name,
      layer: rotation.layer,
      rrule: rotation.rrule,
      participantIds: [...rotation.participantIds],
    })
    setRotationFormOpen(true)
  }

  async function handleRotationSubmit(
    payload: CreateRotationPayload | UpdateRotationPayload,
  ): Promise<void> {
    if ('id' in payload && onUpdateRotation) {
      await onUpdateRotation(payload)
      return
    }

    if (onCreateRotation) {
      await onCreateRotation(payload)
    }
  }

  async function handleConfirmDeleteRotation(): Promise<void> {
    if (!rotationToDelete || !onDeleteRotation) {
      return
    }

    await onDeleteRotation(rotationToDelete)
    setRotationToDelete(null)
  }

  const overrideDays = useMemo(() => {
    const days = new Set<string>()
    for (const override of overrides) {
      const start = new Date(override.startsAt)
      const end = new Date(override.endsAt)
      const cursor = new Date(start)
      cursor.setHours(0, 0, 0, 0)
      const endDay = new Date(end)
      endDay.setHours(0, 0, 0, 0)

      while (cursor <= endDay) {
        days.add(dateKey(cursor))
        cursor.setDate(cursor.getDate() + 1)
      }
    }

    return days
  }, [overrides])

  const visibleOverrides = useMemo(() => {
    if (!selectedRange?.from) {
      return overrides
    }

    const from = selectedRange.from
    const to = selectedRange.to ?? selectedRange.from
    return overrides.filter((override) => {
      const cursor = new Date(from)
      cursor.setHours(0, 0, 0, 0)
      const endDay = new Date(to)
      endDay.setHours(23, 59, 59, 999)
      const overrideStart = new Date(override.startsAt)
      const overrideEnd = new Date(override.endsAt)
      return overrideStart <= endDay && overrideEnd >= cursor
    })
  }, [overrides, selectedRange])

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold text-foreground">{schedule.name}</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {labels.timezoneLabel}: {viewerTimezone} ({schedule.timezone})
          </p>
          {computedAt ? (
            <p className="mt-1 text-xs text-muted-foreground">
              {labels.computedAt}: {formatViewerLocalTime(computedAt).formatted}
            </p>
          ) : null}
        </div>

        {allowOverrideCreate ? (
          <Button onClick={() => setOverrideFormOpen(true)} type="button">
            <PlusIcon />
            {labels.createOverride}
          </Button>
        ) : (
          <Alert className="max-w-md" variant="warning">
            <AlertDescription>{labels.overrideForbidden}</AlertDescription>
          </Alert>
        )}
      </div>

      <Frame>
        <FrameTitle>{labels.onCallNow}</FrameTitle>
        <FrameDescription>{labels.onCallNow}</FrameDescription>
        <FramePanel>
          {onCallLayers.length === 0 ? (
            <p className="text-sm text-muted-foreground">{labels.loading}</p>
          ) : (
            <div className="flex flex-col gap-4">
              {onCallLayers.map((layer) => (
                <div
                  className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border p-3"
                  key={`${layer.layer}-${layer.rotationId}`}
                >
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {labels.layerLabel(layer.layer)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {schedule.rotations.find((rotation) => rotation.id === layer.rotationId)
                        ?.name ?? layer.rotationId}
                    </p>
                  </div>
                  <Group className="-space-x-2">
                    <Avatar className="ring-2 ring-background">
                      <AvatarFallback>
                        {initialsFromLabel(userLabel(users, layer.userId))}
                      </AvatarFallback>
                    </Avatar>
                    <span className="ps-3 text-sm text-foreground">
                      {userLabel(users, layer.userId)}
                    </span>
                  </Group>
                </div>
              ))}
            </div>
          )}
        </FramePanel>
      </Frame>

      <Frame>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <FrameTitle>{labels.rotationsTitle}</FrameTitle>
            <FrameDescription>{labels.rotationsTitle}</FrameDescription>
          </div>
          {allowRotationManage && onCreateRotation ? (
            <Button onClick={openCreateRotation} type="button" variant="outline">
              <PlusIcon />
              {labels.createRotation}
            </Button>
          ) : null}
        </div>
        <FramePanel>
          {!allowRotationManage ? (
            <Alert className="mb-4" variant="warning">
              <AlertDescription>{labels.rotationForbidden}</AlertDescription>
            </Alert>
          ) : null}

          {schedule.rotations.length === 0 ? (
            <p className="text-sm text-muted-foreground">{labels.rotationsEmpty}</p>
          ) : (
            <ul className="flex flex-col gap-3">
              {schedule.rotations.map((rotation) => (
                <li
                  className="flex flex-wrap items-start justify-between gap-3 rounded-lg border border-border p-3"
                  key={rotation.id}
                >
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium text-foreground">{rotation.name}</span>
                      <Badge variant="outline">{layerLabel(labels, rotation.layer)}</Badge>
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {describeRotationRrule(labels, rotation.rrule)}
                    </p>
                    <p className="mt-2 text-xs text-muted-foreground">
                      {rotation.participantIds
                        .map((participantId) => userLabel(users, participantId))
                        .join(', ')}
                    </p>
                  </div>
                  {allowRotationManage && (onUpdateRotation || onDeleteRotation) ? (
                    <div className="flex items-center gap-2">
                      {onUpdateRotation ? (
                        <Button
                          aria-label={labels.editRotation}
                          onClick={() => openEditRotation(rotation.id)}
                          size="icon-sm"
                          type="button"
                          variant="outline"
                        >
                          <PencilIcon />
                        </Button>
                      ) : null}
                      {onDeleteRotation ? (
                        <Button
                          aria-label={labels.deleteRotation}
                          onClick={() => setRotationToDelete(rotation.id)}
                          size="icon-sm"
                          type="button"
                          variant="destructive-outline"
                        >
                          <TrashIcon />
                        </Button>
                      ) : null}
                    </div>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </FramePanel>
      </Frame>

      <Frame>
        <FrameTitle>{labels.title}</FrameTitle>
        <FrameDescription>{labels.timezoneLabel}</FrameDescription>
        <FramePanel>
          <Calendar
            classNames={{
              month:
                'relative first-of-type:before:hidden before:absolute max-sm:before:inset-x-2 max-sm:before:h-px max-sm:before:-top-2 sm:before:inset-y-2 sm:before:w-px before:bg-border sm:before:-left-4',
              months: 'gap-8',
            }}
            modifiers={{
              override: (day) => overrideDays.has(dateKey(day)),
            }}
            modifiersClassNames={{
              override: 'bg-warning/20 text-warning-foreground font-medium',
            }}
            mode="range"
            month={month}
            numberOfMonths={2}
            onMonthChange={setMonth}
            onSelect={setSelectedRange}
            pagedNavigation
            selected={selectedRange}
            showOutsideDays={false}
          />
        </FramePanel>
      </Frame>

      <Frame>
        <FrameTitle>{labels.overrides}</FrameTitle>
        <FramePanel>
          {visibleOverrides.length === 0 ? (
            <p className="text-sm text-muted-foreground">{labels.noOverrides}</p>
          ) : (
            <ul className="flex flex-col gap-3">
              {visibleOverrides.map((override) => {
                const rangeLabel = formatViewerLocalDateRange(
                  override.startsAt,
                  override.endsAt,
                )
                const rotationName =
                  schedule.rotations.find((rotation) => rotation.id === override.rotationId)
                    ?.name ?? override.rotationId

                return (
                  <li
                    className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border p-3"
                    key={override.id}
                  >
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant="outline">{rotationName}</Badge>
                        <span className="text-sm font-medium text-foreground">
                          {userLabel(users, override.userId)}
                        </span>
                      </div>
                      <p className="mt-1 text-sm text-muted-foreground">{rangeLabel}</p>
                      <p className="text-xs text-muted-foreground">
                        {labels.timezoneLabel}: {viewerTimezone}
                      </p>
                    </div>
                    {allowOverrideCreate && onDeleteOverride ? (
                      <Button
                        aria-label={labels.delete}
                        onClick={() => void onDeleteOverride(override.id)}
                        size="icon-sm"
                        type="button"
                        variant="destructive-outline"
                      >
                        <TrashIcon />
                      </Button>
                    ) : null}
                  </li>
                )
              })}
            </ul>
          )}
        </FramePanel>
      </Frame>

      {allowOverrideCreate && onCreateOverride ? (
        <OverrideForm
          labels={labels}
          onOpenChange={setOverrideFormOpen}
          onSubmit={onCreateOverride}
          open={overrideFormOpen}
          rotations={schedule.rotations}
          saving={saving}
          users={users}
        />
      ) : null}

      {allowRotationManage && (onCreateRotation || onUpdateRotation) ? (
        <RotationDialog
          initialValues={editingRotation}
          labels={labels}
          mode={rotationFormMode}
          onOpenChange={setRotationFormOpen}
          onSubmit={handleRotationSubmit}
          open={rotationFormOpen}
          participantOptions={participantOptions}
          saving={saving}
          usedLayers={usedLayers}
        />
      ) : null}

      {allowRotationManage && onDeleteRotation ? (
        <AlertDialog onOpenChange={(open) => !open && setRotationToDelete(null)} open={Boolean(rotationToDelete)}>
          <AlertDialogPopup>
            <AlertDialogHeader>
              <AlertDialogTitle>{labels.deleteRotation}</AlertDialogTitle>
              <AlertDialogDescription>{labels.deleteRotationConfirm}</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogClose render={<Button type="button" variant="outline" />}>
                {labels.cancel}
              </AlertDialogClose>
              <Button
                disabled={saving}
                onClick={() => void handleConfirmDeleteRotation()}
                type="button"
                variant="destructive"
              >
                {labels.deleteRotation}
              </Button>
            </AlertDialogFooter>
          </AlertDialogPopup>
        </AlertDialog>
      ) : null}
    </div>
  )
}
