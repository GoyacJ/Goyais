/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Verify user account menu account switching and settings navigation.
 */

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import UserAccountMenu from '@/components/layout/UserAccountMenu.vue'
import { __resetIdentitySystemForTests, addAccount, identityStorageKey, initIdentitySystem, useIdentityStore } from '@/design-system/identity'
import i18n from '@/i18n'

function createTestRouter() {
  const view = { template: '<div>view</div>' }
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: view },
      { path: '/settings', component: view },
    ],
  })
}

describe('UserAccountMenu', () => {
  beforeEach(() => {
    localStorage.removeItem(identityStorageKey())
    __resetIdentitySystemForTests()
    initIdentitySystem()
    i18n.global.locale.value = 'en-US'
  })

  it('switches account from dropdown list', async () => {
    addAccount({
      nickname: 'Bob',
      userId: 'u_bob',
      tenantId: 't_bob',
      roles: 'admin',
      policyVersion: 'v2',
      workspaceId: 'w_bob',
      workspaceName: 'Bob Space',
    })

    const { activeAccount } = useIdentityStore()
    expect(activeAccount.value?.nickname).toBe('Bob')

    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    const wrapper = mount(UserAccountMenu, {
      global: {
        plugins: [router, i18n],
      },
    })

    await wrapper.get('button[aria-label="User menu"]').trigger('click')

    const target = wrapper.findAll('button').find((button) => button.text().includes('u1'))
    expect(target).toBeDefined()

    await target!.trigger('click')
    await flushPromises()

    expect(activeAccount.value?.nickname).toBe('u1')
  })

  it('navigates to settings from user actions', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    const wrapper = mount(UserAccountMenu, {
      global: {
        plugins: [router, i18n],
      },
    })

    await wrapper.get('button[aria-label="User menu"]').trigger('click')

    const settingsAction = wrapper.findAll('button').find((button) => button.text().includes('Settings'))
    expect(settingsAction).toBeDefined()

    await settingsAction!.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/settings')
  })
})
