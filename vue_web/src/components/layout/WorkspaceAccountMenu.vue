<template>
  <div class="relative">
    <Menu as="div" class="relative block" v-slot="{ open, close }">
      <MenuButton :class="triggerClasses" :aria-label="t('common.openWorkspaceMenu')">
        <span
          class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-ui-border bg-ui-surface2 text-xs font-semibold text-ui-fg"
        >
          {{ avatarText }}
        </span>

        <span v-if="!collapsed" class="min-w-0 flex-1 text-left">
          <p class="truncate text-[11px] uppercase tracking-[0.12em] text-ui-muted">{{ workspaceDisplayName }}</p>
          <p class="truncate text-sm font-semibold text-ui-fg">{{ accountDisplayName }}</p>
        </span>

        <Icon
          v-if="!collapsed"
          name="chevron-down"
          :size="14"
          decorative
          class="text-ui-muted transition-transform duration-120"
          :class="open ? 'rotate-180' : ''"
        />
      </MenuButton>

      <transition
        enter-active-class="transition duration-120 ease-out"
        enter-from-class="scale-95 opacity-0"
        enter-to-class="scale-100 opacity-100"
        leave-active-class="transition duration-90 ease-in"
        leave-from-class="scale-100 opacity-100"
        leave-to-class="scale-95 opacity-0"
      >
        <MenuItems v-if="open" class="ui-overlay-panel absolute left-0 z-40 mt-2 w-[18.5rem] origin-top-left p-2">
          <div class="rounded-[var(--ui-radius-card)] border border-ui-borderSubtle bg-ui-surface2 px-3 py-2">
            <p class="truncate text-sm font-semibold text-ui-fg">{{ workspaceDisplayName }}</p>
            <p class="truncate text-xs text-ui-muted">{{ accountDisplayName }}</p>
          </div>

          <section class="mt-2">
            <p class="px-2 pb-1 text-[11px] uppercase tracking-[0.12em] text-ui-muted">{{ t('common.workspaceList') }}</p>
            <MenuItem
              v-for="workspace in activeAccountWorkspaces"
              :key="workspace.id"
              as="template"
              v-slot="{ active }"
            >
              <button
                type="button"
                class="ui-focus-ring ui-pressable flex w-full items-center justify-between rounded-button border border-transparent px-2 py-2 text-left text-sm"
                :class="[active ? 'ui-state-hovered' : '']"
                @click="onSwitchWorkspace(workspace.id, close)"
              >
                <span class="truncate">{{ workspace.name }}</span>
                <Icon
                  v-if="workspace.id === activeWorkspaceID"
                  name="check"
                  :size="14"
                  decorative
                  class="ui-tone-text-primary"
                />
              </button>
            </MenuItem>
          </section>

          <section class="mt-2 border-t border-ui-borderSubtle pt-2">
            <p class="px-2 pb-1 text-[11px] uppercase tracking-[0.12em] text-ui-muted">{{ t('common.accountList') }}</p>
            <MenuItem
              v-for="account in accounts"
              :key="account.id"
              as="template"
              v-slot="{ active }"
            >
              <button
                type="button"
                class="ui-focus-ring ui-pressable flex w-full items-center justify-between rounded-button border border-transparent px-2 py-2 text-left text-sm"
                :class="[active ? 'ui-state-hovered' : '']"
                @click="onSwitchAccount(account.id, close)"
              >
                <span class="min-w-0">
                  <span class="block truncate text-sm font-medium text-ui-fg">{{ account.nickname }}</span>
                  <span class="block truncate text-xs text-ui-muted">{{ account.tenantId }} · {{ account.userId }}</span>
                </span>
                <Icon
                  v-if="account.id === activeAccountID"
                  name="check"
                  :size="14"
                  decorative
                  class="ui-tone-text-primary"
                />
              </button>
            </MenuItem>
          </section>

          <section class="mt-2 border-t border-ui-borderSubtle pt-2">
            <MenuItem as="template" v-slot="{ active }">
              <button
                type="button"
                class="ui-focus-ring ui-pressable flex w-full items-center gap-2 rounded-button border border-transparent px-2 py-2 text-left text-sm"
                :class="[active ? 'ui-state-hovered' : '']"
                @click="onOpenSettings(close)"
              >
                <Icon name="settings" :size="14" decorative class="text-ui-muted" />
                <span>{{ t('common.settingsAction') }}</span>
              </button>
            </MenuItem>

            <MenuItem as="template" v-slot="{ active }">
              <button
                type="button"
                class="ui-focus-ring ui-pressable mt-1 flex w-full items-center gap-2 rounded-button border border-transparent px-2 py-2 text-left text-sm"
                :class="[active ? 'ui-state-hovered' : '']"
                @click="onOpenAddAccount(close)"
              >
                <Icon name="plus" :size="14" decorative class="text-ui-muted" />
                <span>{{ t('common.addAnotherAccount') }}</span>
              </button>
            </MenuItem>

            <MenuItem as="template" v-slot="{ active }">
              <button
                type="button"
                class="ui-focus-ring ui-pressable mt-1 flex w-full items-center gap-2 rounded-button border border-transparent px-2 py-2 text-left text-sm ui-text-danger"
                :class="[active ? 'ui-state-hovered' : '']"
                @click="onSignOut(close)"
              >
                <Icon name="logout" :size="14" decorative />
                <span>{{ t('common.signOut') }}</span>
              </button>
            </MenuItem>
          </section>
        </MenuItems>
      </transition>
    </Menu>

    <Dialog
      :open="addDialogOpen"
      :title="t('common.addAnotherAccount')"
      :description="t('common.addAccountHint')"
      :show-footer="false"
      @close="addDialogOpen = false"
    >
      <form class="ui-page gap-3" @submit.prevent="onSubmitAccount">
        <label class="ui-page gap-1 text-xs text-ui-muted">
          <span>{{ t('common.nickname') }}</span>
          <Input v-model="draft.nickname" />
        </label>

        <label class="ui-page gap-1 text-xs text-ui-muted">
          <span>{{ t('common.tenantId') }}</span>
          <Input v-model="draft.tenantId" />
        </label>

        <label class="ui-page gap-1 text-xs text-ui-muted">
          <span>{{ t('common.userId') }}</span>
          <Input v-model="draft.userId" />
        </label>

        <label class="ui-page gap-1 text-xs text-ui-muted">
          <span>{{ t('common.roles') }}</span>
          <Input v-model="draft.roles" />
        </label>

        <label class="ui-page gap-1 text-xs text-ui-muted">
          <span>{{ t('common.policyVersion') }}</span>
          <Input v-model="draft.policyVersion" />
        </label>

        <label class="ui-page gap-1 text-xs text-ui-muted">
          <span>{{ t('common.defaultWorkspaceId') }}</span>
          <Input v-model="draft.workspaceId" />
        </label>

        <label class="ui-page gap-1 text-xs text-ui-muted">
          <span>{{ t('common.defaultWorkspaceName') }}</span>
          <Input v-model="draft.workspaceName" />
        </label>

        <div class="mt-1 flex items-center justify-end gap-2">
          <Button variant="ghost" type="button" @click="addDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button type="submit" :disabled="submitDisabled">{{ t('common.createAccount') }}</Button>
        </div>
      </form>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Render workspace and account menu with local account switching capabilities.
 */
