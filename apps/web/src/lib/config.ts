const graphqlUrl = import.meta.env.VITE_GRAPHQL_URL ?? '/graphql'

export const appConfig = {
  graphqlUrl,
} as const
