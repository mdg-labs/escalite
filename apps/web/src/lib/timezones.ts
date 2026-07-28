const FALLBACK_IANA_TIMEZONES = [
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'America/Toronto',
  'America/Sao_Paulo',
  'Europe/London',
  'Europe/Berlin',
  'Europe/Paris',
  'Europe/Vienna',
  'Asia/Tokyo',
  'Asia/Singapore',
  'Asia/Kolkata',
  'Australia/Sydney',
] as const

export function isValidIanaTimezone(timezone: string): boolean {
  const trimmed = timezone.trim()
  if (!trimmed) {
    return false
  }

  try {
    Intl.DateTimeFormat(undefined, { timeZone: trimmed })
    return true
  } catch {
    return false
  }
}

export function listIanaTimezones(): string[] {
  if (typeof Intl !== 'undefined' && 'supportedValuesOf' in Intl) {
    const timezones = [...Intl.supportedValuesOf('timeZone')]
    if (!timezones.includes('UTC')) {
      timezones.push('UTC')
    }
    return timezones.sort((left, right) => left.localeCompare(right))
  }

  return [...FALLBACK_IANA_TIMEZONES]
}

export function defaultIanaTimezone(): string {
  const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  return isValidIanaTimezone(browserTimezone) ? browserTimezone : 'UTC'
}

export function filterIanaTimezones(timezones: string[], query: string): string[] {
  const normalized = query.trim().toLowerCase()
  if (!normalized) {
    return timezones
  }

  return timezones.filter((timezone) => timezone.toLowerCase().includes(normalized))
}
