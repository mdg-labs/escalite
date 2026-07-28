import type { ReactElement } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { useParams, useSearchParams } from 'react-router'
import {
  useCreateEscalationPolicyMutation,
  useEscalationPolicyQuery,
  useOrganizationUsersQuery,
  useSchedulesQuery,
  useServiceQuery,
  useUpdateEscalationPolicyMutation,
} from '@escalite/ts-types'
import {
  EscalationPolicyEditor,
  canSaveEscalationPolicy,
  type EscalationEditorPolicy,
  type EscalationPolicySavePayload,
} from '@escalite/ui/domain/EscalationPolicyEditor'

import { AppShell } from '../components/app-shell'
import { PageBreadcrumbs } from '../components/page-breadcrumbs'
import {
  buildEscalationEditorOptions,
  createDefaultEditorPolicy,
  mapApiPolicyToEditor,
} from '../lib/escalation-policy'
import { t } from '../lib/i18n'
import { notifyMutationSuccess, showMutationError } from '../lib/toast'

function formatGraphQLError(message: string): string {
  return message.replace(/^(\[GraphQL\]\s*)+/, '')
}

export function EscalationPolicyPage(): ReactElement {
  const { policyId, serviceId: routeServiceId } = useParams()
  const [searchParams] = useSearchParams()
  const serviceId = routeServiceId ?? searchParams.get('serviceId') ?? ''
  const isCreateMode = policyId === 'new'

  const [{ data, fetching, error }] = useEscalationPolicyQuery({
    pause: isCreateMode || !policyId,
    variables: { id: policyId ?? '' },
  })

  const [{ data: serviceData }] = useServiceQuery({
    pause: !serviceId,
    variables: { id: serviceId },
  })

  const teamId = serviceData?.service?.teamId ?? ''

  const [{ data: usersData }] = useOrganizationUsersQuery({
    requestPolicy: 'cache-and-network',
  })

  const [{ data: schedulesData }] = useSchedulesQuery({
    pause: !teamId,
    variables: { teamId },
  })

  const editorOptions = useMemo(
    () =>
      buildEscalationEditorOptions(
        usersData?.organizationUsers ?? [],
        schedulesData?.schedules ?? [],
      ),
    [schedulesData?.schedules, usersData?.organizationUsers],
  )

  const [, createEscalationPolicy] = useCreateEscalationPolicyMutation()
  const [, updateEscalationPolicy] = useUpdateEscalationPolicyMutation()

  const serviceName = serviceData?.service?.name ?? t('services.detail.title')
  const policyLabel = isCreateMode
    ? t('services.escalation.create')
    : (data?.escalationPolicy?.name ?? t('services.escalation.title'))
  const breadcrumbItems = useMemo(
    () => [
      { label: t('nav.services'), href: '/services' },
      { label: serviceName, href: `/services/${serviceId}` },
      { label: policyLabel },
    ],
    [policyLabel, serviceId, serviceName],
  )

  const initialPolicy = useMemo(() => {
    if (isCreateMode) {
      return createDefaultEditorPolicy()
    }

    if (!data?.escalationPolicy) {
      return null
    }

    return mapApiPolicyToEditor(data.escalationPolicy)
  }, [data?.escalationPolicy, isCreateMode])

  const [policy, setPolicy] = useState<EscalationEditorPolicy | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [savedMessage, setSavedMessage] = useState<string | null>(null)

  useEffect(() => {
    if (initialPolicy) {
      setPolicy(initialPolicy)
    }
  }, [initialPolicy])

  const editorPolicy = policy ?? initialPolicy

  async function handleSave(payload: EscalationPolicySavePayload): Promise<void> {
    if (!canSaveEscalationPolicy(editorPolicy?.steps ?? [])) {
      setSaveError(t('escalationPolicy.error.incompleteTargets'))
      return
    }

    setSaveError(null)
    setSavedMessage(null)
    setSaving(true)

    if (isCreateMode) {
      if (!serviceId.trim()) {
        setSaving(false)
        setSaveError(t('escalationPolicy.error.missingServiceId'))
        return
      }

      const result = await createEscalationPolicy({
        input: {
          serviceId: serviceId.trim(),
          name: payload.name,
          steps: payload.steps,
        },
      })

      setSaving(false)

      if (result.error) {
        showMutationError(result.error, 'escalationPolicy.error.action')
        return
      }

      if (result.data?.createEscalationPolicy) {
        setPolicy(mapApiPolicyToEditor(result.data.createEscalationPolicy))
      }

      setSavedMessage(t('escalationPolicy.success.created'))
      notifyMutationSuccess('escalationPolicy.toast.created')
      return
    }

    if (!policyId) {
      setSaving(false)
      setSaveError(t('escalationPolicy.error.missingPolicyId'))
      return
    }

    const result = await updateEscalationPolicy({
      input: {
        id: policyId,
        name: payload.name,
        steps: payload.steps,
      },
    })

    setSaving(false)

    if (result.error) {
      showMutationError(result.error, 'escalationPolicy.error.action')
      return
    }

    if (result.data?.updateEscalationPolicy) {
      setPolicy(mapApiPolicyToEditor(result.data.updateEscalationPolicy))
    }

    setSavedMessage(t('escalationPolicy.success.updated'))
    notifyMutationSuccess('escalationPolicy.toast.updated')
  }

  return (
    <AppShell title={policyLabel}>
      <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
          <PageBreadcrumbs items={breadcrumbItems} />
          <h1 className="mt-2 text-xl font-semibold text-foreground">
            {isCreateMode ? t('services.escalation.create') : policyLabel}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">{t('escalationPolicy.description')}</p>

          {fetching ? (
            <p className="mt-6 text-sm text-muted-foreground">{t('escalationPolicy.loading')}</p>
          ) : null}

          {error ? (
            <p className="mt-6 text-sm text-destructive-foreground" role="alert">
              {formatGraphQLError(error.message)}
            </p>
          ) : null}

          {!isCreateMode && !fetching && !data?.escalationPolicy ? (
            <p className="mt-6 text-sm text-destructive-foreground" role="alert">
              {t('escalationPolicy.notFound')}
            </p>
          ) : null}

          {editorPolicy ? (
            <div className="mt-6">
              {!isCreateMode && editorPolicy.steps.every((step) => step.targets.length === 0) ? (
                <p className="mb-4 rounded-lg border border-warning/30 bg-warning/10 p-3 text-sm text-warning-foreground">
                  {t('escalationPolicy.warning.noTargets')}
                </p>
              ) : null}

              <EscalationPolicyEditor
                onChange={setPolicy}
                onSave={handleSave}
                options={editorOptions}
                policy={editorPolicy}
                saving={saving}
              />

              {saveError ? (
                <p className="mt-4 text-sm text-destructive-foreground" role="alert">
                  {saveError}
                </p>
              ) : null}

              {savedMessage ? (
                <p className="mt-4 text-sm text-success-foreground" role="status">
                  {savedMessage}
                </p>
              ) : null}
            </div>
          ) : null}

          {isCreateMode && !serviceId ? (
            <p className="mt-4 text-sm text-destructive-foreground" role="alert">
              {t('escalationPolicy.error.serviceIdHint')}
            </p>
          ) : null}
      </section>
    </AppShell>
  )
}