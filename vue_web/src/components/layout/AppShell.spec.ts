/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'
import TopBar from '@/components/layout/TopBar.vue'
import SideNav from '@/components/layout/SideNav.vue'
import i18n from '@/i18n'
import { __resetLayoutSystemForTests, initLayoutSystem, useLayoutStore } from '@/design-system/layout'

function createTestRouter() {
  const view = { template: '<div>view</div>' }
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/',
        component: view,
        meta: { layoutDefault: 'console', windowManifestKey: 'home' },
      },
      { path: '/canvas', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'canvas' } },
      { path: '/ai', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'ai-workbench' } },
      { path: '/run-center', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'run-center' } },
      { path: '/commands', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'commands' } },
      { path: '/assets', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'assets' } },
      {
        path: '/algorithm-library',
        component: view,
        meta: { layoutDefault: 'console', windowManifestKey: 'algorithm-library' },
      },
      { path: '/plugins', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'plugins' } },
      { path: '/streams', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'streams' } },
      {
        path: '/permissions',
        component: view,
        meta: { layoutDefault: 'console', windowManifestKey: 'permission-management' },
      },
      {
        path: '/context-bundles',
        component: view,
        meta: { layoutDefault: 'console', windowManifestKey: 'context-bundles' },
      },
      { path: '/settings', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'settings' } },
    ],
  })
}

describe('AppShell console-only layout', () => {
  beforeEach(() => {
    __resetLayoutSystemForTests()
    localStorage.clear()
  })

  it('renders SideNav in console mode', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()
    initLayoutSystem(router)

    const { setLayoutPreference } = useLayoutStore()
    setLayoutPreference('console')

    const wrapper = mount(AppShell, {
      global: {
        plugins: [router, i18n],
      },
    })

    await flushPromises()
    expect(wrapper.findComponent(SideNav).exists()).toBe(true)
  })

  it('keeps side navigation visible even when setting topnav or focus preferences', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()
    initLayoutSystem(router)

    const { setLayoutPreference } = useLayoutStore()
    setLayoutPreference('topnav')

    const wrapper = mount(AppShell, {
      global: {
        plugins: [router, i18n],
      },
    })

    await flushPromises()
    expect(wrapper.findComponent(SideNav).exists()).toBe(true)

    setLayoutPreference('focus')
    await flushPromises()
    expect(wrapper.findComponent(SideNav).exists()).toBe(true)
  })

  it('hides shell chrome and enters immersive main mode for valid immersive query', async () => {
    const router = createTestRouter()
    await router.push('/commands?wbMode=immersive&wbPane=list')
    await router.isReady()
    initLayoutSystem(router)

    const { setLayoutPreference } = useLayoutStore()
    setLayoutPreference('console')

    const wrapper = mount(AppShell, {
      global: {
        plugins: [router, i18n],
      },
    })

    await flushPromises()
    expect(wrapper.findComponent(SideNav).exists()).toBe(false)
    expect(wrapper.findComponent(TopBar).exists()).toBe(false)
    expect(wrapper.find('main').classes()).toContain('ui-shell-main--immersive')
  })
})
