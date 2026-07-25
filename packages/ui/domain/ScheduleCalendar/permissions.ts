import type { ViewerRole } from './types'

/** Override creation requires org admin or member with team access (doc 07). */
export function canCreateScheduleOverride(
  role: ViewerRole,
  hasTeamAccess: boolean,
): boolean {
  if (role === 'ADMIN') {
    return true
  }

  return role === 'MEMBER' && hasTeamAccess
}
