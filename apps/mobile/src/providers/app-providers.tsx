import { Provider as UrqlProvider, cacheExchange, createClient, fetchExchange } from 'urql'
import { TamaguiProvider, Theme } from 'tamagui'
import { useMemo, type ReactNode } from 'react'

import { AuthProvider } from '@/auth/context'
import { useServerConfig } from '@/server/context'
import { DEFAULT_THEME, tamaguiConfig } from '@/theme/tamagui.config'

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
      <TamaguiProvider config={tamaguiConfig} defaultTheme={DEFAULT_THEME}>
        <Theme name={DEFAULT_THEME}>
          <AuthProvider>{children}</AuthProvider>
        </Theme>
      </TamaguiProvider>
    </UrqlProvider>
  )
}
