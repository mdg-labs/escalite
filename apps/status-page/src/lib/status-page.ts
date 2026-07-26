import type { BadgeProps } from '@escalite/ui'
import type { LucideIcon } from 'lucide-react'
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  CircleAlertIcon,
  CircleXIcon,
} from 'lucide-react'

import type { MessageKey } from './i18n'

export type ComponentStatus =
  | 'operational'
  | 'degraded'
  | 'partial_outage'
  | 'major_outage'

export type IncidentStatus = 'investigating' | 'identified' | 'monitoring' | 'resolved'

export type PublicStatusPageComponent = {
  id: string
  name: string
  description?: string | null
  status: ComponentStatus
  position: number
}

export type PublicStatusPageIncidentUpdate = {
  id: string
  body: string
  status: IncidentStatus
  createdAt: string
}

export type PublicStatusPageIncident = {
  id: string
  title: string
  status: IncidentStatus
  affectedComponentIds: string[]
  updates: PublicStatusPageIncidentUpdate[]
  createdAt: string
  resolvedAt?: string | null
}

export type PublicStatusPagePayload = {
  title: string
  overallStatus: ComponentStatus
  components: PublicStatusPageComponent[]
  incidents: PublicStatusPageIncident[]
}

export type PublicStatusPageError = {
  code?: string
  message?: string
}

const COMPONENT_STATUS_RANK: Record<ComponentStatus, number> = {
  operational: 1,
  degraded: 2,
  partial_outage: 3,
  major_outage: 4,
}

export function sortComponents(
  components: PublicStatusPageComponent[],
): PublicStatusPageComponent[] {
  return [...components].sort((left, right) => left.position - right.position)
}

export function componentStatusLabel(status: ComponentStatus): MessageKey {
  switch (status) {
    case 'degraded':
      return 'statusPage.component.status.degraded'
    case 'partial_outage':
      return 'statusPage.component.status.partial_outage'
    case 'major_outage':
      return 'statusPage.component.status.major_outage'
    default:
      return 'statusPage.component.status.operational'
  }
}

export function componentStatusBadgeVariant(status: ComponentStatus): BadgeProps['variant'] {
  switch (status) {
    case 'major_outage':
      return 'destructive'
    case 'partial_outage':
      return 'error'
    case 'degraded':
      return 'warning'
    default:
      return 'success'
  }
}

export function componentStatusIcon(status: ComponentStatus): LucideIcon {
  switch (status) {
    case 'major_outage':
      return CircleXIcon
    case 'partial_outage':
      return CircleAlertIcon
    case 'degraded':
      return AlertTriangleIcon
    default:
      return CheckCircle2Icon
  }
}

export function overallStatusLabel(status: ComponentStatus): MessageKey {
  switch (status) {
    case 'degraded':
      return 'statusPage.overall.degraded'
    case 'partial_outage':
      return 'statusPage.overall.partial_outage'
    case 'major_outage':
      return 'statusPage.overall.major_outage'
    default:
      return 'statusPage.overall.operational'
  }
}

export function overallStatusAlertVariant(
  status: ComponentStatus,
): 'success' | 'warning' | 'error' | 'default' {
  switch (status) {
    case 'major_outage':
    case 'partial_outage':
      return 'error'
    case 'degraded':
      return 'warning'
    case 'operational':
      return 'success'
    default:
      return 'default'
  }
}

export function incidentStatusLabel(status: IncidentStatus): MessageKey {
  switch (status) {
    case 'identified':
      return 'statusPage.incident.status.identified'
    case 'monitoring':
      return 'statusPage.incident.status.monitoring'
    case 'resolved':
      return 'statusPage.incident.status.resolved'
    default:
      return 'statusPage.incident.status.investigating'
  }
}

export function incidentStatusBadgeVariant(status: IncidentStatus): BadgeProps['variant'] {
  switch (status) {
    case 'resolved':
      return 'secondary'
    case 'monitoring':
      return 'info'
    case 'identified':
      return 'warning'
    default:
      return 'destructive'
  }
}

export function worstComponentStatus(
  components: PublicStatusPageComponent[],
): ComponentStatus {
  let worst: ComponentStatus = 'operational'
  for (const component of components) {
    if (COMPONENT_STATUS_RANK[component.status] > COMPONENT_STATUS_RANK[worst]) {
      worst = component.status
    }
  }
  return worst
}

export function publicStatusPageUrl(slug: string, apiPublicUrl: string): string {
  const normalizedSlug = encodeURIComponent(slug.trim().toLowerCase())
  return `${apiPublicUrl.replace(/\/$/, '')}/api/v1/public/status/${normalizedSlug}`
}

export function publicStatusPageSubscribeUrl(slug: string, apiPublicUrl: string): string {
  return `${publicStatusPageUrl(slug, apiPublicUrl)}/subscribe`
}

export async function fetchPublicStatusPage(
  slug: string,
  apiPublicUrl: string,
): Promise<PublicStatusPagePayload> {
  const response = await fetch(publicStatusPageUrl(slug, apiPublicUrl), {
    headers: {
      Accept: 'application/json',
    },
  })

  if (response.status === 404) {
    throw new StatusPageNotFoundError()
  }

  if (!response.ok) {
    throw new StatusPageLoadError()
  }

  return (await response.json()) as PublicStatusPagePayload
}

export async function subscribeToStatusPage(
  slug: string,
  email: string,
  apiPublicUrl: string,
): Promise<void> {
  const response = await fetch(publicStatusPageSubscribeUrl(slug, apiPublicUrl), {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email }),
  })

  if (!response.ok) {
    throw new StatusPageSubscribeError()
  }
}

export class StatusPageNotFoundError extends Error {
  constructor() {
    super('not found')
    this.name = 'StatusPageNotFoundError'
  }
}

export class StatusPageLoadError extends Error {
  constructor() {
    super('load failed')
    this.name = 'StatusPageLoadError'
  }
}

export class StatusPageSubscribeError extends Error {
  constructor() {
    super('subscribe failed')
    this.name = 'StatusPageSubscribeError'
  }
}
