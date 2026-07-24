import { useState, type FormEvent, type ReactElement } from 'react'
import { Link, useNavigate } from 'react-router'
import { useLoginMutation } from '@escalite/ts-types'
import { Button, Input } from '@escalite/ui'

import { AuthLayout } from '../components/auth-layout'

function formatGraphQLError(message: string): string {
  return message.replace(/^(\[GraphQL\]\s*)+/, '')
}

export function LoginPage(): ReactElement {
  const navigate = useNavigate()
  const [, login] = useLoginMutation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    setError(null)
    setLoading(true)

    const result = await login({
      input: {
        email: email.trim(),
        password,
      },
    })

    setLoading(false)

    if (result.error) {
      setError(formatGraphQLError(result.error.message))
      return
    }

    navigate('/dashboard', { replace: true })
  }

  return (
    <AuthLayout
      title="Sign in"
      description="Use your organization email and password to access Escalite."
    >
      <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground" htmlFor="email">
            Email
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
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground" htmlFor="password">
            Password
          </label>
          <Input
            autoComplete="current-password"
            id="password"
            name="password"
            onChange={(event) => setPassword(event.target.value)}
            required
            type="password"
            value={password}
          />
        </div>
        {error ? (
          <p className="text-sm text-destructive-foreground" role="alert">
            {error}
          </p>
        ) : null}
        <Button className="w-full" loading={loading} type="submit">
          Sign in
        </Button>
      </form>
      <p className="mt-4 text-center text-sm text-muted-foreground">
        First install?{' '}
        <Link className="font-medium text-foreground underline-offset-4 hover:underline" to="/setup">
          Create the first admin account
        </Link>
      </p>
    </AuthLayout>
  )
}
