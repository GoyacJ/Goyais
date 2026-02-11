import { ref, watch } from 'vue'
import type { DensityMode } from './types'

const STORAGE_KEY = 'goyais.ui.density'

const densityMode = ref<DensityMode>('compact')

let initialized = false

function isDensityMode(value: string | null): value is DensityMode {
  return value === 'compact' || value === 'comfortable'
}

function readDensityMode(): DensityMode {
  const stored = localStorage.getItem(STORAGE_KEY)
  return isDensityMode(stored) ? stored : 'compact'
}

function applyDensity(mode: DensityMode): void {
  document.documentElement.setAttribute('data-density', mode)
}

export function initDensitySystem(): void {
  if (initialized) {
    return
  }

  initialized = true
  densityMode.value = readDensityMode()
  applyDensity(densityMode.value)

  watch(
    densityMode,
    (mode) => {
      localStorage.setItem(STORAGE_KEY, mode)
      applyDensity(mode)
    },
    { flush: 'post' },
  )
}

export function useDensityStore() {
  return {
    densityMode,
    setDensityMode: (mode: DensityMode) => {
      densityMode.value = mode
    },
  }
}

export function densityStorageKey(): string {
  return STORAGE_KEY
}
