import type { ServerEndpoints } from '@/server/types'

const defaultWebBaseUrl = process.env.EXPO_PUBLIC_ESCALITE_WEB_URL ?? 'http://localhost:5173'

export function defaultDevEndpoints(): ServerEndpoints {
  const origin = defaultWebBaseUrl.replace(/\/$/, '')
  return deriveServerEndpoints(origin)
}

export function normalizeServerOrigin(input: string): string {
  const trimmed = input.trim()
  if (!trimmed) {
    throw new Error('Enter your Escalite server URL')
  }

  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`
  let parsed: URL
  try {
    parsed = new URL(withScheme)
  } catch {
    throw new Error('Enter a valid URL such as https://escalite.example.com')
  }

  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error('Only http and https URLs are supported')
  }

  if (!parsed.hostname) {
    throw new Error('Enter a valid server hostname')
  }

  return `${parsed.protocol}//${parsed.host}`
}

// API and web share one origin here. Split-host deployments (API on a different
// public host than the frontend) require the frontend to proxy /api and /graphql,
// or a future server-setup field for a separate API URL.
export function deriveServerEndpoints(origin: string): ServerEndpoints {
  const normalizedOrigin = normalizeServerOrigin(origin)
  return {
    origin: normalizedOrigin,
    apiBaseUrl: normalizedOrigin,
    webBaseUrl: normalizedOrigin,
    graphqlUrl: `${normalizedOrigin}/graphql`,
  }
}
