import { mobileAuthConfig } from '@/auth/config'

export type MobileAuthUser = {
  id: string
  email: string
  role: string
}

type ApiErrorBody = {
  code?: string
  error?: string
}

export class MobileAuthError extends Error {
  readonly code: string

  constructor(code: string, message: string) {
    super(message)
    this.name = 'MobileAuthError'
    this.code = code
  }
}

async function parseApiError(response: Response): Promise<MobileAuthError> {
  let body: ApiErrorBody = {}
  try {
    body = (await response.json()) as ApiErrorBody
  } catch {
    // ignore JSON parse failures
  }

  return new MobileAuthError(body.code ?? 'UNKNOWN', body.error ?? 'request failed')
}

export async function exchangeMobileAuthCode(code: string): Promise<{
  refreshToken: string
  user: MobileAuthUser
}> {
  const response = await fetch(`${mobileAuthConfig.apiBaseUrl}/api/v1/mobile/auth/exchange`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code }),
  })

  if (!response.ok) {
    throw await parseApiError(response)
  }

  const body = (await response.json()) as {
    refresh_token: string
    user: MobileAuthUser
  }

  return {
    refreshToken: body.refresh_token,
    user: body.user,
  }
}

export async function refreshMobileSession(refreshToken: string): Promise<MobileAuthUser> {
  const response = await fetch(`${mobileAuthConfig.apiBaseUrl}/api/v1/mobile/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  })

  if (!response.ok) {
    throw await parseApiError(response)
  }

  const body = (await response.json()) as { user: MobileAuthUser }
  return body.user
}
