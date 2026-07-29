import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  buildMobileAuthRedirectUrl,
  DEFAULT_MOBILE_AUTH_REDIRECT_URI,
  isAllowedMobileAuthRedirectUri,
  resolveMobileAuthRedirectUri,
} from '../src/lib/mobile-auth-redirect.ts'

describe('mobile auth redirect URI', () => {
  it('allows production escalite scheme', () => {
    assert.equal(isAllowedMobileAuthRedirectUri('escalite://auth'), true)
  })

  it('allows Expo Go exp scheme', () => {
    assert.equal(isAllowedMobileAuthRedirectUri('exp://192.168.1.10:8081/--/auth'), true)
  })

  it('allows Expo development client scheme', () => {
    assert.equal(isAllowedMobileAuthRedirectUri('exp+escalite://auth'), true)
  })

  it('rejects http and https open redirects', () => {
    assert.equal(isAllowedMobileAuthRedirectUri('https://evil.example/auth'), false)
    assert.equal(isAllowedMobileAuthRedirectUri('http://localhost:5173/login/mobile'), false)
  })

  it('rejects unrelated custom schemes', () => {
    assert.equal(isAllowedMobileAuthRedirectUri('otherapp://auth'), false)
  })

  it('falls back to escalite://auth when redirect_uri is missing or invalid', () => {
    assert.equal(resolveMobileAuthRedirectUri(null), DEFAULT_MOBILE_AUTH_REDIRECT_URI)
    assert.equal(resolveMobileAuthRedirectUri('https://evil.example'), DEFAULT_MOBILE_AUTH_REDIRECT_URI)
  })

  it('builds redirect URLs with auth code', () => {
    assert.equal(
      buildMobileAuthRedirectUrl('escalite://auth', 'abc123'),
      'escalite://auth?code=abc123',
    )
    assert.equal(
      buildMobileAuthRedirectUrl('exp://192.168.1.10:8081/--/auth', 'abc123'),
      'exp://192.168.1.10:8081/--/auth?code=abc123',
    )
  })
})
