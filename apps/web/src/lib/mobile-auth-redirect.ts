export const DEFAULT_MOBILE_AUTH_REDIRECT_URI = 'escalite://auth'

function normalizeAuthPath(pathname: string, hostname: string): string {
  const path = pathname && pathname !== '/' ? pathname : `/${hostname}`
  return path.replace(/^\/+|\/+$/g, '')
}

function pathEndsWithAuthSegment(path: string): boolean {
  return path === 'auth' || path.endsWith('/auth') || path.endsWith('/--/auth')
}

export function isAllowedMobileAuthRedirectUri(uri: string): boolean {
  try {
    const parsed = new URL(uri)
    const path = normalizeAuthPath(parsed.pathname, parsed.hostname)

    if (parsed.protocol === 'escalite:') {
      return pathEndsWithAuthSegment(path)
    }

    if (parsed.protocol === 'exp:') {
      return pathEndsWithAuthSegment(path)
    }

    // Expo development client: exp+escalite://auth, etc.
    if (parsed.protocol.startsWith('exp+') && parsed.protocol.endsWith(':')) {
      return pathEndsWithAuthSegment(path)
    }

    return false
  } catch {
    return false
  }
}

export function resolveMobileAuthRedirectUri(raw: string | null | undefined): string {
  if (!raw || !isAllowedMobileAuthRedirectUri(raw)) {
    return DEFAULT_MOBILE_AUTH_REDIRECT_URI
  }
  return raw
}

export function buildMobileAuthRedirectUrl(redirectUri: string, code: string): string {
  const url = new URL(resolveMobileAuthRedirectUri(redirectUri))
  url.searchParams.set('code', code)
  return url.toString()
}
