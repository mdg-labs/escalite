import { appConfig } from './config'

export const AUDIT_LOG_PAGE_SIZE = 25

export function dateInputToStartOfDayUTC(value: string): string | undefined {
  const trimmed = value.trim()
  if (!trimmed) {
    return undefined
  }
  return `${trimmed}T00:00:00.000Z`
}

export function dateInputToEndOfDayUTC(value: string): string | undefined {
  const trimmed = value.trim()
  if (!trimmed) {
    return undefined
  }
  return `${trimmed}T23:59:59.999Z`
}

export type AuditLogExportFilters = {
  action?: string
  from?: string
  to?: string
}

export async function downloadAuditLogCsv(filters: AuditLogExportFilters): Promise<void> {
  const url = new URL('/api/v1/audit-events/export.csv', appConfig.apiPublicUrl)

  if (filters.action?.trim()) {
    url.searchParams.set('action', filters.action.trim())
  }
  if (filters.from?.trim()) {
    url.searchParams.set('from', filters.from.trim())
  }
  if (filters.to?.trim()) {
    url.searchParams.set('to', filters.to.trim())
  }

  const response = await fetch(url, { credentials: 'include' })
  if (!response.ok) {
    throw new Error(`export failed (${response.status})`)
  }

  const blob = await response.blob()
  const objectUrl = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = 'audit-events.csv'
  anchor.click()
  URL.revokeObjectURL(objectUrl)
}

export function formatAuditActionLabel(action: string): string {
  return action.replaceAll('.', ' · ')
}
