export type AlertCardStatus = string

export type AlertCardPriority = string

export type AlertCardAlert = {
  id: string
  summary: string
  description?: string | null
  status: AlertCardStatus
  priority: AlertCardPriority
  eventCount: number
  updatedAt: string
}

export type AlertCardLabels = {
  statusLabel: (status: AlertCardStatus) => string
  priorityLabel: (priority: AlertCardPriority) => string
}
