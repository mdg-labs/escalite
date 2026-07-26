const apiPublicUrl = import.meta.env.VITE_API_PUBLIC_URL ?? window.location.origin
const pollIntervalMs = Number.parseInt(
  import.meta.env.VITE_STATUS_POLL_INTERVAL_MS ?? '60000',
  10,
)

export const statusPageConfig = {
  apiPublicUrl,
  pollIntervalMs: Number.isFinite(pollIntervalMs) && pollIntervalMs > 0 ? pollIntervalMs : 60_000,
} as const
