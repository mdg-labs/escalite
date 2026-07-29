import * as Linking from 'expo-linking'
import * as WebBrowser from 'expo-web-browser'

import { mobileLoginUrl } from '@/auth/config'
import { isParsedAuthCallback, mobileAuthRedirectUri } from '@/auth/redirect-uri'
import { getServerEndpoints } from '@/server/endpoints'

export function isAuthCallbackUrl(url: string): boolean {
  return isParsedAuthCallback(Linking.parse(url))
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
  const redirectUri = mobileAuthRedirectUri()
  const result = await WebBrowser.openAuthSessionAsync(mobileLoginUrl(webBaseUrl, redirectUri), redirectUri)
  if (result.type !== 'success' || !result.url) {
    return null
  }

  return parseAuthCodeFromUrl(result.url)
}
