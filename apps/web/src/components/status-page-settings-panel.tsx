import { useEffect, useMemo, useState, type FormEvent, type ReactElement } from 'react'
import {
  useSaveStatusPageMutation,
  useStatusPageQuery,
} from '@escalite/ts-types'
import {
  Alert,
  AlertDescription,
  Badge,
  Button,
  Input,
  Textarea,
} from '@escalite/ui'
import { CircleCheckIcon, ExternalLinkIcon, GlobeIcon } from 'lucide-react'

import { formatGraphQLError } from '../lib/format'
import { t } from '../lib/i18n'
import {
  normalizeStatusPageSlug,
  statusPagePublicAppUrl,
  validateStatusPageSlug,
} from '../lib/status-page'

export function StatusPageSettingsPanel(): ReactElement {
  const [{ data, fetching, error }, reexecuteQuery] = useStatusPageQuery({
    pause: false,
    requestPolicy: 'network-only',
  })
  const [, saveStatusPage] = useSaveStatusPageMutation()

  const statusPage = data?.statusPage

  const [slug, setSlug] = useState('')
  const [title, setTitle] = useState('')
  const [enabled, setEnabled] = useState(false)
  const [frameAncestorsCsp, setFrameAncestorsCsp] = useState('')
  const [formError, setFormError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (statusPage) {
      setSlug(statusPage.slug)
      setTitle(statusPage.title)
      setEnabled(statusPage.enabled)
      setFrameAncestorsCsp(statusPage.frameAncestorsCsp ?? '')
    }
  }, [statusPage])

  const slugErrorKey = useMemo(() => validateStatusPageSlug(slug), [slug])
  const previewUrl = useMemo(() => {
    if (slugErrorKey) {
      return null
    }
    return statusPagePublicAppUrl(slug)
  }, [slug, slugErrorKey])

  async function handleSave(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    setFormError(null)
    setActionError(null)

    const normalizedSlug = normalizeStatusPageSlug(slug)
    const slugValidationError = validateStatusPageSlug(normalizedSlug)
    if (slugValidationError) {
      setFormError(t(slugValidationError))
      return
    }

    const trimmedTitle = title.trim()
    if (trimmedTitle === '') {
      setFormError(t('statusPages.error.requiredTitle'))
      return
    }

    setSaving(true)
    const result = await saveStatusPage({
      input: {
        slug: normalizedSlug,
        title: trimmedTitle,
        enabled,
        frameAncestorsCsp: frameAncestorsCsp.trim() || null,
      },
    })
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
        <GlobeIcon className="mt-0.5 size-5 text-muted-foreground" />
        <div>
          <h2 className="text-xl font-semibold text-foreground">{t('statusPages.settings.title')}</h2>
          <p className="mt-2 text-sm text-muted-foreground">{t('statusPages.settings.description')}</p>
        </div>
      </div>

      {(error || actionError) && (
        <Alert className="mt-6" variant="error">
          <AlertDescription>
            {actionError ?? (error ? formatGraphQLError(error.message) : null)}
          </AlertDescription>
        </Alert>
      )}

      {formError ? (
        <Alert className="mt-6" variant="error">
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      <div className="mt-6 space-y-6">
        {fetching && !statusPage ? (
          <p className="text-sm text-muted-foreground">{t('statusPages.settings.loading')}</p>
        ) : statusPage ? (
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={statusPage.enabled ? 'success' : 'secondary'}>
              <CircleCheckIcon />
              {statusPage.enabled
                ? t('statusPages.settings.status.enabled')
                : t('statusPages.settings.status.disabled')}
            </Badge>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">{t('statusPages.settings.status.notConfigured')}</p>
        )}

        {previewUrl ? (
          <div className="rounded-lg border border-border bg-muted/30 p-4">
            <p className="text-sm font-medium text-foreground">{t('statusPages.settings.preview.title')}</p>
            <p className="mt-1 text-sm text-muted-foreground">{t('statusPages.settings.preview.description')}</p>
            <a
              className="mt-3 inline-flex items-center gap-1.5 text-sm font-medium text-primary hover:underline"
              href={previewUrl}
              rel="noreferrer"
              target="_blank"
            >
              {previewUrl}
              <ExternalLinkIcon className="size-3.5" />
            </a>
          </div>
        ) : null}

        <form className="space-y-4" onSubmit={(event) => void handleSave(event)}>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="status-page-slug">
              {t('statusPages.settings.field.slug')}
            </label>
            <Input
              autoComplete="off"
              id="status-page-slug"
              onChange={(event) => setSlug(event.target.value)}
              placeholder={t('statusPages.settings.field.slugPlaceholder')}
              value={slug}
            />
            <p className="text-xs text-muted-foreground">{t('statusPages.settings.field.slugHelp')}</p>
            {slugErrorKey && slug.trim() !== '' ? (
              <p className="text-xs text-destructive">{t(slugErrorKey)}</p>
            ) : null}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="status-page-title">
              {t('statusPages.settings.field.title')}
            </label>
            <Input
              id="status-page-title"
              onChange={(event) => setTitle(event.target.value)}
              placeholder={t('statusPages.settings.field.titlePlaceholder')}
              value={title}
            />
          </div>

          <label className="flex items-center gap-2 text-sm text-foreground">
            <input
              checked={enabled}
              className="size-4 rounded border border-input"
              onChange={(event) => setEnabled(event.target.checked)}
              type="checkbox"
            />
            {t('statusPages.settings.field.enabled')}
          </label>

          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="status-page-csp">
              {t('statusPages.settings.field.frameAncestorsCsp')}
            </label>
            <Textarea
              id="status-page-csp"
              onChange={(event) => setFrameAncestorsCsp(event.target.value)}
              placeholder={t('statusPages.settings.field.frameAncestorsCspPlaceholder')}
              rows={3}
              value={frameAncestorsCsp}
            />
            <p className="text-xs text-muted-foreground">
              {t('statusPages.settings.field.frameAncestorsCspHelp')}
            </p>
          </div>

          <Button disabled={saving} type="submit">
            {saving ? t('statusPages.settings.action.saving') : t('statusPages.settings.action.save')}
          </Button>
        </form>
      </div>
    </section>
  )
}
