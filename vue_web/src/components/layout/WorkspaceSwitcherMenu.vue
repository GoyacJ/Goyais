<template>
  <div class="relative">
    <Menu as="div" class="relative block" v-slot="{ open, close }">
      <MenuButton :class="triggerClasses" :aria-label="t('common.openWorkspaceMenu')">
        <span class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-ui-border bg-ui-surface2">
          <Icon name="canvas" :size="14" decorative class="text-ui-muted" />
        </span>

        <span v-if="!collapsed" class="min-w-0 flex-1 text-left">
          <p class="truncate text-[11px] uppercase tracking-[0.12em] text-ui-muted">{{ t('common.workspaceLabel') }}</p>
          <p class="truncate text-sm font-semibold text-ui-fg">{{ workspaceDisplayName }}</p>
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
        <MenuItems v-if="open" class="ui-overlay-panel absolute left-0 z-40 mt-2 w-[18rem] origin-top-left p-2">
          <section>
            <p class="px-2 pb-1 text-[11px] uppercase tracking-[0.12em] text-ui-muted">{{ t('common.workspaceList') }}</p>
            <MenuItem v-for="workspace in workspaces" :key="workspace.id" as="template" v-slot="{ active }">
              <button
                type="button"
                class="ui-focus-ring ui-pressable flex w-full items-center justify-between rounded-button border border-transparent px-2 py-2 text-left text-sm"
                :class="active ? 'ui-state-hovered' : ''"
                @click="onSwitchWorkspace(workspace.id, close)"
              >
                <span class="truncate">{{ workspace.name }}</span>
                <Icon
                  v-if="workspace.id === activeWorkspaceId"
                  name="check"
                  :size="14"
                  decorative
                  class="ui-tone-text-primary"
                />
              </button>
            </MenuItem>
          </section>
        </MenuItems>
      </transition>
    </Menu>
  </div>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Render workspace-only switcher for the sidebar top area.
 */
import Icon from '@/components/ui/Icon.vue'
import { useIdentityStore } from '@/design-system/identity'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    collapsed?: boolean
  }>(),
  {
    collapsed: false,
  },
)

const { t } = useI18n({ useScope: 'global' })
const { activeWorkspace, activeAccount, switchWorkspace } = useIdentityStore()

const triggerClasses = computed(() => [
  'ui-control ui-focus-ring ui-pressable inline-flex min-h-0 w-full items-center gap-2 rounded-button border-ui-border px-2 py-2 text-left',
  props.collapsed ? 'justify-center px-1' : '',
])

const workspaceDisplayName = computed(() => activeWorkspace.value?.name ?? t('common.workspace'))
const activeWorkspaceId = computed(() => activeWorkspace.value?.id ?? '')
const workspaces = computed(() => activeAccount.value?.workspaces ?? [])

function onSwitchWorkspace(workspaceId: string, close: () => void): void {
  switchWorkspace(workspaceId)
  close()
}
</script>
