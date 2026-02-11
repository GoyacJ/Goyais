/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { nextTick } from 'vue'
import { __resetLayoutSystemForTests, initLayoutSystem, layoutStorageKey, useLayoutStore } from '@/design-system/layout'

describe('layout system', () => {
  beforeEach(() => {
    __resetLayoutSystemForTests()
    localStorage.clear()
  })

  it('uses console preference by default and ignores non-console route defaults', async () => {
    initLayoutSystem()
    const store = useLayoutStore()

    expect(store.layoutPreference.value).toBe('console')
    expect(store.effectiveLayout.value).toBe('console')

    store.setRouteLayoutDefault('focus')
    await nextTick()
    expect(store.routeLayoutDefault.value).toBe('console')
    expect(store.effectiveLayout.value).toBe('console')
    expect(document.documentElement.getAttribute('data-layout')).toBe('console')
  })

  it('normalizes any manual selection to console and persists console value', async () => {
    initLayoutSystem()
    const store = useLayoutStore()

    store.setRouteLayoutDefault('focus')
    store.setLayoutPreference('topnav')
    await nextTick()

    expect(store.layoutPreference.value).toBe('console')
    expect(store.effectiveLayout.value).toBe('console')
    expect(localStorage.getItem(layoutStorageKey())).toBe('console')

    store.setLayoutPreference('focus')
    await nextTick()

    expect(store.layoutPreference.value).toBe('console')
    expect(store.effectiveLayout.value).toBe('console')
  })

  it('restores stored non-console preference as console', async () => {
    localStorage.setItem(layoutStorageKey(), 'focus')
    initLayoutSystem()

    const store = useLayoutStore()
    await nextTick()

    expect(store.layoutPreference.value).toBe('console')
    expect(store.effectiveLayout.value).toBe('console')
    expect(document.documentElement.getAttribute('data-layout-pref')).toBe('console')
  })
})
