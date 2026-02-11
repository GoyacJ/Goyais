/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

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
