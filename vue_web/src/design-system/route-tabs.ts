/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Persist and manage dynamic route tabs for console navigation.
 */

import { NAV_ITEMS } from '@/design-system/navigation'
import { computed, ref } from 'vue'
import type { Router } from 'vue-router'

const STORAGE_KEY = 'goyais.ui.route-tabs'
const HOME_PATH = '/'

interface RouteTabsSnapshot {
  tabs: string[]
  recentPaths: string[]
}

const navPathSet = new Set(NAV_ITEMS.map((item) => item.to))

const tabs = ref<string[]>([HOME_PATH])
const recentPaths = ref<string[]>([HOME_PATH])
const activeTabPath = ref(HOME_PATH)

let initialized = false
let boundRouter: Router | null = null
let removeAfterEachHook: (() => void) | null = null

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0
}

function normalizePath(value: string): string {
  const trimmed = value.trim()
  return trimmed.length > 0 ? trimmed : HOME_PATH
}

function isNavPath(value: string): boolean {
  return navPathSet.has(value)
}

function uniquePaths(paths: string[]): string[] {
  const result: string[] = []
  const seen = new Set<string>()

  for (const path of paths) {
    if (seen.has(path)) {
      continue
    }
    seen.add(path)
    result.push(path)
  }

  return result
}

function sanitizePaths(raw: unknown): string[] {
  if (!Array.isArray(raw)) {
    return []
  }

  return uniquePaths(
    raw
      .filter(isNonEmptyString)
      .map((item) => normalizePath(item))
      .filter((item) => isNavPath(item)),
  )
}

function readSnapshot(): RouteTabsSnapshot {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return {
        tabs: [HOME_PATH],
        recentPaths: [HOME_PATH],
      }
    }

    const parsed = JSON.parse(raw) as Record<string, unknown>
    const persistedTabs = sanitizePaths(parsed.tabs)
    const persistedRecentPaths = sanitizePaths(parsed.recentPaths)

    const normalizedTabs = persistedTabs.length > 0 ? persistedTabs : [HOME_PATH]
    const normalizedRecent = uniquePaths([...persistedRecentPaths.filter((item) => normalizedTabs.includes(item)), ...normalizedTabs])

    return {
      tabs: normalizedTabs,
      recentPaths: normalizedRecent,
    }
  } catch {
    return {
      tabs: [HOME_PATH],
      recentPaths: [HOME_PATH],
    }
  }
}

function persistSnapshot(): void {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        tabs: tabs.value,
        recentPaths: recentPaths.value,
      } satisfies RouteTabsSnapshot),
    )
  } catch {
    // Ignore storage failures (private mode / quota).
  }
}

function touchRecentPath(path: string): void {
  recentPaths.value = [...recentPaths.value.filter((item) => item !== path), path]
}

function ensureTabExists(path: string): void {
  if (tabs.value.includes(path)) {
    return
  }
  tabs.value = [...tabs.value, path]
}

function trackRoute(path: string): void {
  const normalized = normalizePath(path)
  if (!isNavPath(normalized)) {
    return
  }

  ensureTabExists(normalized)
  activeTabPath.value = normalized
  touchRecentPath(normalized)
  persistSnapshot()
}

function bindRouter(router: Router): void {
  boundRouter = router
  removeAfterEachHook?.()
  removeAfterEachHook = router.afterEach((to) => {
    trackRoute(to.path)
  })
}

export function initRouteTabsSystem(router: Router): void {
  if (!initialized) {
    const snapshot = readSnapshot()
    tabs.value = snapshot.tabs
    recentPaths.value = snapshot.recentPaths
    activeTabPath.value = snapshot.tabs[0] ?? HOME_PATH
    initialized = true
  }

  bindRouter(router)
  trackRoute(router.currentRoute.value.path)
}

export function navigateTab(path: string): void {
  const normalized = normalizePath(path)
  if (!boundRouter || !isNavPath(normalized)) {
    return
  }

  void boundRouter.push(normalized)
}

export function openTab(path: string): void {
  const normalized = normalizePath(path)
  if (!isNavPath(normalized)) {
    return
  }

  ensureTabExists(normalized)
  touchRecentPath(normalized)
  persistSnapshot()
  navigateTab(normalized)
}

export function closeTab(path: string): void {
  const normalized = normalizePath(path)
  if (!tabs.value.includes(normalized)) {
    return
  }

  const nextTabs = tabs.value.filter((item) => item !== normalized)
  const safeTabs = nextTabs.length > 0 ? nextTabs : [HOME_PATH]
  tabs.value = safeTabs
  recentPaths.value = recentPaths.value.filter((item) => item !== normalized)

  const shouldChangeActive = activeTabPath.value === normalized || !safeTabs.includes(activeTabPath.value)
  if (shouldChangeActive) {
    const fallback = [...recentPaths.value]
      .reverse()
      .find((item) => safeTabs.includes(item)) ?? safeTabs[0] ?? HOME_PATH
    activeTabPath.value = fallback
    navigateTab(fallback)
  }

  persistSnapshot()
}

export function routeTabsStorageKey(): string {
  return STORAGE_KEY
}

export function useRouteTabsStore() {
  const availableNavItems = computed(() => NAV_ITEMS.filter((item) => !tabs.value.includes(item.to)))

  return {
    tabs,
    activeTabPath,
    availableNavItems,
    openTab,
    closeTab,
    navigateTab,
  }
}

export function __resetRouteTabsSystemForTests(): void {
  initialized = false
  removeAfterEachHook?.()
  removeAfterEachHook = null
  boundRouter = null
  tabs.value = [HOME_PATH]
  recentPaths.value = [HOME_PATH]
  activeTabPath.value = HOME_PATH
}
