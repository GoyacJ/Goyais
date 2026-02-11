/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Verify RouteTabBar rendering and add-tab interaction.
 */

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import RouteTabBar from '@/components/layout/RouteTabBar.vue'
import { __resetRouteTabsSystemForTests, initRouteTabsSystem, routeTabsStorageKey } from '@/design-system/route-tabs'
import i18n from '@/i18n'

function createTestRouter() {
  const view = { template: '<div>view</div>' }
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: view },
      { path: '/commands', component: view },
      { path: '/plugins', component: view },
    ],
  })
}

describe('RouteTabBar', () => {
  beforeEach(() => {
    localStorage.removeItem(routeTabsStorageKey())
    __resetRouteTabsSystemForTests()
    i18n.global.locale.value = 'en-US'
  })

  it('renders current route tab and opens a new tab from plus menu', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    initRouteTabsSystem(router)

    const wrapper = mount(RouteTabBar, {
      global: {
        plugins: [router, i18n],
      },
    })

    expect(wrapper.text()).toContain('Home')

    await wrapper.get('button[aria-label="Open new tab menu"]').trigger('click')

    const pluginsOption = wrapper.findAll('button').find((button) => button.text().includes('Plugins'))
    expect(pluginsOption).toBeDefined()

    await pluginsOption!.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/plugins')
    expect(wrapper.text()).toContain('Plugins')
  })
})
