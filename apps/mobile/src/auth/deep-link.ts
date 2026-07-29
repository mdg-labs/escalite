import * as Linking from 'expo-linking'
import type { ParsedURL } from 'expo-linking'
import * as WebBrowser from 'expo-web-browser'

import { mobileAuthConstants, mobileAuthDeepLink, mobileLoginUrl } from '@/auth/config'
import { getServerEndpoints } from '@/server/endpoints'

function normalizeAuthPath(path: string | null | undefined): string {
  if (!path) {
    return ''
  }
  return path.replace(/^\/+|\/+$/g, '')
}

export function isAuthCallbackUrl(url: string): boolean {
  return isParsedAuthCallback(Linking.parse(url))
}

function isParsedAuthCallback(parsed: ParsedURL): boolean {
  const authSegment = mobileAuthConstants.deepLinkAuthPath
  if (parsed.scheme && parsed.scheme !== mobileAuthConstants.deepLinkScheme) {
    return false
  }
  if (parsed.hostname === authSegment) {
    return true
  }
  const normalizedPath = normalizeAuthPath(parsed.path)
  return normalizedPath === authSegment || normalizedPath.endsWith(`/${authSegment}`)
}

export function parseAuthCodeFromUrl(url: string): string | null {
  const parsed = Linking.parse(url)
  if (!isParsedAuthCallback(parsed)) {
    return null
  }

  const code = parsed.queryParams?.code
  if (typeof code === 'string' && code.length > 0) {
    return code
  }
  if (Array.isArray(code) && typeof code[0] === 'string' && code[0].length > 0) {
    return code[0]
  }
  return null
}

export async function openMobileLoginSession(): Promise<string | null> {
  WebBrowser.maybeCompleteAuthSession()

  const { webBaseUrl } = getServerEndpoints()
  const result = await WebBrowser.openAuthSessionAsync(mobileLoginUrl(webBaseUrl), mobileAuthDeepLink())
  if (result.type !== 'success' || !result.url) {
    return null
  }

  return parseAuthCodeFromUrl(result.url)
}
