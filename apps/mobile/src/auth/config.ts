export const mobileAuthConstants = {
  mobileLoginPath: '/login/mobile',
  deepLinkScheme: 'escalite',
  deepLinkAuthPath: 'auth',
  refreshTokenStorageKey: 'escalite.mobile.refresh_token',
} as const

export function mobileLoginUrl(webBaseUrl: string): string {
  return `${webBaseUrl}${mobileAuthConstants.mobileLoginPath}`
}

export function mobileAuthDeepLink(): string {
  return `${mobileAuthConstants.deepLinkScheme}://${mobileAuthConstants.deepLinkAuthPath}`
}
