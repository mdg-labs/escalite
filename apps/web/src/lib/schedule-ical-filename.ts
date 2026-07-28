export function scheduleICalFilename(scheduleName: string): string {
  const slug = scheduleName
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60)

  return slug.length > 0 ? `${slug}.ics` : 'schedule.ics'
}
