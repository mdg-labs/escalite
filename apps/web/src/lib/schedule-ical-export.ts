import { appConfig } from './config'
import { scheduleICalFilename } from './schedule-ical-filename'

export { scheduleICalFilename } from './schedule-ical-filename'

export function scheduleICalExportUrl(scheduleId: string): string {
  return new URL(`/api/v1/schedules/${scheduleId}/calendar.ics`, appConfig.apiPublicUrl).toString()
}

export async function downloadScheduleICal(
  scheduleId: string,
  scheduleName: string,
): Promise<void> {
  const response = await fetch(scheduleICalExportUrl(scheduleId), { credentials: 'include' })
  if (!response.ok) {
    throw new Error(`export failed (${response.status})`)
  }

  const blob = await response.blob()
  const objectUrl = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = scheduleICalFilename(scheduleName)
  anchor.click()
  URL.revokeObjectURL(objectUrl)
}
