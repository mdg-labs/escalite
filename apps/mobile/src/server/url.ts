import {
  resolveApiPublicUrlFrom,
  resolveGraphqlUrlFrom,
  type EscaliteRuntimeConfig,
} from '@escalite/runtime-config'

import type { ServerEndpoints } from '@/server/types'

const defaultWebBaseUrl = process.env.EXPO_PUBLIC_ESCALITE_WEB_URL ?? 'http://localhost:5173'
const defaultApiBaseUrl = process.env.EXPO_PUBLIC_ESCALITE_API_URL?.trim()

export function defaultDevEndpoints(): ServerEndpoints {
  const webOrigin = defaultWebBaseUrl.replace(/\/$/, '')
  return resolveServerEndpoints(webOrigin, {
    apiPublicUrl: defaultApiBaseUrl ?? '',
    graphqlUrl: defaultApiBaseUrl ? `${defaultApiBaseUrl.replace(/\/$/, '')}/graphql` : '/graphql',
  })
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

export function resolveServerEndpoints(
  webOrigin: string,
  runtime?: EscaliteRuntimeConfig,
): ServerEndpoints {
  const normalizedWebOrigin = normalizeServerOrigin(webOrigin)
  const apiPublicUrl = resolveApiPublicUrlFrom(
    runtime,
    defaultApiBaseUrl,
    normalizedWebOrigin,
  ).replace(/\/$/, '')
  const rawGraphqlUrl = resolveGraphqlUrlFrom(runtime, undefined)
  const graphqlUrl = resolveAbsoluteGraphqlUrl(rawGraphqlUrl, normalizedWebOrigin)

  return {
    origin: normalizedWebOrigin,
    webBaseUrl: normalizedWebOrigin,
    apiBaseUrl: apiPublicUrl,
    graphqlUrl,
  }
}

function resolveAbsoluteGraphqlUrl(rawGraphqlUrl: string, webOrigin: string): string {
  if (/^https?:\/\//i.test(rawGraphqlUrl)) {
    return rawGraphqlUrl.replace(/\/$/, '')
  }

  const path = rawGraphqlUrl.startsWith('/') ? rawGraphqlUrl : `/${rawGraphqlUrl}`
  return `${webOrigin}${path}`
}
