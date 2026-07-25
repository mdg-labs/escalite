import * as Linking from 'expo-linking'
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import { MobileAuthError, exchangeMobileAuthCode, refreshMobileSession } from '@/auth/api'
import { openMobileLoginSession, parseAuthCodeFromUrl } from '@/auth/deep-link'
import {
  clearStoredRefreshToken,
  getStoredRefreshToken,
  setStoredRefreshToken,
} from '@/auth/secure-store'
import type { MobileAuthUser } from '@/auth/api'

type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated'

type AuthContextValue = {
  status: AuthStatus
  user: MobileAuthUser | null
  error: string | null
  signIn: () => Promise<void>
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

type AuthProviderProps = {
  children: ReactNode
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [user, setUser] = useState<MobileAuthUser | null>(null)
  const [error, setError] = useState<string | null>(null)

  const completeSignIn = useCallback(async (code: string) => {
    const { refreshToken, user: nextUser } = await exchangeMobileAuthCode(code)
    await setStoredRefreshToken(refreshToken)
    setUser(nextUser)
    setStatus('authenticated')
    setError(null)
  }, [])

  const restoreSession = useCallback(async () => {
    const refreshToken = await getStoredRefreshToken()
    if (!refreshToken) {
      setStatus('unauthenticated')
      setUser(null)
      return
    }

    try {
      const nextUser = await refreshMobileSession(refreshToken)
      setUser(nextUser)
      setStatus('authenticated')
      setError(null)
    } catch (err) {
      await clearStoredRefreshToken()
      setUser(null)
      setStatus('unauthenticated')
      if (err instanceof MobileAuthError && err.code === 'UNAUTHENTICATED') {
        setError('Session expired. Sign in again.')
      } else {
        setError(err instanceof Error ? err.message : 'Unable to restore session')
      }
    }
  }, [])

  useEffect(() => {
    void restoreSession()
  }, [restoreSession])

  useEffect(() => {
    const handleUrl = (url: string) => {
      const code = parseAuthCodeFromUrl(url)
      if (!code) {
        return
      }
      void completeSignIn(code).catch((err: unknown) => {
        setStatus('unauthenticated')
        setError(err instanceof Error ? err.message : 'Sign in failed')
      })
    }

    const subscription = Linking.addEventListener('url', ({ url }) => {
      handleUrl(url)
    })

    void Linking.getInitialURL().then((url) => {
      if (url) {
        handleUrl(url)
      }
    })

    return () => {
      subscription.remove()
    }
  }, [completeSignIn])

  const signIn = useCallback(async () => {
    setError(null)
    setStatus('loading')

    try {
      const code = await openMobileLoginSession()
      if (!code) {
        setStatus(user ? 'authenticated' : 'unauthenticated')
        return
      }
      await completeSignIn(code)
    } catch (err) {
      setStatus('unauthenticated')
      setError(err instanceof Error ? err.message : 'Sign in failed')
    }
  }, [completeSignIn, user])

  const signOut = useCallback(async () => {
    await clearStoredRefreshToken()
    setUser(null)
    setStatus('unauthenticated')
    setError(null)
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      error,
      signIn,
      signOut,
    }),
    [status, user, error, signIn, signOut],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
