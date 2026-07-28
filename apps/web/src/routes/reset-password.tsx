import { useState, type FormEvent, type ReactElement } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'
import { useResetPasswordMutation } from '@escalite/ts-types'
import { Button, Input } from '@escalite/ui'

import { AuthLayout } from '../components/auth-layout'
import { t } from '../lib/i18n'

function formatGraphQLError(message: string): string {
  return message.replace(/^(\[GraphQL\]\s*)+/, '')
}

export function ResetPasswordPage(): ReactElement {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')?.trim() ?? ''
  const [, resetPassword] = useResetPasswordMutation()
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    setError(null)

    if (password !== confirmPassword) {
      setError(t('auth.resetPassword.error.mismatch'))
      return
    }

    setLoading(true)

    const result = await resetPassword({
      input: {
        token,
        password,
      },
    })

    setLoading(false)

    if (result.error) {
      setError(formatGraphQLError(result.error.message))
      return
    }

    navigate('/login', { replace: true })
  }

  if (!token) {
    return (
      <AuthLayout
        title={t('auth.resetPassword.title')}
        description={t('auth.resetPassword.missingToken')}
      >
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">{t('auth.resetPassword.missingTokenHelp')}</p>
          <Button className="w-full" render={<Link to="/forgot-password" />} type="button">
            {t('auth.resetPassword.requestNewLink')}
          </Button>
          <p className="text-center text-sm text-muted-foreground">
            <Link className="font-medium text-foreground underline-offset-4 hover:underline" to="/login">
              {t('auth.resetPassword.backToLogin')}
            </Link>
          </p>
        </div>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      title={t('auth.resetPassword.title')}
      description={t('auth.resetPassword.description')}
    >
      <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground" htmlFor="password">
            {t('auth.resetPassword.passwordLabel')}
          </label>
          <Input
            autoComplete="new-password"
            id="password"
            minLength={8}
            name="password"
            onChange={(event) => setPassword(event.target.value)}
            required
            type="password"
            value={password}
          />
        </div>
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground" htmlFor="confirmPassword">
            {t('auth.resetPassword.confirmPasswordLabel')}
          </label>
          <Input
            autoComplete="new-password"
            id="confirmPassword"
            minLength={8}
            name="confirmPassword"
            onChange={(event) => setConfirmPassword(event.target.value)}
            required
            type="password"
            value={confirmPassword}
          />
        </div>
        {error ? (
          <p className="text-sm text-destructive-foreground" role="alert">
            {error}
          </p>
        ) : null}
        <Button className="w-full" loading={loading} type="submit">
          {t('auth.resetPassword.submit')}
        </Button>
      </form>
      <p className="mt-4 text-center text-sm text-muted-foreground">
        <Link className="font-medium text-foreground underline-offset-4 hover:underline" to="/login">
          {t('auth.resetPassword.backToLogin')}
        </Link>
      </p>
    </AuthLayout>
  )
}
