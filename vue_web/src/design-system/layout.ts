/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { computed, ref, watch } from 'vue'
import type { Router, RouteLocationNormalizedLoaded } from 'vue-router'
import type { LayoutMode, LayoutPreference } from '@/design-system/types'

const STORAGE_KEY = 'goyais.ui.layout'
const CONSOLE_LAYOUT: LayoutMode = 'console'

const layoutPreference = ref<LayoutPreference>(CONSOLE_LAYOUT)
const routeLayoutDefault = ref<LayoutMode>(CONSOLE_LAYOUT)

const effectiveLayout = computed<LayoutMode>(() => CONSOLE_LAYOUT)

let initialized = false
let unwatchRouter: (() => void) | null = null

function isLayoutMode(value: string | null): value is LayoutMode {
  return value === CONSOLE_LAYOUT
}

function isLayoutPreference(value: string | null): value is LayoutPreference {
  return isLayoutMode(value)
}

function readLayoutPreference(): LayoutPreference {
  const stored = localStorage.getItem(STORAGE_KEY)
  return isLayoutPreference(stored) ? stored : CONSOLE_LAYOUT
}

function resolveRouteLayout(route: RouteLocationNormalizedLoaded | null): LayoutMode {
  const candidate = route?.meta?.layoutDefault
  if (typeof candidate === 'string' && isLayoutMode(candidate)) {
    return candidate
  }
  return CONSOLE_LAYOUT
}

function applyLayoutAttributes(): void {
  document.documentElement.setAttribute('data-layout', CONSOLE_LAYOUT)
  document.documentElement.setAttribute('data-layout-pref', CONSOLE_LAYOUT)
}

function bindRouter(router: Router): void {
  routeLayoutDefault.value = CONSOLE_LAYOUT
  unwatchRouter?.()
  unwatchRouter = router.afterEach(() => {
    routeLayoutDefault.value = CONSOLE_LAYOUT
  })
}

export function initLayoutSystem(router?: Router): void {
  if (initialized) {
    if (router) {
      bindRouter(router)
    }
    return
  }

  initialized = true
  layoutPreference.value = readLayoutPreference()
  routeLayoutDefault.value = resolveRouteLayout(router?.currentRoute.value ?? null)
  if (router) {
    bindRouter(router)
  }

  localStorage.setItem(STORAGE_KEY, CONSOLE_LAYOUT)
  applyLayoutAttributes()

  watch(
    layoutPreference,
    () => {
      layoutPreference.value = CONSOLE_LAYOUT
      localStorage.setItem(STORAGE_KEY, CONSOLE_LAYOUT)
      applyLayoutAttributes()
    },
    { flush: 'post' },
  )

  watch(routeLayoutDefault, () => {
    routeLayoutDefault.value = CONSOLE_LAYOUT
    applyLayoutAttributes()
  })
}

export function useLayoutStore() {
  return {
    layoutPreference,
    routeLayoutDefault,
    effectiveLayout,
    setLayoutPreference: (_value: LayoutPreference) => {
      layoutPreference.value = CONSOLE_LAYOUT
      localStorage.setItem(STORAGE_KEY, CONSOLE_LAYOUT)
      applyLayoutAttributes()
    },
    setRouteLayoutDefault: (_value: LayoutMode) => {
      routeLayoutDefault.value = CONSOLE_LAYOUT
      applyLayoutAttributes()
    },
  }
}

export function layoutStorageKey(): string {
  return STORAGE_KEY
}

export function __resetLayoutSystemForTests(): void {
  unwatchRouter?.()
  unwatchRouter = null
  initialized = false
  layoutPreference.value = CONSOLE_LAYOUT
  routeLayoutDefault.value = CONSOLE_LAYOUT
}
