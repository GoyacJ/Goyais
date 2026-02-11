<template>
  <header class="ui-surface flex items-center gap-2 rounded-none border-x-0 border-t-0 px-3 py-2">
    <div class="ui-scrollbar flex min-w-0 flex-1 items-center gap-1 overflow-x-auto">
      <div
        v-for="tab in resolvedTabs"
        :key="tab.to"
        class="group inline-flex h-9 shrink-0 items-center rounded-button border border-transparent"
        :class="tab.to === activeTabPath ? 'ui-state-selected' : ''"
      >
        <button
          type="button"
          class="ui-focus-ring ui-pressable inline-flex h-full items-center gap-2 rounded-button border border-transparent px-3 text-sm"
          :class="tab.to === activeTabPath ? 'border-transparent' : ''"
          @click="navigateTab(tab.to)"
        >
          <Icon :name="tab.icon" :size="14" decorative class="text-ui-muted" />
          <span>{{ tab.label }}</span>
        </button>

        <button
          type="button"
          class="ui-focus-ring ui-pressable mr-1 inline-flex h-7 w-7 items-center justify-center rounded-button border border-transparent text-ui-muted"
          :aria-label="t('common.closeTab')"
          :disabled="tabs.length === 1"
          @click="closeTab(tab.to)"
        >
          <Icon name="close" :size="12" decorative />
        </button>
      </div>
    </div>

    <Menu as="div" class="relative inline-block" v-slot="{ open, close }">
      <MenuButton
        class="ui-control ui-focus-ring ui-pressable inline-flex h-9 w-9 min-h-0 items-center justify-center p-0"
        :aria-label="t('common.openNewTabMenu')"
      >
        <Icon name="plus" :size="14" decorative class="transition-transform duration-120" :class="open ? 'rotate-90' : ''" />
      </MenuButton>

      <transition
        enter-active-class="transition duration-120 ease-out"
        enter-from-class="scale-95 opacity-0"
        enter-to-class="scale-100 opacity-100"
        leave-active-class="transition duration-90 ease-in"
        leave-from-class="scale-100 opacity-100"
        leave-to-class="scale-95 opacity-0"
      >
        <MenuItems v-if="open" class="ui-overlay-panel absolute right-0 z-40 mt-1 w-56 origin-top-right p-1">
          <MenuItem v-for="item in availableNavItems" :key="item.to" as="template" v-slot="{ active }">
            <button
              type="button"
              class="ui-focus-ring ui-pressable flex w-full items-center gap-2 rounded-button border border-transparent px-2 py-2 text-sm"
              :class="active ? 'ui-state-hovered' : ''"
              @click="onOpenTab(item.to, close)"
            >
              <Icon :name="item.icon" :size="14" decorative class="text-ui-muted" />
              <span>{{ t(item.label) }}</span>
            </button>
          </MenuItem>
          <p v-if="availableNavItems.length === 0" class="px-2 py-2 text-xs text-ui-muted">{{ t('common.noMoreTabs') }}</p>
        </MenuItems>
      </transition>
    </Menu>
  </header>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Render dynamic route tabs with close and add actions.
 */
import Icon from '@/components/ui/Icon.vue'
import { NAV_ITEMS } from '@/design-system/navigation'
import { useRouteTabsStore } from '@/design-system/route-tabs'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { tabs, activeTabPath, availableNavItems, navigateTab, openTab, closeTab } = useRouteTabsStore()

const resolvedTabs = computed(() =>
  tabs.value.map((path) => {
    const matched = NAV_ITEMS.find((item) => item.to === path)
    if (matched) {
      return {
        to: matched.to,
        icon: matched.icon,
        label: t(matched.label),
      }
    }

    return {
      to: path,
      icon: 'home' as const,
      label: path,
    }
  }),
)

function onOpenTab(path: string, close: () => void): void {
  openTab(path)
  close()
}
</script>
