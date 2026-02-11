/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Verify route tabs open, close, and fallback navigation behavior.
 */

import { beforeEach, describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import {
  __resetRouteTabsSystemForTests,
  closeTab,
  initRouteTabsSystem,
  openTab,
  routeTabsStorageKey,
  useRouteTabsStore,
} from '@/design-system/route-tabs'

function createTestRouter() {
  const view = { template: '<div>view</div>' }
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: view },
      { path: '/commands', component: view },
      { path: '/assets', component: view },
      { path: '/plugins', component: view },
    ],
  })
}

describe('route tabs store', () => {
  beforeEach(() => {
    __resetRouteTabsSystemForTests()
    localStorage.removeItem(routeTabsStorageKey())
  })

  it('tracks visited routes into tabs', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    initRouteTabsSystem(router)

    await router.push('/commands')
    await router.push('/assets')

    const { tabs, activeTabPath } = useRouteTabsStore()
    expect(tabs.value).toContain('/')
    expect(tabs.value).toContain('/commands')
    expect(tabs.value).toContain('/assets')
    expect(activeTabPath.value).toBe('/assets')
  })

  it('opens new tab via openTab and navigates to it', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    initRouteTabsSystem(router)
    openTab('/plugins')

    await new Promise((resolve) => setTimeout(resolve, 0))

    const { tabs, activeTabPath } = useRouteTabsStore()
    expect(tabs.value).toContain('/plugins')
    expect(activeTabPath.value).toBe('/plugins')
  })

  it('closes active tab and falls back to recent tab', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    initRouteTabsSystem(router)

    await router.push('/commands')
    await router.push('/assets')

    closeTab('/assets')
    await new Promise((resolve) => setTimeout(resolve, 0))

    const { tabs, activeTabPath } = useRouteTabsStore()
    expect(tabs.value).not.toContain('/assets')
    expect(activeTabPath.value).toBe('/commands')
  })
})
