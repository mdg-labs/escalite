import { resolveApiPublicUrl, resolveStatusPollIntervalMs } from '@escalite/runtime-config'

const apiPublicUrl = resolveApiPublicUrl(import.meta.env.VITE_API_PUBLIC_URL)
const pollIntervalMs = resolveStatusPollIntervalMs(import.meta.env.VITE_STATUS_POLL_INTERVAL_MS)

export const statusPageConfig = {
  apiPublicUrl,
  pollIntervalMs,
} as const
