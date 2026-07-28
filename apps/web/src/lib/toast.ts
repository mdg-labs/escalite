import { toastManager } from '@escalite/ui'

import { formatGraphQLError } from './format'
import { t, type MessageKey } from './i18n'

export { toastManager } from '@escalite/ui'

export type ToastSemanticType = 'success' | 'error' | 'info' | 'warning'

export function showToast(options: {
  type: ToastSemanticType
  title: string
  description?: string
}): void {
  toastManager.add({
    type: options.type,
    title: options.title,
    ...(options.description ? { description: options.description } : {}),
  })
}

/** p-toast-2 success variant */
export function showSuccessToast(title: string, description?: string): void {
  showToast({ type: 'success', title, description })
}

/** p-toast-2 error variant */
export function showErrorToast(title: string, description?: string): void {
  showToast({ type: 'error', title, description })
}

export function showMutationError(
  error: { message: string } | string,
  titleKey: MessageKey = 'toast.error.action',
): void {
  const message = typeof error === 'string' ? error : error.message
  showErrorToast(t(titleKey), formatGraphQLError(message))
}

export function notifyMutationSuccess(titleKey: MessageKey, description?: string): void {
  showSuccessToast(t(titleKey), description)
}
