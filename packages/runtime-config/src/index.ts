export type EscaliteRuntimeConfig = {
  graphqlUrl?: string
  apiPublicUrl?: string
  statusPollIntervalMs?: string
}

declare global {
  interface Window {
    __ESCALITE_RUNTIME__?: EscaliteRuntimeConfig
  }
}

export function getRuntimeConfig(): EscaliteRuntimeConfig | undefined {
  if (typeof window === 'undefined') {
    return undefined
  }

  return window.__ESCALITE_RUNTIME__
}

function readRuntimeStringFrom(
  runtime: EscaliteRuntimeConfig | undefined,
  key: keyof EscaliteRuntimeConfig,
  viteValue: string | undefined,
  defaultValue: string,
): string {
  const runtimeValue = runtime?.[key]?.trim()
  if (runtimeValue) {
    return runtimeValue
  }

  const buildValue = viteValue?.trim()
  if (buildValue) {
    return buildValue
  }

  return defaultValue
}

export function resolveGraphqlUrl(viteValue?: string): string {
  return readRuntimeStringFrom(
    getRuntimeConfig(),
    'graphqlUrl',
    viteValue,
    '/graphql',
  )
}

export function resolveApiPublicUrl(viteValue?: string): string {
  const runtime = getRuntimeConfig()
  const runtimeValue = runtime?.apiPublicUrl?.trim()
  if (runtimeValue) {
    return runtimeValue
  }

  const buildValue = viteValue?.trim()
  if (buildValue) {
    return buildValue
  }

  if (typeof window !== 'undefined') {
    return window.location.origin
  }

  return ''
}

export function resolveStatusPollIntervalMs(viteValue?: string): number {
  const raw = readRuntimeStringFrom(
    getRuntimeConfig(),
    'statusPollIntervalMs',
    viteValue,
    '60000',
  )
  const parsed = Number.parseInt(raw, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 60_000
}

/** @internal Test helper */
export function resolveGraphqlUrlFrom(
  runtime: EscaliteRuntimeConfig | undefined,
  viteValue?: string,
): string {
  return readRuntimeStringFrom(runtime, 'graphqlUrl', viteValue, '/graphql')
}

/** @internal Test helper */
export function resolveApiPublicUrlFrom(
  runtime: EscaliteRuntimeConfig | undefined,
  viteValue?: string,
  origin = 'https://app.example.com',
): string {
  const runtimeValue = runtime?.apiPublicUrl?.trim()
  if (runtimeValue) {
    return runtimeValue
  }

  const buildValue = viteValue?.trim()
  if (buildValue) {
    return buildValue
  }

  return origin
}

/** @internal Test helper */
export function resolveStatusPollIntervalMsFrom(
  runtime: EscaliteRuntimeConfig | undefined,
  viteValue?: string,
): number {
  const raw = readRuntimeStringFrom(runtime, 'statusPollIntervalMs', viteValue, '60000')
  const parsed = Number.parseInt(raw, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 60_000
}
