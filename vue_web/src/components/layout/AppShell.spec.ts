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
import TopNavBar from '@/components/layout/TopNavBar.vue'
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
      { path: '/canvas', component: view, meta: { layoutDefault: 'focus', windowManifestKey: 'canvas' } },
      { path: '/ai', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'ai-workbench' } },
      { path: '/commands', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'commands' } },
      { path: '/assets', component: view, meta: { layoutDefault: 'console', windowManifestKey: 'assets' } },
      { path: '/plugins', component: view, meta: { layoutDefault: 'topnav', windowManifestKey: 'plugins' } },
      { path: '/streams', component: view, meta: { layoutDefault: 'topnav', windowManifestKey: 'streams' } },
      { path: '/settings', component: view, meta: { layoutDefault: 'topnav', windowManifestKey: 'settings' } },
    ],
  })
}

describe('AppShell layout switching', () => {
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
    expect(wrapper.findComponent(TopNavBar).exists()).toBe(false)
  })

  it('renders TopNavBar in topnav mode', async () => {
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
    expect(wrapper.findComponent(SideNav).exists()).toBe(false)
    expect(wrapper.findComponent(TopNavBar).exists()).toBe(true)
  })

  it('hides both nav containers in focus mode', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()
    initLayoutSystem(router)

    const { setLayoutPreference } = useLayoutStore()
    setLayoutPreference('focus')

    const wrapper = mount(AppShell, {
      global: {
        plugins: [router, i18n],
      },
    })

    await flushPromises()
    expect(wrapper.findComponent(SideNav).exists()).toBe(false)
    expect(wrapper.findComponent(TopNavBar).exists()).toBe(false)
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
    expect(wrapper.findComponent(TopNavBar).exists()).toBe(false)
    expect(wrapper.findComponent(TopBar).exists()).toBe(false)
    expect(wrapper.find('main').classes()).toContain('ui-shell-main--immersive')
  })
})
