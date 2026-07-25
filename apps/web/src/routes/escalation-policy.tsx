import type { ReactElement } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router'
import {
  useCreateEscalationPolicyMutation,
  useEscalationPolicyQuery,
  useUpdateEscalationPolicyMutation,
} from '@escalite/ts-types'
import {
  EscalationPolicyEditor,
  canSaveEscalationPolicy,
  type EscalationEditorPolicy,
  type EscalationPolicySavePayload,
} from '@escalite/ui/domain/EscalationPolicyEditor'

import {
  createDefaultEditorPolicy,
  mapApiPolicyToEditor,
} from '../lib/escalation-policy'

function formatGraphQLError(message: string): string {
  return message.replace(/^(\[GraphQL\]\s*)+/, '')
}

export function EscalationPolicyPage(): ReactElement {
  const { policyId } = useParams()
  const [searchParams] = useSearchParams()
  const serviceId = searchParams.get('serviceId') ?? ''
  const isCreateMode = policyId === 'new'

  const [{ data, fetching, error }] = useEscalationPolicyQuery({
    pause: isCreateMode || !policyId,
    variables: { id: policyId ?? '' },
  })

  const [, createEscalationPolicy] = useCreateEscalationPolicyMutation()
  const [, updateEscalationPolicy] = useUpdateEscalationPolicyMutation()

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
      setSaveError('Each step must include at least one complete target before saving.')
      return
    }

    setSaveError(null)
    setSavedMessage(null)
    setSaving(true)

    if (isCreateMode) {
      if (!serviceId.trim()) {
        setSaving(false)
        setSaveError('A serviceId query parameter is required to create a policy.')
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
        setSaveError(formatGraphQLError(result.error.message))
        return
      }

      setSavedMessage('Escalation policy created. Step order was saved to the API.')
      return
    }

    if (!policyId) {
      setSaving(false)
      setSaveError('Policy ID is missing.')
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
      setSaveError(formatGraphQLError(result.error.message))
      return
    }

    if (result.data?.updateEscalationPolicy) {
      setPolicy(mapApiPolicyToEditor(result.data.updateEscalationPolicy))
    }

    setSavedMessage('Escalation policy updated. Step order was saved to the API.')
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border bg-card">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <Link className="text-sm font-semibold tracking-wide text-foreground" to="/dashboard">
              Escalite
            </Link>
            <span className="text-sm text-muted-foreground">Escalation policy</span>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-8">
        <section className="rounded-xl border border-border bg-card p-6 shadow-xs/5">
          <h1 className="text-xl font-semibold text-foreground">
            {isCreateMode ? 'Create escalation policy' : 'Edit escalation policy'}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Configure ordered steps, delays, and notification targets. Drag steps to reorder; save
            persists step order to the API.
          </p>

          {fetching ? (
            <p className="mt-6 text-sm text-muted-foreground">Loading policy…</p>
          ) : null}

          {error ? (
            <p className="mt-6 text-sm text-destructive-foreground" role="alert">
              {formatGraphQLError(error.message)}
            </p>
          ) : null}

          {!isCreateMode && !fetching && !data?.escalationPolicy ? (
            <p className="mt-6 text-sm text-destructive-foreground" role="alert">
              Escalation policy not found.
            </p>
          ) : null}

          {editorPolicy ? (
            <div className="mt-6">
              {!isCreateMode && editorPolicy.steps.every((step) => step.targets.length === 0) ? (
                <p className="mb-4 rounded-lg border border-warning/30 bg-warning/10 p-3 text-sm text-warning-foreground">
                  Loaded steps from the API do not include targets yet. Add at least one target per
                  step before saving.
                </p>
              ) : null}

              <EscalationPolicyEditor
                onChange={setPolicy}
                onSave={handleSave}
                options={{
                  users: [],
                  schedules: [],
                }}
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
              Add <code className="font-mono">?serviceId=&lt;uuid&gt;</code> to the URL to create a
              policy for a service.
            </p>
          ) : null}
        </section>
      </main>
    </div>
  )
}