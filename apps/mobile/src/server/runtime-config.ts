import type { EscaliteRuntimeConfig } from '@escalite/runtime-config'

export async function fetchEscaliteRuntimeConfig(webOrigin: string): Promise<EscaliteRuntimeConfig> {
  const configUrl = `${webOrigin.replace(/\/$/, '')}/runtime-config.js`
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), 10_000)

  try {
    const response = await fetch(configUrl, {
      method: 'GET',
      signal: controller.signal,
      cache: 'no-store',
    })

    if (!response.ok) {
      throw new Error(`Unable to load server configuration (${response.status})`)
    }

    const text = await response.text()
    return parseRuntimeConfigScript(text)
  } catch (err) {
    if (err instanceof Error && err.name === 'AbortError') {
      throw new Error('Server configuration request timed out')
    }
    throw err instanceof Error ? err : new Error('Unable to load server configuration')
  } finally {
    clearTimeout(timeout)
  }
}

function parseRuntimeConfigScript(text: string): EscaliteRuntimeConfig {
  const apiPublicUrl = extractRuntimeConfigValue(text, 'apiPublicUrl')
  const graphqlUrl = extractRuntimeConfigValue(text, 'graphqlUrl')
  const statusPollIntervalMs = extractRuntimeConfigValue(text, 'statusPollIntervalMs')

  return {
    apiPublicUrl,
    graphqlUrl,
    statusPollIntervalMs,
  }
}

function extractRuntimeConfigValue(text: string, key: string): string | undefined {
  const match = text.match(new RegExp(`${key}:\\s*"([^"]*)"`, 'i'))
  return match?.[1]
}
