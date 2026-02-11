<template>
  <aside
    class="ui-surface hidden h-full flex-col rounded-none border-y-0 border-l-0 transition-[width] duration-150 lg:flex"
    :class="collapsed ? 'w-[4.75rem]' : 'w-64'"
    @mouseenter="hovering = true"
    @mouseleave="hovering = false"
  >
    <header
      class="flex h-[3.25rem] shrink-0 items-center border-b border-ui-border py-2"
      :class="collapsed ? 'gap-1 px-1' : 'gap-2 px-3'"
      data-testid="sidenav-workspace"
    >
      <div :class="collapsed ? 'shrink-0' : 'min-w-0 flex-1'">
        <WorkspaceSwitcherMenu :collapsed="collapsed" />
      </div>
      <button
        type="button"
        class="ui-focus-ring ui-pressable inline-flex min-h-0 shrink-0 items-center justify-center rounded-button border border-transparent bg-transparent p-0"
        :class="collapsed ? 'h-7 w-7' : 'h-8 w-8'"
        :aria-label="pinned ? t('common.unpinNav') : t('common.pinNav')"
        :data-pinned="pinned ? 'true' : 'false'"
        @click="onTogglePinned"
      >
        <Icon :name="pinned ? 'sidebar-collapse' : 'sidebar-expand'" :size="14" decorative />
      </button>
    </header>

    <nav class="ui-page min-h-0 flex-1 overflow-auto p-3" data-testid="sidenav-nav">
      <RouterLink
        v-for="item in NAV_ITEMS"
        :key="item.to"
        :to="item.to"
        class="ui-nav-link ui-focus-ring ui-pressable flex items-center text-sm font-medium"
        :class="collapsed ? 'justify-center px-1' : 'justify-between'"
        :title="collapsed ? t(item.label) : undefined"
        active-class="ui-nav-link-active"
      >
        <span class="flex min-w-0 items-center gap-2">
          <Icon :name="item.icon" :size="16" decorative class="opacity-90" />
          <span v-if="!collapsed" class="truncate">{{ t(item.label) }}</span>
        </span>
      </RouterLink>
    </nav>

    <footer class="shrink-0 border-t border-ui-border px-3 py-2" data-testid="sidenav-user">
      <UserAccountMenu :collapsed="collapsed" />
    </footer>
  </aside>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Render desktop side navigation with pinned-floating and three-zone layout.
 */
import Icon from '@/components/ui/Icon.vue'
import WorkspaceSwitcherMenu from '@/components/layout/WorkspaceSwitcherMenu.vue'
import UserAccountMenu from '@/components/layout/UserAccountMenu.vue'
import { NAV_ITEMS } from '@/design-system/navigation'
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const { t } = useI18n({ useScope: 'global' })

const PINNED_STORAGE_KEY = 'goyais.ui.sidenav.pinned'

function readPinnedState(): boolean {
  try {
    return localStorage.getItem(PINNED_STORAGE_KEY) === 'true'
  } catch {
    return false
  }
}

function persistPinnedState(value: boolean): void {
  try {
    localStorage.setItem(PINNED_STORAGE_KEY, value ? 'true' : 'false')
  } catch {
    // Ignore storage failures (private mode / quota).
  }
}

const pinned = ref(false)
const hovering = ref(false)

const collapsed = computed(() => !pinned.value && !hovering.value)

onMounted(() => {
  pinned.value = readPinnedState()
})

watch(pinned, (value) => {
  persistPinnedState(value)
})

function onTogglePinned(): void {
  const nextPinned = !pinned.value
  pinned.value = nextPinned

  // When switching to floating mode, collapse immediately even if cursor stays inside sidenav.
  if (!nextPinned) {
    hovering.value = false
  }
}
</script>
