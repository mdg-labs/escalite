import { useState, type ReactElement } from 'react'
import { CheckIcon, CopyIcon } from 'lucide-react'

import { Button } from '@escalite/ui'
import { t } from '../lib/i18n'

type CopyButtonProps = {
  value: string
  label?: string
}

export function CopyButton({ value, label }: CopyButtonProps): ReactElement {
  const [copied, setCopied] = useState(false)
  const buttonLabel = label ?? t('common.copy')

  async function handleCopy(): Promise<void> {
    await navigator.clipboard.writeText(value)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Button onClick={() => void handleCopy()} size="sm" type="button" variant="outline">
      {copied ? (
        <>
          <CheckIcon />
          {t('common.copied')}
        </>
      ) : (
        <>
          <CopyIcon />
          {buttonLabel}
        </>
      )}
    </Button>
  )
}
