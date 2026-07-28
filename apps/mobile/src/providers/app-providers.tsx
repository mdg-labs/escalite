import { Provider as UrqlProvider, cacheExchange, createClient, fetchExchange } from 'urql'
import { useMemo, type ReactNode } from 'react'

import { AuthProvider } from '@/auth/context'
import { useServerConfig } from '@/server/context'

type AppProvidersProps = {
  children: ReactNode
}

export function AppProviders({ children }: AppProvidersProps) {
  const { endpoints } = useServerConfig()

  const urqlClient = useMemo(
    () =>
      createClient({
        url: endpoints?.graphqlUrl ?? 'http://localhost:8080/graphql',
        exchanges: [cacheExchange, fetchExchange],
      }),
    [endpoints?.graphqlUrl],
  )

  return (
    <UrqlProvider value={urqlClient}>
      <AuthProvider>{children}</AuthProvider>
    </UrqlProvider>
  )
}
