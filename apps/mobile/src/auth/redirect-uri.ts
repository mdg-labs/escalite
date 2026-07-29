import * as Linking from 'expo-linking'
import type { ParsedURL } from 'expo-linking'

import { mobileAuthConstants } from '@/auth/config'

function normalizeAuthPath(path: string | null | undefined): string {
  if (!path) {
    return ''
  }
  return path.replace(/^\/+|\/+$/g, '')
}

function pathEndsWithAuthSegment(path: string): boolean {
  return path === mobileAuthConstants.deepLinkAuthPath || path.endsWith(`/${mobileAuthConstants.deepLinkAuthPath}`)
}

export function mobileAuthRedirectUri(): string {
  return Linking.createURL(mobileAuthConstants.deepLinkAuthPath)
}

export function isParsedAuthCallback(parsed: ParsedURL): boolean {
  const authSegment = mobileAuthConstants.deepLinkAuthPath

  if (parsed.scheme === mobileAuthConstants.deepLinkScheme) {
    if (parsed.hostname === authSegment) {
      return true
    }
    const normalizedPath = normalizeAuthPath(parsed.path)
    return pathEndsWithAuthSegment(normalizedPath)
  }

  if (parsed.scheme === 'exp') {
    const normalizedPath = normalizeAuthPath(parsed.path)
    return pathEndsWithAuthSegment(normalizedPath) || normalizedPath.endsWith(`/--/${authSegment}`)
  }

  if (parsed.scheme?.startsWith('exp+')) {
    if (parsed.hostname === authSegment) {
      return true
    }
    const normalizedPath = normalizeAuthPath(parsed.path)
    return pathEndsWithAuthSegment(normalizedPath)
  }

  return false
}
