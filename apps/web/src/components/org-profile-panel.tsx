import { useEffect, useState, type FormEvent, type ReactElement } from 'react'
import {
  UserRole,
  useMeQuery,
  useMyOrganizationsQuery,
  useUpdateOrganizationMutation,
} from '@escalite/ts-types'
import { Alert, AlertDescription, Button, Input } from '@escalite/ui'
import { Building2Icon } from 'lucide-react'

import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'

export function OrgProfilePanel(): ReactElement {
  const [{ data: meData }] = useMeQuery({ requestPolicy: 'cache-first' })
  const [{ data: orgData, fetching, error }, reexecuteQuery] = useMyOrganizationsQuery({
    requestPolicy: 'cache-first',
  })
  const [, updateOrganization] = useUpdateOrganizationMutation()

  const isAdmin = meData?.me?.role === UserRole.Admin
  const activeOrganizationId = meData?.me?.organizationId ?? ''
  const activeOrganization = orgData?.myOrganizations.find(
    (membership) => membership.organization.id === activeOrganizationId,
  )?.organization

  const [name, setName] = useState('')
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (activeOrganization?.name) {
      setName(activeOrganization.name)
    }
  }, [activeOrganization?.name])

  const trimmedName = name.trim()
  const isDirty = activeOrganization ? trimmedName !== activeOrganization.name : false

  async function handleSave(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()

    if (!trimmedName) {
      setActionError(t('settings.orgProfile.error.requiredName'))
      return
    }

    setActionError(null)
    setSaving(true)

    const result = await updateOrganization({ input: { name: trimmedName } })
    setSaving(false)

    if (result.error) {
      setActionError(formatGraphQLError(result.error.message))
      return
    }

    reexecuteQuery({ requestPolicy: 'network-only' })
  }

  return (
    <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
      <div className="flex items-start gap-3">
        <Building2Icon className="mt-0.5 size-5 text-muted-foreground" />
        <div>
          <h2 className="text-xl font-semibold text-foreground">{t('settings.orgProfile.title')}</h2>
          <p className="mt-2 text-sm text-muted-foreground">{t('settings.orgProfile.description')}</p>
        </div>
      </div>

      {(error || actionError) && (
        <Alert className="mt-6" variant="error">
          <AlertDescription>
            {actionError ?? (error ? formatGraphQLError(error.message) : null)}
          </AlertDescription>
        </Alert>
      )}

      <div className="mt-6">
        {fetching && !activeOrganization ? (
          <p className="text-sm text-muted-foreground">{t('settings.orgProfile.loading')}</p>
        ) : isAdmin ? (
          <form className="space-y-4" onSubmit={(event) => void handleSave(event)}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground" htmlFor="org-name">
                {t('settings.orgProfile.name.label')}
              </label>
              <Input
                id="org-name"
                onChange={(event) => setName(event.target.value)}
                placeholder={t('settings.orgProfile.name.placeholder')}
                value={name}
              />
            </div>
            <Button disabled={saving || !isDirty || !trimmedName} type="submit">
              {saving ? t('settings.orgProfile.action.saving') : t('settings.orgProfile.action.save')}
            </Button>
          </form>
        ) : (
          <div className="space-y-2">
            <p className="text-sm font-medium text-foreground">{t('settings.orgProfile.name.label')}</p>
            <p className="text-sm text-foreground">{activeOrganization?.name ?? '—'}</p>
            <p className="text-xs text-muted-foreground">{t('settings.orgProfile.readOnly')}</p>
          </div>
        )}
      </div>
    </section>
  )
}
