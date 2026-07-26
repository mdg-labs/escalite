import type { FormEvent, ReactElement } from 'react'
import { useState } from 'react'
import { Alert, AlertTitle, Button, Input } from '@escalite/ui'

import { t } from '../lib/i18n'
import { subscribeToStatusPage } from '../lib/status-page'

type SubscribeFormProps = {
  slug: string
  apiPublicUrl: string
}

export function SubscribeForm({ slug, apiPublicUrl }: SubscribeFormProps): ReactElement {
  const [email, setEmail] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    setSubmitting(true)
    setError(false)
    setSuccess(false)

    try {
      await subscribeToStatusPage(slug, email.trim(), apiPublicUrl)
      setSuccess(true)
      setEmail('')
    } catch {
      setError(true)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section aria-labelledby="subscribe-heading" className="space-y-3">
      <div>
        <h2 className="text-lg font-semibold text-foreground" id="subscribe-heading">
          {t('statusPage.subscribe.title')}
        </h2>
        <p className="text-sm text-muted-foreground">{t('statusPage.subscribe.description')}</p>
      </div>

      {success ? (
        <Alert variant="success">
          <AlertTitle>{t('statusPage.subscribe.success')}</AlertTitle>
        </Alert>
      ) : null}

      {error ? (
        <Alert variant="error">
          <AlertTitle>{t('statusPage.subscribe.error')}</AlertTitle>
        </Alert>
      ) : null}

      <form className="flex flex-col gap-3 sm:flex-row sm:items-end" onSubmit={handleSubmit}>
        <div className="flex-1 space-y-1.5">
          <label className="text-sm font-medium text-foreground" htmlFor="subscribe-email">
            {t('statusPage.subscribe.email')}
          </label>
          <Input
            autoComplete="email"
            id="subscribe-email"
            onChange={(event) => setEmail(event.target.value)}
            placeholder={t('statusPage.subscribe.emailPlaceholder')}
            required
            type="email"
            value={email}
          />
        </div>
        <Button disabled={submitting || email.trim() === ''} type="submit">
          {submitting ? t('statusPage.subscribe.submitting') : t('statusPage.subscribe.submit')}
        </Button>
      </form>
    </section>
  )
}
