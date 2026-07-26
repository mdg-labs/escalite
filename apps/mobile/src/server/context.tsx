import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import { setActiveServerEndpoints, resetActiveServerEndpoints } from '@/server/endpoints'
import { probeServerHealth } from '@/server/health'
import {
  clearStoredServerOrigin,
  getStoredServerOrigin,
  setStoredServerOrigin,
} from '@/server/store'
import type { ServerEndpoints } from '@/server/types'
import { deriveServerEndpoints, normalizeServerOrigin } from '@/server/url'

type ServerConfigStatus = 'loading' | 'configured' | 'unconfigured'

type ServerConfigContextValue = {
  status: ServerConfigStatus
  endpoints: ServerEndpoints | null
  error: string | null
  setServerOrigin: (input: string) => Promise<void>
  clearServer: () => Promise<void>
}

const ServerConfigContext = createContext<ServerConfigContextValue | null>(null)

type ServerConfigProviderProps = {
  children: ReactNode
}

export function ServerConfigProvider({ children }: ServerConfigProviderProps) {
  const [status, setStatus] = useState<ServerConfigStatus>('loading')
  const [endpoints, setEndpoints] = useState<ServerEndpoints | null>(null)
  const [error, setError] = useState<string | null>(null)

  const applyEndpoints = useCallback((nextEndpoints: ServerEndpoints) => {
    setActiveServerEndpoints(nextEndpoints)
    setEndpoints(nextEndpoints)
    setStatus('configured')
    setError(null)
  }, [])

  useEffect(() => {
    let cancelled = false

    void (async () => {
      const storedOrigin = await getStoredServerOrigin()
      if (cancelled) {
        return
      }

      if (!storedOrigin) {
        resetActiveServerEndpoints()
        setEndpoints(null)
        setStatus('unconfigured')
        return
      }

      try {
        const nextEndpoints = deriveServerEndpoints(storedOrigin)
        applyEndpoints(nextEndpoints)
      } catch {
        await clearStoredServerOrigin()
        resetActiveServerEndpoints()
        setEndpoints(null)
        setStatus('unconfigured')
      }
    })()

    return () => {
      cancelled = true
    }
  }, [applyEndpoints])

  const setServerOrigin = useCallback(
    async (input: string) => {
      const origin = normalizeServerOrigin(input)
      await probeServerHealth(origin)
      const nextEndpoints = deriveServerEndpoints(origin)
      await setStoredServerOrigin(origin)
      applyEndpoints(nextEndpoints)
    },
    [applyEndpoints],
  )

  const clearServer = useCallback(async () => {
    await clearStoredServerOrigin()
    resetActiveServerEndpoints()
    setEndpoints(null)
    setStatus('unconfigured')
    setError(null)
  }, [])

  const value = useMemo<ServerConfigContextValue>(
    () => ({
      status,
      endpoints,
      error,
      setServerOrigin,
      clearServer,
    }),
    [status, endpoints, error, setServerOrigin, clearServer],
  )

  return <ServerConfigContext.Provider value={value}>{children}</ServerConfigContext.Provider>
}

export function useServerConfig(): ServerConfigContextValue {
  const context = useContext(ServerConfigContext)
  if (!context) {
    throw new Error('useServerConfig must be used within ServerConfigProvider')
  }
  return context
}
