import type { ReactNode } from 'react'

import { usePushNotifications } from '@/push/use-push-notifications'

type PushNotificationBootstrapProps = {
  children: ReactNode
}

export function PushNotificationBootstrap({ children }: PushNotificationBootstrapProps) {
  usePushNotifications()
  return children
}
