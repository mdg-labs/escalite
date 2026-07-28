import type { ReactElement } from 'react'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router'
import { Alert, AlertDescription, AlertTitle, Button } from '@escalite/ui'

import { statusPageConfig } from '../lib/config'
import { t } from '../lib/i18n'
import {
  confirmStatusPageUnsubscribe,
  StatusPageUnsubscribeError,
  StatusPageUnsubscribeInvalidTokenError,
} from '../lib/status-page'

type UnsubscribeState = 'idle' | 'loading' | 'success' | 'invalid' | 'error'

export function UnsubscribeRoute(): ReactElement {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')?.trim() ?? ''
  const [state, setState] = useState<UnsubscribeState>('idle')

  useEffect(() => {
    document.title = t('statusPage.unsubscribe.documentTitle')
  }, [])

  async function handleConfirm(): Promise<void> {
    if (!token) {
      setState('invalid')
      return
    }

    setState('loading')

    try {
      await confirmStatusPageUnsubscribe(token, statusPageConfig.apiPublicUrl)
      setState('success')
    } catch (error) {
      if (error instanceof StatusPageUnsubscribeInvalidTokenError) {
        setState('invalid')
        return
      }
      if (error instanceof StatusPageUnsubscribeError) {
        setState('error')
        return
      }
      setState('error')
    }
  }

  if (!token) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-6">
        <div className="w-full max-w-lg space-y-4">
          <Alert variant="error">
            <AlertTitle>{t('statusPage.unsubscribe.invalidTitle')}</AlertTitle>
            <AlertDescription>{t('statusPage.unsubscribe.invalidDescription')}</AlertDescription>
          </Alert>
        </div>
      </div>
    )
  }

  if (state === 'success') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-6">
        <div className="w-full max-w-lg space-y-4">
          <Alert variant="success">
            <AlertTitle>{t('statusPage.unsubscribe.successTitle')}</AlertTitle>
            <AlertDescription>{t('statusPage.unsubscribe.successDescription')}</AlertDescription>
          </Alert>
        </div>
      </div>
    )
  }

  if (state === 'invalid') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-6">
        <div className="w-full max-w-lg space-y-4">
          <Alert variant="error">
            <AlertTitle>{t('statusPage.unsubscribe.invalidTitle')}</AlertTitle>
            <AlertDescription>{t('statusPage.unsubscribe.invalidDescription')}</AlertDescription>
          </Alert>
        </div>
      </div>
    )
  }

  if (state === 'error') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-6">
        <div className="w-full max-w-lg space-y-4">
          <Alert variant="error">
            <AlertTitle>{t('statusPage.unsubscribe.errorTitle')}</AlertTitle>
            <AlertDescription>{t('statusPage.unsubscribe.errorDescription')}</AlertDescription>
          </Alert>
          <Button onClick={() => void handleConfirm()}>
            {t('statusPage.unsubscribe.retry')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <div className="w-full max-w-lg space-y-4 text-center">
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">
            {t('statusPage.unsubscribe.title')}
          </h1>
          <p className="text-sm text-muted-foreground">{t('statusPage.unsubscribe.description')}</p>
        </div>
        <Button disabled={state === 'loading'} onClick={() => void handleConfirm()}>
          {state === 'loading'
            ? t('statusPage.unsubscribe.confirming')
            : t('statusPage.unsubscribe.confirm')}
        </Button>
      </div>
    </div>
  )
}
