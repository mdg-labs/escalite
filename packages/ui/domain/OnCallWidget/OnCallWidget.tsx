'use client'

import type { ReactElement } from 'react'

import { Avatar, AvatarFallback, AvatarImage } from '../../primitives/avatar'
import { Badge } from '../../primitives/badge'
import { Frame, FrameDescription, FramePanel, FrameTitle } from '../../primitives/frame'
import { Group } from '../../primitives/group'
import { cn } from '../../lib/utils'
import type {
  OnCallWidgetLabels,
  OnCallWidgetSchedule,
  OnCallWidgetUntilByScheduleLayer,
  OnCallWidgetUser,
} from './types'
import {
  initialsFromLabel,
  layerRoleLabel,
  scheduleLayerKey,
  sortOnCallLayers,
  userAvatarUrl,
  userLabel,
} from './utils'

export type OnCallWidgetProps = {
  schedules: OnCallWidgetSchedule[]
  users: OnCallWidgetUser[]
  labels: OnCallWidgetLabels
  loading?: boolean
  highlighted?: boolean
  viewerUserId?: string
  untilByScheduleLayer?: OnCallWidgetUntilByScheduleLayer
  className?: string
}

function OnCallLayerRow({
  layer,
  labels,
  scheduleId,
  untilByScheduleLayer,
  users,
  viewerUserId,
}: {
  layer: OnCallWidgetSchedule['layers'][number]
  labels: OnCallWidgetLabels
  scheduleId: string
  untilByScheduleLayer?: OnCallWidgetUntilByScheduleLayer
  users: OnCallWidgetUser[]
  viewerUserId?: string
}): ReactElement {
  const displayName = userLabel(users, layer.userId)
  const avatarUrl = userAvatarUrl(users, layer.userId)
  const isViewerLayer = viewerUserId != null && layer.userId === viewerUserId
  const until =
    isViewerLayer && untilByScheduleLayer
      ? untilByScheduleLayer[scheduleLayerKey(scheduleId, layer.layer)]
      : undefined

  return (
    <div
      className={cn(
        'flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border bg-background p-3',
        isViewerLayer && until && 'border-primary/40 bg-primary/5 ring-1 ring-primary/20',
      )}
      key={`${layer.layer}-${layer.rotationId}`}
    >
      <div className="min-w-0 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <Badge size="sm" variant="outline">
            {layerRoleLabel(layer.layer, labels)}
          </Badge>
          <p className="text-sm font-medium text-foreground">{labels.layerLabel(layer.layer)}</p>
        </div>
        {layer.rotationName ? (
          <p className="truncate text-xs text-muted-foreground">{layer.rotationName}</p>
        ) : null}
        {until && labels.until ? (
          <p className="text-xs text-muted-foreground">{labels.until(until)}</p>
        ) : null}
      </div>
      <Group className="-space-x-2">
        <Avatar className="ring-2 ring-background">
          {avatarUrl ? <AvatarImage alt={displayName} src={avatarUrl} /> : null}
          <AvatarFallback>{initialsFromLabel(displayName)}</AvatarFallback>
        </Avatar>
        <span className="ps-3 text-sm text-foreground">{displayName}</span>
      </Group>
    </div>
  )
}

function OnCallSchedulePanel({
  labels,
  schedule,
  untilByScheduleLayer,
  users,
  viewerUserId,
}: {
  labels: OnCallWidgetLabels
  schedule: OnCallWidgetSchedule
  untilByScheduleLayer?: OnCallWidgetUntilByScheduleLayer
  users: OnCallWidgetUser[]
  viewerUserId?: string
}): ReactElement {
  const layers = sortOnCallLayers(schedule.layers)

  return (
    <div className="space-y-3" data-schedule-id={schedule.id}>
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="text-sm font-medium text-foreground">{schedule.name}</p>
        {schedule.computedAt && labels.computedAt ? (
          <p className="text-xs text-muted-foreground">
            {labels.computedAt} {schedule.computedAt}
          </p>
        ) : null}
      </div>
      {layers.length === 0 ? (
        <p className="text-sm text-muted-foreground">{labels.empty}</p>
      ) : (
        <div className="flex flex-col gap-3">
          {layers.map((layer) => (
            <OnCallLayerRow
              key={`${layer.layer}-${layer.rotationId}`}
              labels={labels}
              layer={layer}
              scheduleId={schedule.id}
              untilByScheduleLayer={untilByScheduleLayer}
              users={users}
              viewerUserId={viewerUserId}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export function OnCallWidget({
  className,
  highlighted = false,
  labels,
  loading = false,
  schedules,
  untilByScheduleLayer,
  users,
  viewerUserId,
}: OnCallWidgetProps): ReactElement {
  const hasSchedules = schedules.length > 0

  return (
    <Frame
      className={cn(className, highlighted && 'ring-2 ring-primary/50')}
      data-on-call-highlighted={highlighted ? 'true' : undefined}
    >
      <div className="flex flex-wrap items-center gap-2 px-5 pt-4">
        <FrameTitle>{labels.title}</FrameTitle>
        {highlighted && labels.youAreOnCall ? (
          <Badge size="sm" variant="default">
            {labels.youAreOnCall}
          </Badge>
        ) : null}
      </div>
      <FrameDescription className="px-5">{labels.title}</FrameDescription>
      <FramePanel>
        {loading ? (
          <p className="text-sm text-muted-foreground">{labels.loading}</p>
        ) : !hasSchedules ? (
          <p className="text-sm text-muted-foreground">{labels.empty}</p>
        ) : (
          <div className="flex flex-col gap-5">
            {schedules.map((schedule) => (
              <OnCallSchedulePanel
                key={schedule.id}
                labels={labels}
                schedule={schedule}
                untilByScheduleLayer={untilByScheduleLayer}
                users={users}
                viewerUserId={viewerUserId}
              />
            ))}
          </div>
        )}
      </FramePanel>
    </Frame>
  )
}
