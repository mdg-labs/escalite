import { useState, type ReactElement } from 'react'
import {
  useMeQuery,
  useMyOrganizationsQuery,
  useSwitchOrganizationMutation,
  type MyOrganizationsQuery,
} from '@escalite/ts-types'
import {
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
} from '@escalite/ui'
import { Building2Icon } from 'lucide-react'

import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

type OrganizationMembership = MyOrganizationsQuery['myOrganizations'][number]

function membershipRoleLabel(role: OrganizationMembership['role']): string {
  return role === 'ADMIN' ? t('org.switcher.role.admin') : t('org.switcher.role.member')
}

export function OrgSwitcher(): ReactElement | null {
  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const [{ data, fetching, error }] = useMyOrganizationsQuery({ requestPolicy: 'cache-first' })
  const [, switchOrganization] = useSwitchOrganizationMutation()
  const [switching, setSwitching] = useState(false)
  const [switchError, setSwitchError] = useState<string | null>(null)

  const memberships = data?.myOrganizations ?? []
  const activeOrganizationId = meData?.me?.organizationId ?? ''

  if (fetching && memberships.length === 0) {
    return (
      <span className="text-sm text-muted-foreground">{t('org.switcher.loading')}</span>
    )
  }

  if (memberships.length === 0) {
    return null
  }

  async function handleOrganizationChange(organizationId: string | null): Promise<void> {
    if (!organizationId || organizationId === activeOrganizationId || switching) {
      return
    }

    setSwitchError(null)
    setSwitching(true)

    const result = await switchOrganization({ organizationId })
    setSwitching(false)

    if (result.error) {
      showMutationError(result.error, 'org.switcher.error')
      return
    }

    notifyMutationSuccess('org.toast.switched')
  }

  const activeMembership =
    memberships.find((membership) => membership.organization.id === activeOrganizationId) ??
    memberships[0]

  return (
    <div className="flex flex-col items-end gap-1">
      <Select
        disabled={switching || memberships.length <= 1}
        onValueChange={(value) => void handleOrganizationChange(value)}
        value={activeMembership.organization.id}
      >
        <SelectTrigger
          aria-label={t('org.switcher.label')}
          className="w-auto min-w-44"
          size="sm"
        >
          <SelectValue placeholder={t('org.switcher.placeholder')}>
            <span className="flex items-center gap-2">
              <Building2Icon className="size-4 shrink-0 text-muted-foreground" />
              <span className="truncate">{activeMembership.organization.name}</span>
            </span>
          </SelectValue>
        </SelectTrigger>
        <SelectPopup>
          {memberships.map((membership) => (
            <SelectItem key={membership.organization.id} value={membership.organization.id}>
              <span className="flex min-w-0 flex-col">
                <span className="truncate font-medium">{membership.organization.name}</span>
                <span className="text-xs text-muted-foreground">
                  {membershipRoleLabel(membership.role)}
                </span>
              </span>
            </SelectItem>
          ))}
        </SelectPopup>
      </Select>
      {(switchError || error) && (
        <span className="max-w-56 truncate text-xs text-destructive">
          {switchError ?? (error ? formatGraphQLError(error.message) : null)}
        </span>
      )}
    </div>
  )
}
