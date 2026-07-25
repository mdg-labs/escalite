const graphqlUrl = import.meta.env.VITE_GRAPHQL_URL ?? '/graphql'
const apiPublicUrl = import.meta.env.VITE_API_PUBLIC_URL ?? window.location.origin

export const appConfig = {
  graphqlUrl,
  apiPublicUrl,
} as const
