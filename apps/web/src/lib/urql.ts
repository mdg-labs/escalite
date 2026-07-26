import {
  createClient,
  fetchExchange,
  subscriptionExchange,
} from 'urql'
import { cacheExchange, type Cache } from '@urql/exchange-graphcache'
import { createClient as createWSClient } from 'graphql-ws'

import { appConfig } from './config'

function graphqlWebSocketUrl(httpUrl: string): string {
  if (httpUrl.startsWith('https://')) {
    return `wss://${httpUrl.slice('https://'.length)}`
  }
  if (httpUrl.startsWith('http://')) {
    return `ws://${httpUrl.slice('http://'.length)}`
  }
  const origin = window.location.origin
  if (origin.startsWith('https://')) {
    return `wss://${origin.slice('https://'.length)}${httpUrl}`
  }
  return `ws://${origin.slice('http://'.length)}${httpUrl}`
}

const wsClient = createWSClient({
  url: graphqlWebSocketUrl(appConfig.graphqlUrl),
})

function invalidateQueryCache(cache: Cache): void {
  cache.inspectFields('Query').forEach((field) => {
    cache.invalidate('Query', field.fieldName, field.arguments ?? undefined)
  })
}

const graphCache = cacheExchange({
  updates: {
    Mutation: {
      switchOrganization(_result, _args, cache) {
        invalidateQueryCache(cache)
      },
    },
  },
})

export const urqlClient = createClient({
  url: appConfig.graphqlUrl,
  exchanges: [
    graphCache,
    fetchExchange,
    subscriptionExchange({
      forwardSubscription(request) {
        const input = { ...request, query: request.query ?? '' }
        return {
          subscribe(sink) {
            const unsubscribe = wsClient.subscribe(input, sink)
            return { unsubscribe }
          },
        }
      },
    }),
  ],
  fetchOptions: {
    credentials: 'include',
  },
})
