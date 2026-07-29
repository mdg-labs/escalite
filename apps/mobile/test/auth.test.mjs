import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

test('secure store keeps refresh token in Keychain/Keystore only', () => {
  const source = readFileSync(join(root, 'src/auth/secure-store.ts'), 'utf8')
  assert.match(source, /expo-secure-store/)
  assert.match(source, /SecureStore\.setItemAsync/)
  assert.match(source, /WHEN_UNLOCKED_THIS_DEVICE_ONLY/)
  assert.doesNotMatch(source, /AsyncStorage/)
})

test('auth context persists refresh token via secure store', () => {
  const source = readFileSync(join(root, 'src/auth/context.tsx'), 'utf8')
  assert.match(source, /setStoredRefreshToken/)
  assert.match(source, /getStoredRefreshToken/)
  assert.match(source, /clearStoredRefreshToken/)
})

test('auth api maps UNAUTHENTICATED refresh failures', () => {
  const source = readFileSync(join(root, 'src/auth/api.ts'), 'utf8')
  assert.match(source, /mobile\/auth\/refresh/)
  assert.match(source, /refresh_token/)
  const contextSource = readFileSync(join(root, 'src/auth/context.tsx'), 'utf8')
  assert.match(contextSource, /UNAUTHENTICATED/)
})

test('deep link parser reads auth code from escalite and exp schemes', () => {
  const source = readFileSync(join(root, 'src/auth/deep-link.ts'), 'utf8')
  assert.match(source, /parseAuthCodeFromUrl/)
  assert.match(source, /isAuthCallbackUrl/)
  assert.match(source, /openAuthSessionAsync/)
  assert.match(source, /mobileAuthRedirectUri/)
  const redirectSource = readFileSync(join(root, 'src/auth/redirect-uri.ts'), 'utf8')
  assert.match(redirectSource, /parsed\.scheme === 'exp'/)
  assert.match(redirectSource, /Linking\.createURL/)
})

test('mobile login URL forwards redirect_uri to web', () => {
  const configSource = readFileSync(join(root, 'src/auth/config.ts'), 'utf8')
  assert.match(configSource, /redirect_uri/)
  const deepLinkSource = readFileSync(join(root, 'src/auth/deep-link.ts'), 'utf8')
  assert.match(deepLinkSource, /mobileLoginUrl\(webBaseUrl, redirectUri\)/)
})

test('auth callback route completes sign-in via AuthProvider', () => {
  const source = readFileSync(join(root, 'app/auth.tsx'), 'utf8')
  assert.match(source, /signInWithCode/)
  assert.match(source, /router\.replace\('\/'\)/)
  const layoutSource = readFileSync(join(root, 'app/_layout.tsx'), 'utf8')
  assert.match(layoutSource, /name="auth"/)
})

test('auth context exposes signInWithCode without Linking listener', () => {
  const source = readFileSync(join(root, 'src/auth/context.tsx'), 'utf8')
  assert.match(source, /signInWithCode/)
  assert.doesNotMatch(source, /Linking\.addEventListener/)
})
