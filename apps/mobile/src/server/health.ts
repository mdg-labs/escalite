export async function probeServerHealth(origin: string): Promise<void> {
  const healthUrl = `${origin.replace(/\/$/, '')}/healthz`
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), 10_000)

  try {
    const response = await fetch(healthUrl, {
      method: 'GET',
      signal: controller.signal,
    })

    if (!response.ok) {
      throw new Error(`Server health check failed (${response.status})`)
    }
  } catch (err) {
    if (err instanceof Error && err.name === 'AbortError') {
      throw new Error('Server health check timed out')
    }
    throw err instanceof Error ? err : new Error('Unable to reach Escalite server')
  } finally {
    clearTimeout(timeout)
  }
}
