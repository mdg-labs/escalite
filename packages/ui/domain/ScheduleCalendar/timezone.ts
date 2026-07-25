export type ViewerLocalTime = {
  formatted: string
  timezoneLabel: string
}

export function formatViewerLocalTime(
  iso: string,
  locale?: string,
  timeZone?: string,
): ViewerLocalTime {
  const date = new Date(iso)
  const resolvedTimeZone =
    timeZone ?? Intl.DateTimeFormat().resolvedOptions().timeZone ?? 'UTC'
  const resolvedLocale = locale ?? undefined

  if (Number.isNaN(date.getTime())) {
    return {
      formatted: iso,
      timezoneLabel: resolvedTimeZone,
    }
  }

  const formatted = new Intl.DateTimeFormat(resolvedLocale, {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: resolvedTimeZone,
  }).format(date)

  return {
    formatted,
    timezoneLabel: resolvedTimeZone,
  }
}

export function formatViewerLocalDateRange(
  startsAt: string,
  endsAt: string,
  locale?: string,
  timeZone?: string,
): string {
  const start = formatViewerLocalTime(startsAt, locale, timeZone)
  const end = formatViewerLocalTime(endsAt, locale, timeZone)
  return `${start.formatted} – ${end.formatted}`
}
