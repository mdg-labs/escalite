import { cacheExchange, createClient, fetchExchange } from 'urql'

import { appConfig } from './config'

export const urqlClient = createClient({
  url: appConfig.graphqlUrl,
  exchanges: [cacheExchange, fetchExchange],
  fetchOptions: {
    credentials: 'include',
  },
})
