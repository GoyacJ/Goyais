/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { watch } from 'vue'
import { createI18n } from 'vue-i18n'
import enUS from '@/locales/en-US'
import zhCN from '@/locales/zh-CN'
import type { SupportedLocale } from '@/design-system/types'

const STORAGE_KEY = 'goyais.ui.locale'
const LEGACY_STORAGE_KEY = 'goyais.locale'

const SUPPORTED_LOCALES: SupportedLocale[] = ['zh-CN', 'en-US']

function isSupportedLocale(value: string | null): value is SupportedLocale {
  return value === 'zh-CN' || value === 'en-US'
}

function migrateLegacyLocale(): void {
  const legacy = localStorage.getItem(LEGACY_STORAGE_KEY)

  if (isSupportedLocale(legacy) && !localStorage.getItem(STORAGE_KEY)) {
    localStorage.setItem(STORAGE_KEY, legacy)
  }
}

function readLocale(): SupportedLocale {
  migrateLegacyLocale()
  const stored = localStorage.getItem(STORAGE_KEY)

  return isSupportedLocale(stored) ? stored : 'zh-CN'
}

const isDev = import.meta.env.DEV

const i18n = createI18n({
  legacy: false,
  locale: readLocale(),
  fallbackLocale: 'en-US',
  missingWarn: isDev,
  fallbackWarn: isDev,
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
  missing: (...args: unknown[]) => {
    const locale = String(args[0] ?? 'unknown')
    const key = String(args[1] ?? '')

    if (isDev) {
      console.warn(`[i18n missing] locale=${locale} key=${key}`)
    }

    return key
  },
})

watch(
  () => i18n.global.locale.value,
  (locale) => {
    if (isSupportedLocale(locale)) {
      localStorage.setItem(STORAGE_KEY, locale)
    }
  },
  { flush: 'post' },
)

export function localeStorageKey(): string {
  return STORAGE_KEY
}

export function supportedLocales(): SupportedLocale[] {
  return [...SUPPORTED_LOCALES]
}

export default i18n
