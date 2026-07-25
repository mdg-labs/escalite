const apiBaseUrl = process.env.EXPO_PUBLIC_ESCALITE_API_URL ?? 'http://localhost:8080'
const webBaseUrl = process.env.EXPO_PUBLIC_ESCALITE_WEB_URL ?? 'http://localhost:5173'

export const mobileAuthConfig = {
  apiBaseUrl,
  webBaseUrl,
  mobileLoginPath: '/login/mobile',
  deepLinkScheme: 'escalite',
  deepLinkAuthPath: 'auth',
  refreshTokenStorageKey: 'escalite.mobile.refresh_token',
} as const

export function mobileLoginUrl(): string {
  return `${mobileAuthConfig.webBaseUrl}${mobileAuthConfig.mobileLoginPath}`
}

export function mobileAuthDeepLink(): string {
  return `${mobileAuthConfig.deepLinkScheme}://${mobileAuthConfig.deepLinkAuthPath}`
}
