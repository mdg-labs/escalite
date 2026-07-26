import * as Linking from 'expo-linking'
import * as WebBrowser from 'expo-web-browser'

import { mobileAuthConstants, mobileAuthDeepLink, mobileLoginUrl } from '@/auth/config'
import { getServerEndpoints } from '@/server/endpoints'

export function parseAuthCodeFromUrl(url: string): string | null {
  const parsed = Linking.parse(url)
  if (
    parsed.hostname !== mobileAuthConstants.deepLinkAuthPath &&
    parsed.path !== mobileAuthConstants.deepLinkAuthPath
  ) {
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
