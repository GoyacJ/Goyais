/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { computed, ref, watch, type MaybeRef } from 'vue'
import type { LayoutMode, WindowRect, WindowState } from '@/design-system/types'
import type { WindowPaneManifest } from '@/design-system/window-manifests'

const STORAGE_PREFIX = 'goyais.ui.windows'
const DEFAULT_PERSIST_DEBOUNCE_MS = 120

export interface WindowLayoutOptions {
  persistDebounceMs?: number
}

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
    const rect = existing && isWindowRect(existing)
      ? existing
      : {
          x: pane.x,
          y: pane.y,
          w: pane.w,
          h: pane.h,
          z: index + 1,
        }

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
  return `${STORAGE_PREFIX}.${mode}.${routeKey}.v2`
}

function safeReadStorage(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function safeWriteStorage(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    // Ignore persistence failures (private mode/quota exceeded).
  }
}

export function useWindowLayout(
  routeKeyInput: MaybeRef<string>,
  modeInput: MaybeRef<LayoutMode>,
  manifestInput: MaybeRef<WindowPaneManifest[]>,
  options: WindowLayoutOptions = {},
) {
  const state = ref<WindowState>({ panes: {}, nextZ: 1 })

  const routeKey = computed(() => resolveMaybeRef(routeKeyInput))
  const mode = computed(() => resolveMaybeRef(modeInput))
  const manifest = computed(() => resolveMaybeRef(manifestInput))
  const storageKey = computed(() => windowStorageKey(routeKey.value, mode.value))
  const persistDebounceMs = Math.max(0, options.persistDebounceMs ?? DEFAULT_PERSIST_DEBOUNCE_MS)

  let persistTimer: ReturnType<typeof setTimeout> | null = null
  let pendingPersist: { key: string; payload: string } | null = null

  function clearPersistTimer(): void {
    if (!persistTimer) {
      return
    }
    clearTimeout(persistTimer)
    persistTimer = null
  }

  function buildPersistSnapshot(): { key: string; payload: string } {
    return {
      key: storageKey.value,
      payload: JSON.stringify(state.value),
    }
  }

  function persistSnapshot(snapshot: { key: string; payload: string }): void {
    safeWriteStorage(snapshot.key, snapshot.payload)
  }

  function load(): void {
    const defaultState = createStateFromManifest(manifest.value)
    const persisted = parseState(safeReadStorage(storageKey.value))
    const merged = persisted ? mergeWithManifest(persisted, manifest.value) : defaultState
    state.value = merged
  }

  function schedulePersist(): void {
    const snapshot = buildPersistSnapshot()
    if (persistDebounceMs === 0) {
      persistSnapshot(snapshot)
      return
    }

    pendingPersist = snapshot
    clearPersistTimer()
    persistTimer = setTimeout(() => {
      persistTimer = null
      if (!pendingPersist) {
        return
      }
      persistSnapshot(pendingPersist)
      pendingPersist = null
    }, persistDebounceMs)
  }

  function flushPersist(forceCurrent = false): void {
    clearPersistTimer()
    if (pendingPersist) {
      persistSnapshot(pendingPersist)
      pendingPersist = null
      return
    }
    if (forceCurrent) {
      persistSnapshot(buildPersistSnapshot())
    }
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

  function resetLayout(overrideManifest?: WindowPaneManifest[]): void {
    state.value = createStateFromManifest(overrideManifest ?? manifest.value)
  }

  watch([storageKey, manifest], () => {
    flushPersist(false)
    load()
  }, { immediate: true, deep: true })

  watch(
    state,
    () => {
      schedulePersist()
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
    flushPersist,
  }
}
