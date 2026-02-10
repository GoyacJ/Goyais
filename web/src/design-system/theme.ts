import { ref, watch } from 'vue'
import type { ThemeMode } from './types'

const STORAGE_KEY = 'goyais.ui.theme'
const LEGACY_STORAGE_KEY = 'goyais.theme'

const themeMode = ref<ThemeMode>('system')

let initialized = false
let mediaQuery: MediaQueryList | null = null

function isThemeMode(value: string | null): value is ThemeMode {
  return value === 'system' || value === 'light' || value === 'dark'
}

function resolveTheme(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }

  return mode
}

function applyResolvedTheme(mode: ThemeMode): void {
  const resolved = resolveTheme(mode)
  document.documentElement.classList.toggle('dark', resolved === 'dark')
  document.documentElement.setAttribute('data-theme-mode', mode)
}

function migrateLegacyTheme(): void {
  const legacy = localStorage.getItem(LEGACY_STORAGE_KEY)

  if (isThemeMode(legacy) && !localStorage.getItem(STORAGE_KEY)) {
    localStorage.setItem(STORAGE_KEY, legacy)
  }
}

function readThemeMode(): ThemeMode {
  migrateLegacyTheme()
  const stored = localStorage.getItem(STORAGE_KEY)
  return isThemeMode(stored) ? stored : 'system'
}

function handleSystemThemeChange(): void {
  if (themeMode.value === 'system') {
    applyResolvedTheme('system')
  }
}

function bindSystemListener(): void {
  if (!mediaQuery) {
    mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  }

  mediaQuery.addEventListener('change', handleSystemThemeChange)
}

function unbindSystemListener(): void {
  mediaQuery?.removeEventListener('change', handleSystemThemeChange)
}

function syncSystemListener(mode: ThemeMode): void {
  if (mode === 'system') {
    bindSystemListener()
    return
  }

  unbindSystemListener()
}

export function initThemeSystem(): void {
  if (initialized) {
    return
  }

  initialized = true
  themeMode.value = readThemeMode()
  applyResolvedTheme(themeMode.value)
  syncSystemListener(themeMode.value)

  watch(
    themeMode,
    (mode) => {
      localStorage.setItem(STORAGE_KEY, mode)
      applyResolvedTheme(mode)
      syncSystemListener(mode)
    },
    { flush: 'post' },
  )
}

export function useThemeStore() {
  return {
    themeMode,
    setThemeMode: (mode: ThemeMode) => {
      themeMode.value = mode
    },
  }
}

export function themeStorageKey(): string {
  return STORAGE_KEY
}
