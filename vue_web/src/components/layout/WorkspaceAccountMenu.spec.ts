/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Verify workspace/account menu interactions and account creation flow.
 */

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import WorkspaceAccountMenu from '@/components/layout/WorkspaceAccountMenu.vue'
import {
  __resetIdentitySystemForTests,
  identityStorageKey,
  initIdentitySystem,
  useIdentityStore,
} from '@/design-system/identity'
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

describe('WorkspaceAccountMenu', () => {
  beforeEach(() => {
    localStorage.removeItem(identityStorageKey())
    __resetIdentitySystemForTests()
    initIdentitySystem()
    i18n.global.locale.value = 'en-US'
  })

  it('creates account from dialog and switches active account', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    const wrapper = mount(WorkspaceAccountMenu, {
      attachTo: document.body,
      global: {
        plugins: [router, i18n],
      },
    })

    await wrapper.get('button[aria-label="Open workspace menu"]').trigger('click')

    const addAction = wrapper
      .findAll('button')
      .find((button) => button.text().includes('Add another account'))

    expect(addAction).toBeDefined()
    await addAction!.trigger('click')
    await flushPromises()

    const inputs = Array.from(document.body.querySelectorAll('input'))
    expect(inputs.length).toBeGreaterThanOrEqual(7)

    const nextValues = ['Demo User', 't_demo', 'u_demo', 'owner', 'v2.0', 'w_demo', 'Demo Workspace']
    for (const [index, value] of nextValues.entries()) {
      const input = inputs[index]
      if (!input) {
        continue
      }
      input.value = value
      input.dispatchEvent(new Event('input'))
    }
    await flushPromises()

    const submitButton = Array.from(document.body.querySelectorAll('button')).find((button) =>
      (button.textContent ?? '').includes('Create account'),
    )

    expect(submitButton).toBeDefined()
    submitButton!.click()
    await flushPromises()

    const { accounts, activeAccount } = useIdentityStore()
    expect(accounts.value).toHaveLength(2)
    expect(activeAccount.value?.nickname).toBe('Demo User')

    wrapper.unmount()
  })

  it('navigates to settings when selecting settings action', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    const wrapper = mount(WorkspaceAccountMenu, {
      attachTo: document.body,
      global: {
        plugins: [router, i18n],
      },
    })

    await wrapper.get('button[aria-label="Open workspace menu"]').trigger('click')

    const settingsAction = wrapper
      .findAll('button')
      .find((button) => button.text().includes('Settings'))

    expect(settingsAction).toBeDefined()
    await settingsAction!.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/settings')
    wrapper.unmount()
  })
})
