export type RotationFrequency = 'HOURLY' | 'DAILY' | 'WEEKLY'

export type ParsedRRule = {
  frequency: RotationFrequency
  interval: number
}

export function buildRRule(frequency: RotationFrequency, interval: number): string {
  const safeInterval = Math.max(1, Math.floor(interval))
  return `FREQ=${frequency};INTERVAL=${safeInterval}`
}

export function parseRRule(rrule: string): ParsedRRule | null {
  const parts = Object.fromEntries(
    rrule.split(';').map((part) => {
      const [key, value] = part.split('=')
      return [key?.trim().toUpperCase() ?? '', value?.trim() ?? '']
    }),
  )

  const frequency = parts.FREQ
  if (frequency !== 'HOURLY' && frequency !== 'DAILY' && frequency !== 'WEEKLY') {
    return null
  }

  const interval = Number.parseInt(parts.INTERVAL ?? '1', 10)
  if (!Number.isFinite(interval) || interval < 1) {
    return null
  }

  return { frequency, interval }
}
