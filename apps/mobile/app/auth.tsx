import { useLocalSearchParams, useRouter } from 'expo-router'
import { useEffect, useRef } from 'react'
import { Paragraph, Spinner, YStack } from 'tamagui'

import { useAuth } from '@/auth/context'

function resolveAuthCode(code: string | string[] | undefined): string | null {
  if (typeof code === 'string' && code.length > 0) {
    return code
  }
  if (Array.isArray(code) && typeof code[0] === 'string' && code[0].length > 0) {
    return code[0]
  }
  return null
}

export default function AuthCallbackScreen() {
  const { code } = useLocalSearchParams<{ code?: string | string[] }>()
  const { signInWithCode } = useAuth()
  const router = useRouter()
  const handledRef = useRef(false)

  useEffect(() => {
    if (handledRef.current) {
      return
    }

    const authCode = resolveAuthCode(code)
    if (!authCode) {
      router.replace('/')
      return
    }

    handledRef.current = true
    void signInWithCode(authCode)
      .then(() => {
        router.replace('/')
      })
      .catch(() => {
        router.replace('/')
      })
  }, [code, router, signInWithCode])

  return (
    <YStack flex={1} alignItems="center" justifyContent="center" backgroundColor="$background" padding="$4" gap="$3">
      <Spinner size="large" color="$color" />
      <Paragraph color="$color" textAlign="center">
        Completing sign in…
      </Paragraph>
    </YStack>
  )
}
