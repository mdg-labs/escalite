export const mobileAuthConstants = {
  mobileLoginPath: '/login/mobile',
  deepLinkScheme: 'escalite',
  deepLinkAuthPath: 'auth',
  refreshTokenStorageKey: 'escalite.mobile.refresh_token',
} as const

export function mobileLoginUrl(webBaseUrl: string, redirectUri: string): string {
  const base = webBaseUrl.replace(/\/$/, '')
  const url = new URL(`${base}${mobileAuthConstants.mobileLoginPath}`)
  url.searchParams.set('redirect_uri', redirectUri)
  return url.toString()
}
