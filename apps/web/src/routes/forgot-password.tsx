import { useState, type FormEvent, type ReactElement } from 'react'
import { Link } from 'react-router'
import { useRequestPasswordResetMutation } from '@escalite/ts-types'
import { Button, Input } from '@escalite/ui'

import { AuthLayout } from '../components/auth-layout'
import { t } from '../lib/i18n'

function formatGraphQLError(message: string): string {
  return message.replace(/^(\[GraphQL\]\s*)+/, '')
}

export function ForgotPasswordPage(): ReactElement {
  const [, requestPasswordReset] = useRequestPasswordResetMutation()
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [submitted, setSubmitted] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    setError(null)
    setLoading(true)

    const result = await requestPasswordReset({
      input: {
        email: email.trim(),
      },
    })

    setLoading(false)

    if (result.error) {
      setError(formatGraphQLError(result.error.message))
      return
    }

    setSubmitted(true)
  }

  return (
    <AuthLayout
      title={t('auth.forgotPassword.title')}
      description={t('auth.forgotPassword.description')}
    >
      {submitted ? (
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">{t('auth.forgotPassword.success')}</p>
          <Button className="w-full" render={<Link to="/login" />} type="button" variant="outline">
            {t('auth.forgotPassword.backToLogin')}
          </Button>
        </div>
      ) : (
        <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="email">
              {t('auth.forgotPassword.emailLabel')}
            </label>
            <Input
              autoComplete="email"
              id="email"
              name="email"
              onChange={(event) => setEmail(event.target.value)}
              required
              type="email"
              value={email}
            />
          </div>
          {error ? (
            <p className="text-sm text-destructive-foreground" role="alert">
              {error}
            </p>
          ) : null}
          <Button className="w-full" loading={loading} type="submit">
            {t('auth.forgotPassword.submit')}
          </Button>
        </form>
      )}
      {!submitted ? (
        <p className="mt-4 text-center text-sm text-muted-foreground">
          <Link className="font-medium text-foreground underline-offset-4 hover:underline" to="/login">
            {t('auth.forgotPassword.backToLogin')}
          </Link>
        </p>
      ) : null}
    </AuthLayout>
  )
}
