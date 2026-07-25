import { useState, type ReactElement } from 'react'
import { CheckIcon, CopyIcon } from 'lucide-react'

import { Button } from '@escalite/ui'

type CopyButtonProps = {
  value: string
  label?: string
}

export function CopyButton({ value, label = 'Copy' }: CopyButtonProps): ReactElement {
  const [copied, setCopied] = useState(false)

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
          Copied
        </>
      ) : (
        <>
          <CopyIcon />
          {label}
        </>
      )}
    </Button>
  )
}
