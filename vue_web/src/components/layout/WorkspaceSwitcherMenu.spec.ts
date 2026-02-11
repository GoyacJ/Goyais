/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Verify workspace switcher menu interaction.
 */

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import WorkspaceSwitcherMenu from '@/components/layout/WorkspaceSwitcherMenu.vue'
import { __resetIdentitySystemForTests, addAccount, identityStorageKey, initIdentitySystem, switchWorkspace, useIdentityStore } from '@/design-system/identity'
import i18n from '@/i18n'

describe('WorkspaceSwitcherMenu', () => {
  beforeEach(() => {
    localStorage.removeItem(identityStorageKey())
    __resetIdentitySystemForTests()
    initIdentitySystem()
    i18n.global.locale.value = 'en-US'
  })

  it('switches workspace within active account', async () => {
    addAccount({
      nickname: 'alice',
      userId: 'u_alice',
      tenantId: 't_alice',
      roles: 'member',
      policyVersion: 'v1',
      workspaceId: 'w_alpha',
      workspaceName: 'Alpha',
    })

    const { activeAccount, activeWorkspace } = useIdentityStore()
    activeAccount.value?.workspaces.push({ id: 'w_beta', name: 'Beta' })
    switchWorkspace('w_alpha')

    const wrapper = mount(WorkspaceSwitcherMenu, {
      global: {
        plugins: [i18n],
      },
    })

    await wrapper.get('button[aria-label="Open workspace menu"]').trigger('click')
    const target = wrapper.findAll('button').find((button) => button.text().includes('Beta'))
    expect(target).toBeDefined()

    await target!.trigger('click')
    await flushPromises()

    expect(activeWorkspace.value?.id).toBe('w_beta')
  })
})
