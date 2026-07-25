import { Provider as UrqlProvider, cacheExchange, createClient, fetchExchange } from 'urql'
import { TamaguiProvider, Theme } from 'tamagui'
import type { ReactNode } from 'react'

import { AuthProvider } from '@/auth/context'
import { DEFAULT_THEME, tamaguiConfig } from '@/theme/tamagui.config'

const urqlClient = createClient({
  url: process.env.EXPO_PUBLIC_ESCALITE_GRAPHQL_URL ?? 'http://localhost:8080/graphql',
  exchanges: [cacheExchange, fetchExchange],
})

type AppProvidersProps = {
  children: ReactNode
}

export function AppProviders({ children }: AppProvidersProps) {
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