import Button from '@/components/ui/Button.vue'
import Dialog from '@/components/ui/Dialog.vue'
import Icon from '@/components/ui/Icon.vue'
import Input from '@/components/ui/Input.vue'
import type { IdentityAccountDraft } from '@/design-system/identity'
import { useIdentityStore } from '@/design-system/identity'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const props = withDefaults(
  defineProps<{
    collapsed?: boolean
  }>(),
  {
    collapsed: false,
  },
)

const { t } = useI18n({ useScope: 'global' })
const router = useRouter()
const { accounts, activeAccount, activeAccountId, activeWorkspace, switchAccount, switchWorkspace, addAccount, signOutCurrentAccount } =
  useIdentityStore()

const addDialogOpen = ref(false)

const draft = reactive<IdentityAccountDraft>({
  nickname: '',
  userId: '',
  tenantId: '',
  roles: 'member',
  policyVersion: 'v0.1',
  workspaceId: '',
  workspaceName: '',
})

const triggerClasses = computed(() => [
  'ui-control ui-focus-ring ui-pressable inline-flex min-h-0 w-full items-center gap-2 rounded-button border-ui-border px-2 py-2 text-left',
  props.collapsed ? 'justify-center px-1' : '',
])

const accountDisplayName = computed(() => activeAccount.value?.nickname ?? 'User')

const workspaceDisplayName = computed(() => activeWorkspace.value?.name ?? t('common.workspace'))

const avatarText = computed(() => {
  const source = accountDisplayName.value.trim()
  if (!source) {
    return 'U'
  }
  return source.slice(0, 1).toUpperCase()
})

const activeAccountID = computed(() => activeAccountId.value)
const activeWorkspaceID = computed(() => activeWorkspace.value?.id ?? '')
const activeAccountWorkspaces = computed(() => activeAccount.value?.workspaces ?? [])

const submitDisabled = computed(() => {
  return (
    draft.nickname.trim().length === 0 ||
    draft.tenantId.trim().length === 0 ||
    draft.userId.trim().length === 0 ||
    draft.workspaceId.trim().length === 0
  )
})

function resetDraft(): void {
  draft.nickname = ''
  draft.userId = ''
  draft.tenantId = ''
  draft.roles = 'member'
  draft.policyVersion = 'v0.1'
  draft.workspaceId = ''
  draft.workspaceName = ''
}

function onSwitchWorkspace(workspaceId: string, close: () => void): void {
  switchWorkspace(workspaceId)
  close()
}

function onSwitchAccount(accountId: string, close: () => void): void {
  switchAccount(accountId)
  close()
}

function onOpenSettings(close: () => void): void {
  close()
  void router.push('/settings')
}

function onOpenAddAccount(close: () => void): void {
  close()
  resetDraft()

  const currentAccount = activeAccount.value
  if (currentAccount) {
    draft.tenantId = currentAccount.tenantId
    draft.roles = currentAccount.roles
    draft.policyVersion = currentAccount.policyVersion
  }

  addDialogOpen.value = true
}

function onSubmitAccount(): void {
  if (submitDisabled.value) {
    return
  }

  addAccount({
    nickname: draft.nickname,
    userId: draft.userId,
    tenantId: draft.tenantId,
    roles: draft.roles,
    policyVersion: draft.policyVersion,
    workspaceId: draft.workspaceId,
    workspaceName: draft.workspaceName,
  })

  addDialogOpen.value = false
}

function onSignOut(close: () => void): void {
  close()
  signOutCurrentAccount()
}
</script>
