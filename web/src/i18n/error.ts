import type { ComposerTranslation } from 'vue-i18n'

export interface BackendErrorShape {
  code: string
  messageKey: string
  details?: Record<string, unknown>
}

export function translateBackendError(t: ComposerTranslation, error: BackendErrorShape): string {
  const key = error.messageKey || 'error.common.unknown'
  return t(key, error.details ?? {})
}
