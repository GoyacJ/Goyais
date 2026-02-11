/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Manage frontend identity and workspace context for account switching.
 */

import { getApiRuntimeConfig, setApiRuntimeContext } from '@/api/http'
import { computed, ref } from 'vue'

const STORAGE_KEY = 'goyais.ui.identity'

export interface IdentityWorkspace {
  id: string
  name: string
}

export interface IdentityAccount {
  id: string
  nickname: string
  userId: string
  tenantId: string
  roles: string
  policyVersion: string
  workspaces: IdentityWorkspace[]
  activeWorkspaceId: string
}

interface IdentitySnapshot {
  accounts: IdentityAccount[]
  activeAccountId: string
}

export interface IdentityAccountDraft {
  nickname: string
  userId: string
  tenantId: string
  roles: string
  policyVersion: string
  workspaceId: string
  workspaceName: string
}

const accounts = ref<IdentityAccount[]>([])
const activeAccountId = ref('')

let initialized = false

const activeAccount = computed<IdentityAccount | null>(() => {
  if (accounts.value.length === 0) {
    return null
  }
  return accounts.value.find((item) => item.id === activeAccountId.value) ?? accounts.value[0] ?? null
})

const activeWorkspace = computed<IdentityWorkspace | null>(() => {
  const account = activeAccount.value
  if (!account) {
    return null
  }
  return account.workspaces.find((item) => item.id === account.activeWorkspaceId) ?? account.workspaces[0] ?? null
})

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0
}

function sanitizeWorkspace(raw: unknown, index: number): IdentityWorkspace | null {
  if (!raw || typeof raw !== 'object') {
    return null
  }

  const source = raw as Record<string, unknown>
  const id = isNonEmptyString(source.id) ? source.id.trim() : `workspace_${index + 1}`
  const name = isNonEmptyString(source.name) ? source.name.trim() : id
  return { id, name }
}

function sanitizeAccount(raw: unknown, index: number): IdentityAccount | null {
  if (!raw || typeof raw !== 'object') {
    return null
  }

  const source = raw as Record<string, unknown>
  if (!isNonEmptyString(source.userId) || !isNonEmptyString(source.tenantId)) {
    return null
  }

  const workspacesRaw = Array.isArray(source.workspaces) ? source.workspaces : []
  const workspaces = workspacesRaw
    .map((item, itemIndex) => sanitizeWorkspace(item, itemIndex))
    .filter((item): item is IdentityWorkspace => item !== null)

  if (workspaces.length === 0) {
    return null
  }

  const accountId = isNonEmptyString(source.id) ? source.id.trim() : `account_${index + 1}`
  const nickname = isNonEmptyString(source.nickname) ? source.nickname.trim() : String(source.userId).trim()
  const roles = isNonEmptyString(source.roles) ? source.roles.trim() : 'member'
  const policyVersion = isNonEmptyString(source.policyVersion) ? source.policyVersion.trim() : 'v0.1'

  const preferredWorkspaceId = isNonEmptyString(source.activeWorkspaceId) ? source.activeWorkspaceId.trim() : workspaces[0].id
  const activeWorkspaceId = workspaces.some((item) => item.id === preferredWorkspaceId)
    ? preferredWorkspaceId
    : workspaces[0].id

  return {
    id: accountId,
    nickname,
    userId: String(source.userId).trim(),
    tenantId: String(source.tenantId).trim(),
    roles,
    policyVersion,
    workspaces,
    activeWorkspaceId,
  }
}

function buildFallbackAccount(): IdentityAccount {
  const runtime = getApiRuntimeConfig()
  const workspaceId = runtime.workspaceId.trim() || 'w1'

  return {
    id: 'account_default',
    nickname: runtime.userId,
    userId: runtime.userId,
    tenantId: runtime.tenantId,
    roles: runtime.roles,
    policyVersion: runtime.policyVersion,
    workspaces: [
      {
        id: workspaceId,
        name: workspaceId,
      },
    ],
    activeWorkspaceId: workspaceId,
  }
}

function ensureState(snapshot?: IdentitySnapshot | null): IdentitySnapshot {
  const fallback = buildFallbackAccount()
  if (!snapshot || snapshot.accounts.length === 0) {
    return {
      accounts: [fallback],
      activeAccountId: fallback.id,
    }
  }

  const normalizedAccounts = snapshot.accounts.map((item) => {
    if (item.workspaces.length > 0 && item.workspaces.some((workspace) => workspace.id === item.activeWorkspaceId)) {
      return item
    }
    return {
      ...item,
      activeWorkspaceId: item.workspaces[0].id,
    }
  })

  const activeId = normalizedAccounts.some((item) => item.id === snapshot.activeAccountId)
    ? snapshot.activeAccountId
    : normalizedAccounts[0].id

  return {
    accounts: normalizedAccounts,
    activeAccountId: activeId,
  }
}

