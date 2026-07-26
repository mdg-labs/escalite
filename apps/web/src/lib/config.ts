import { resolveApiPublicUrl, resolveGraphqlUrl } from '@escalite/runtime-config'

const graphqlUrl = resolveGraphqlUrl(import.meta.env.VITE_GRAPHQL_URL)
const apiPublicUrl = resolveApiPublicUrl(import.meta.env.VITE_API_PUBLIC_URL)

export const appConfig = {
  graphqlUrl,
  apiPublicUrl,
} as const
