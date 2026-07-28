import type { MessageKey } from '../lib/i18n'

const STATUS_PAGE_SLUG_PATTERN = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/

type StatusPageSlugErrorKey = Extract<
  MessageKey,
  'statusPages.error.requiredSlug' | 'statusPages.error.invalidSlug'
>

export function normalizeStatusPageSlug(slug: string): string {
  return slug.trim().toLowerCase()
}

export function validateStatusPageSlug(slug: string): StatusPageSlugErrorKey | null {
  const normalized = normalizeStatusPageSlug(slug)
  if (normalized === '') {
    return 'statusPages.error.requiredSlug'
  }
  if (!STATUS_PAGE_SLUG_PATTERN.test(normalized)) {
    return 'statusPages.error.invalidSlug'
  }
  return null
}

export function statusPagePublicAppUrl(slug: string): string {
  const configured = import.meta.env.VITE_STATUS_PAGE_PUBLIC_URL?.trim()
  const base =
    configured ||
    (import.meta.env.DEV ? 'http://localhost:5174' : window.location.origin)
  return `${base.replace(/\/$/, '')}/${encodeURIComponent(normalizeStatusPageSlug(slug))}`
}
