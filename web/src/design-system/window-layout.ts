import { computed, ref, watch, type MaybeRef } from 'vue'
import type { LayoutMode, WindowRect, WindowState } from '@/design-system/types'
import type { WindowPaneManifest } from '@/design-system/window-manifests'

const STORAGE_PREFIX = 'goyais.ui.windows'

function resolveMaybeRef<T>(value: MaybeRef<T>): T {
  return typeof value === 'object' && value !== null && 'value' in value
    ? (value as { value: T }).value
    : (value as T)
}

function isNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}

function isWindowRect(value: unknown): value is WindowRect {
  if (!value || typeof value !== 'object') {
    return false
  }
  const candidate = value as Record<string, unknown>
  return (
    isNumber(candidate.x) &&
    isNumber(candidate.y) &&
    isNumber(candidate.w) &&
    isNumber(candidate.h) &&
    isNumber(candidate.z)
  )
}

function fallbackRect(index: number): WindowRect {
  return {
    x: 24 + (index % 2) * 520,
    y: 24 + Math.floor(index / 2) * 280,
    w: 500,
    h: 260,
    z: index + 1,
  }
}

function createStateFromManifest(manifest: WindowPaneManifest[]): WindowState {
  const panes: WindowState['panes'] = {}
  manifest.forEach((pane, index) => {
    panes[pane.id] = {
      x: pane.x,
      y: pane.y,
      w: pane.w,
      h: pane.h,
      z: index + 1,
    }
  })
  return {
    panes,
    nextZ: Math.max(1, manifest.length + 1),
  }
}

function parseState(raw: string | null): WindowState | null {
  if (!raw) {
    return null
  }
  try {
    const parsed = JSON.parse(raw) as WindowState
    if (!parsed || typeof parsed !== 'object' || !parsed.panes || !isNumber(parsed.nextZ)) {
      return null
    }
    const entries = Object.entries(parsed.panes)
    if (entries.some(([, value]) => !isWindowRect(value))) {
      return null
    }
    return parsed
  } catch {
    return null
  }
}

function mergeWithManifest(state: WindowState, manifest: WindowPaneManifest[]): WindowState {
  const nextPanes: WindowState['panes'] = {}
  let maxZ = 1

  manifest.forEach((pane, index) => {
    const existing = state.panes[pane.id]
    const rect = existing && isWindowRect(existing) ? existing : fallbackRect(index)

    nextPanes[pane.id] = {
      x: rect.x,
      y: rect.y,
      w: rect.w,
      h: rect.h,
      z: rect.z,
    }
    maxZ = Math.max(maxZ, rect.z)
  })

  return {
    panes: nextPanes,
    nextZ: Math.max(maxZ + 1, 2),
  }
}

export function windowStorageKey(routeKey: string, mode: LayoutMode): string {
  return `${STORAGE_PREFIX}.${mode}.${routeKey}.v1`
}

export function useWindowLayout(
  routeKeyInput: MaybeRef<string>,
  modeInput: MaybeRef<LayoutMode>,
  manifestInput: MaybeRef<WindowPaneManifest[]>,
) {
  const state = ref<WindowState>({ panes: {}, nextZ: 1 })

  const routeKey = computed(() => resolveMaybeRef(routeKeyInput))
  const mode = computed(() => resolveMaybeRef(modeInput))
  const manifest = computed(() => resolveMaybeRef(manifestInput))
  const storageKey = computed(() => windowStorageKey(routeKey.value, mode.value))

  function load(): void {
    const defaultState = createStateFromManifest(manifest.value)
    const persisted = parseState(localStorage.getItem(storageKey.value))
    const merged = persisted ? mergeWithManifest(persisted, manifest.value) : defaultState
    state.value = merged
  }

  function persist(): void {
    localStorage.setItem(storageKey.value, JSON.stringify(state.value))
  }

  function bringToFront(paneId: string): void {
    const target = state.value.panes[paneId]
    if (!target) {
      return
    }
    target.z = state.value.nextZ
    state.value.nextZ += 1
  }

  function updatePaneRect(paneId: string, next: WindowRect): void {
    if (!state.value.panes[paneId]) {
      return
    }
    state.value.panes[paneId] = { ...next }
  }

  function resetLayout(): void {
    state.value = createStateFromManifest(manifest.value)
  }

  watch([storageKey, manifest], () => {
    load()
  }, { immediate: true, deep: true })

  watch(
    state,
    () => {
      persist()
    },
    { deep: true },
  )

  return {
    state,
    panes: computed(() => state.value.panes),
    bringToFront,
    updatePaneRect,
    resetLayout,
    storageKey,
  }
}
