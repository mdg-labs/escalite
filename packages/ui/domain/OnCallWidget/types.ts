export type OnCallWidgetUser = {
  id: string
  label: string
  avatarUrl?: string | null
}

export type OnCallWidgetLayer = {
  layer: number
  rotationId: string
  rotationName?: string
  userId: string
}

export type OnCallWidgetSchedule = {
  id: string
  name: string
  layers: OnCallWidgetLayer[]
  computedAt?: string
}

export type OnCallWidgetLabels = {
  title: string
  loading: string
  empty: string
  layerLabel: (layer: number) => string
  primaryLayer: string
  secondaryLayer: string
  computedAt?: string
  youAreOnCall?: string
  until?: (until: string) => string
}

export type OnCallWidgetUntilByScheduleLayer = Record<string, string>
