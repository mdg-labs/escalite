import type { OnCallWidgetLabels, OnCallWidgetLayer, OnCallWidgetUser } from './types'

export function scheduleLayerKey(scheduleId: string, layer: number): string {
  return `${scheduleId}:${layer}`
}

export function initialsFromLabel(label: string): string {
  const parts = label.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) {
    return '?'
  }

  if (parts.length === 1) {
    return parts[0]?.slice(0, 2).toUpperCase() ?? '?'
  }

  return `${parts[0]?.[0] ?? ''}${parts[1]?.[0] ?? ''}`.toUpperCase()
}

export function userLabel(users: OnCallWidgetUser[], userId: string): string {
  return users.find((user) => user.id === userId)?.label ?? userId
}

export function userAvatarUrl(
  users: OnCallWidgetUser[],
  userId: string,
): string | null | undefined {
  return users.find((user) => user.id === userId)?.avatarUrl
}

export function sortOnCallLayers(layers: OnCallWidgetLayer[]): OnCallWidgetLayer[] {
  return [...layers].sort((left, right) => left.layer - right.layer)
}

export function layerRoleLabel(
  layer: number,
  labels: Pick<OnCallWidgetLabels, 'layerLabel' | 'primaryLayer' | 'secondaryLayer'>,
): string {
  if (layer === 1) {
    return labels.primaryLayer
  }

  if (layer === 2) {
    return labels.secondaryLayer
  }

  return labels.layerLabel(layer)
}
