/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Verify identity store persistence and runtime context sync.
 */

import { beforeEach, describe, expect, it } from 'vitest'
import { getApiRuntimeConfig, setApiRuntimeContext } from '@/api/http'
import {
  __resetIdentitySystemForTests,
  addAccount,
  identityStorageKey,
  initIdentitySystem,
  signOutCurrentAccount,
  switchAccount,
  useIdentityStore,
} from '@/design-system/identity'

const DEFAULT_CONTEXT = {
  tenantId: 't1',
  workspaceId: 'w1',
  userId: 'u1',
  roles: 'member',
  policyVersion: 'v0.1',
}

describe('identity store', () => {
  beforeEach(() => {
    setApiRuntimeContext(DEFAULT_CONTEXT)
    localStorage.removeItem(identityStorageKey())
    __resetIdentitySystemForTests()
    initIdentitySystem()
  })

  it('loads fallback account from runtime context', () => {
    const { accounts, activeAccount, activeWorkspace } = useIdentityStore()

    expect(accounts.value).toHaveLength(1)
    expect(activeAccount.value?.tenantId).toBe('t1')
    expect(activeWorkspace.value?.id).toBe('w1')
  })

  it('adds account and syncs runtime headers to the new account context', () => {
    addAccount({
      nickname: 'Bob',
      userId: 'u_bob',
      tenantId: 't_bob',
      roles: 'admin',
      policyVersion: 'v1.0',
      workspaceId: 'w_bob',
      workspaceName: 'Bob Workspace',
    })

    const runtime = getApiRuntimeConfig()
    expect(runtime.tenantId).toBe('t_bob')
    expect(runtime.workspaceId).toBe('w_bob')
    expect(runtime.userId).toBe('u_bob')
    expect(runtime.roles).toBe('admin')
    expect(runtime.policyVersion).toBe('v1.0')
  })

  it('signs out current account and falls back to default account when none left', () => {
    const { activeAccount } = useIdentityStore()

    expect(activeAccount.value?.id).toBe('account_default')

    signOutCurrentAccount()

    expect(activeAccount.value?.id).toBe('account_default')
    expect(activeAccount.value?.tenantId).toBe('t1')
    expect(activeAccount.value?.activeWorkspaceId).toBe('w1')
  })

  it('switches between two accounts and persists active account id', () => {
    addAccount({
      nickname: 'Alice',
      userId: 'u_alice',
      tenantId: 't_alice',
      roles: 'member',
      policyVersion: 'v0.2',
      workspaceId: 'w_alice',
      workspaceName: 'Alice Workspace',
    })

    addAccount({
      nickname: 'Charlie',
      userId: 'u_charlie',
      tenantId: 't_charlie',
      roles: 'owner',
      policyVersion: 'v0.3',
      workspaceId: 'w_charlie',
      workspaceName: 'Charlie Workspace',
    })

    const { accounts, activeAccount } = useIdentityStore()
    const target = accounts.value.find((item) => item.nickname === 'Alice')
    expect(target).toBeDefined()

    switchAccount(target!.id)

    expect(activeAccount.value?.id).toBe(target!.id)
    const raw = localStorage.getItem(identityStorageKey()) ?? ''
    expect(raw).toContain(target!.id)
  })
})