function readSnapshot(): IdentitySnapshot {
  const fallback = ensureState(null)

  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return fallback
    }

    const parsed = JSON.parse(raw) as Record<string, unknown>
    const parsedAccounts = Array.isArray(parsed.accounts) ? parsed.accounts : []
    const normalizedAccounts = parsedAccounts
      .map((item, index) => sanitizeAccount(item, index))
      .filter((item): item is IdentityAccount => item !== null)

    const rawActiveAccountId = isNonEmptyString(parsed.activeAccountId) ? parsed.activeAccountId.trim() : ''

    return ensureState({
      accounts: normalizedAccounts,
      activeAccountId: rawActiveAccountId,
    })
  } catch {
    return fallback
  }
}

function persistSnapshot(): void {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        accounts: accounts.value,
        activeAccountId: activeAccountId.value,
      } satisfies IdentitySnapshot),
    )
  } catch {
    // Ignore storage failures (private mode / quota).
  }
}

function syncRuntimeContext(): void {
  const account = activeAccount.value
  const workspace = activeWorkspace.value
  if (!account || !workspace) {
    return
  }

  setApiRuntimeContext({
    tenantId: account.tenantId,
    workspaceId: workspace.id,
    userId: account.userId,
    roles: account.roles,
    policyVersion: account.policyVersion,
  })
}

function applySnapshot(snapshot: IdentitySnapshot): void {
  accounts.value = snapshot.accounts
  activeAccountId.value = snapshot.activeAccountId
  persistSnapshot()
  syncRuntimeContext()
}

function createAccountId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `account_${crypto.randomUUID()}`
  }
  return `account_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`
}

function cleanText(value: string): string {
  return value.trim()
}

export function initIdentitySystem(): void {
  if (initialized) {
    return
  }

  initialized = true
  applySnapshot(readSnapshot())
}

export function switchAccount(accountId: string): void {
  const normalizedId = cleanText(accountId)
  if (!normalizedId || !accounts.value.some((item) => item.id === normalizedId)) {
    return
  }

  activeAccountId.value = normalizedId
  persistSnapshot()
  syncRuntimeContext()
}

export function switchWorkspace(workspaceId: string): void {
  const normalizedWorkspaceId = cleanText(workspaceId)
  if (!normalizedWorkspaceId) {
    return
  }

  const current = activeAccount.value
  if (!current || !current.workspaces.some((item) => item.id === normalizedWorkspaceId)) {
    return
  }

  accounts.value = accounts.value.map((item) =>
    item.id === current.id
      ? {
          ...item,
          activeWorkspaceId: normalizedWorkspaceId,
        }
      : item,
  )

  persistSnapshot()
  syncRuntimeContext()
}

export function addAccount(draft: IdentityAccountDraft): void {
  const nickname = cleanText(draft.nickname)
  const userId = cleanText(draft.userId)
  const tenantId = cleanText(draft.tenantId)
  const roles = cleanText(draft.roles)
  const policyVersion = cleanText(draft.policyVersion)
  const workspaceId = cleanText(draft.workspaceId)
  const workspaceName = cleanText(draft.workspaceName)

  if (!nickname || !userId || !tenantId || !workspaceId) {
    return
  }

  const nextAccount: IdentityAccount = {
    id: createAccountId(),
    nickname,
    userId,
    tenantId,
    roles: roles || 'member',
    policyVersion: policyVersion || 'v0.1',
    workspaces: [
      {
        id: workspaceId,
        name: workspaceName || workspaceId,
      },
    ],
    activeWorkspaceId: workspaceId,
  }

  accounts.value = [...accounts.value, nextAccount]
  activeAccountId.value = nextAccount.id
  persistSnapshot()
  syncRuntimeContext()
}

export function signOutCurrentAccount(): void {
  const current = activeAccount.value
  if (!current) {
    applySnapshot(ensureState(null))
    return
  }

  const remaining = accounts.value.filter((item) => item.id !== current.id)
  if (remaining.length === 0) {
    applySnapshot(ensureState(null))
    return
  }

  const nextActiveAccountId = remaining[0].id
  applySnapshot(
    ensureState({
      accounts: remaining,
      activeAccountId: nextActiveAccountId,
    }),
  )
}

export function identityStorageKey(): string {
  return STORAGE_KEY
}

export function useIdentityStore() {
  return {
    accounts,
    activeAccountId,
    activeAccount,
    activeWorkspace,
    switchAccount,
    switchWorkspace,
    addAccount,
    signOutCurrentAccount,
  }
}

export function __resetIdentitySystemForTests(): void {
  initialized = false
  accounts.value = []
  activeAccountId.value = ''
}
