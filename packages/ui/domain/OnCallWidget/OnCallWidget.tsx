'use client'

import type { ReactElement } from 'react'

import { Avatar, AvatarFallback, AvatarImage } from '../../primitives/avatar'
import { Badge } from '../../primitives/badge'
import { Frame, FrameDescription, FramePanel, FrameTitle } from '../../primitives/frame'
import { Group } from '../../primitives/group'
import { cn } from '../../lib/utils'
import type { OnCallWidgetLabels, OnCallWidgetSchedule, OnCallWidgetUser } from './types'
import {
  initialsFromLabel,
  layerRoleLabel,
  sortOnCallLayers,
  userAvatarUrl,
  userLabel,
} from './utils'

export type OnCallWidgetProps = {
  schedules: OnCallWidgetSchedule[]
  users: OnCallWidgetUser[]
  labels: OnCallWidgetLabels
  loading?: boolean
  className?: string
}

function OnCallLayerRow({
  layer,
  labels,
  users,
}: {
  layer: OnCallWidgetSchedule['layers'][number]
  labels: OnCallWidgetLabels
  users: OnCallWidgetUser[]
}): ReactElement {
  const displayName = userLabel(users, layer.userId)
  const avatarUrl = userAvatarUrl(users, layer.userId)

  return (
    <div
      className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border bg-background p-3"
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
  users,
}: {
  labels: OnCallWidgetLabels
  schedule: OnCallWidgetSchedule
  users: OnCallWidgetUser[]
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
            <OnCallLayerRow key={`${layer.layer}-${layer.rotationId}`} labels={labels} layer={layer} users={users} />
          ))}
        </div>
      )}
    </div>
  )
}

export function OnCallWidget({
  className,
  labels,
  loading = false,
  schedules,
  users,
}: OnCallWidgetProps): ReactElement {
  const hasSchedules = schedules.length > 0

  return (
    <Frame className={cn(className)}>
      <FrameTitle>{labels.title}</FrameTitle>
      <FrameDescription>{labels.title}</FrameDescription>
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
                users={users}
              />
            ))}
          </div>
        )}
      </FramePanel>
    </Frame>
  )
}
