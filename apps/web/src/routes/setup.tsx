import { useState, type FormEvent, type ReactElement } from 'react'
import { Link, useNavigate } from 'react-router'
import { useSetupMutation } from '@escalite/ts-types'
import { Button, Input } from '@escalite/ui'

import { AuthLayout } from '../components/auth-layout'

function formatGraphQLError(message: string): string {
  return message.replace(/^(\[GraphQL\]\s*)+/, '')
}

export function SetupPage(): ReactElement {
  const navigate = useNavigate()
  const [, setup] = useSetupMutation()
  const [organizationName, setOrganizationName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    setError(null)
    setLoading(true)

    const result = await setup({
      input: {
        organizationName: organizationName.trim(),
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
      title="Set up Escalite"
      description="Create your organization and the first admin account."
    >
      <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground" htmlFor="organizationName">
            Organization name
          </label>
          <Input
            autoComplete="organization"
            id="organizationName"
            name="organizationName"
            onChange={(event) => setOrganizationName(event.target.value)}
            required
            value={organizationName}
          />
        </div>
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground" htmlFor="email">
            Admin email
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
        {error ? (
          <p className="text-sm text-destructive-foreground" role="alert">
            {error}
          </p>
        ) : null}
        <Button className="w-full" loading={loading} type="submit">
          Create organization
        </Button>
      </form>
      <p className="mt-4 text-center text-sm text-muted-foreground">
        Already set up?{' '}
        <Link className="font-medium text-foreground underline-offset-4 hover:underline" to="/login">
          Sign in
        </Link>
      </p>
    </AuthLayout>
  )
}
