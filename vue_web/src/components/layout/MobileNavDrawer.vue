<template>
  <transition
    enter-active-class="transition-opacity duration-120"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition-opacity duration-90"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div
      v-if="open"
      class="ui-overlay-backdrop fixed inset-0 z-40 lg:hidden"
      role="presentation"
      @click="emitClose"
    />
  </transition>

  <transition
    enter-active-class="transition-transform duration-150 ease-out"
    enter-from-class="-translate-x-full"
    enter-to-class="translate-x-0"
    leave-active-class="transition-transform duration-120 ease-in"
    leave-from-class="translate-x-0"
    leave-to-class="-translate-x-full"
  >
    <aside
      v-if="open"
      ref="drawerRef"
      class="ui-surface fixed inset-y-0 left-0 z-50 flex w-72 flex-col rounded-none border-y-0 border-l-0 lg:hidden"
      :aria-label="t('common.openNavigation')"
      tabindex="-1"
      @keydown.esc.prevent.stop="emitClose"
    >
      <header class="flex items-center justify-between border-b border-ui-border px-4 py-3">
        <p class="text-sm font-semibold">{{ t('common.appName') }}</p>
        <button
          type="button"
          class="ui-control ui-focus-ring ui-pressable h-8 min-h-0 px-2 py-1 text-xs"
          @click="emitClose"
        >
          {{ t('common.close') }}
        </button>
      </header>

      <section class="border-b border-ui-border px-3 py-3">
        <WorkspaceSwitcherMenu />
      </section>

      <nav class="ui-page min-h-0 flex-1 overflow-auto p-3">
        <RouterLink
          v-for="item in NAV_ITEMS"
          :key="item.to"
          :to="item.to"
          class="ui-nav-link ui-focus-ring ui-pressable flex items-center justify-between text-sm"
          active-class="ui-nav-link-active"
          @click="emitClose"
        >
          <span class="flex min-w-0 items-center gap-2">
            <Icon :name="item.icon" :size="16" decorative />
            <span class="truncate">{{ t(item.label) }}</span>
          </span>
        </RouterLink>
      </nav>

      <section class="mt-auto border-t border-ui-border px-3 py-3">
        <UserAccountMenu />
      </section>
    </aside>
  </transition>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Render mobile side drawer with workspace-nav-user three-zone layout.
 */
import Icon from '@/components/ui/Icon.vue'
import UserAccountMenu from '@/components/layout/UserAccountMenu.vue'
import WorkspaceSwitcherMenu from '@/components/layout/WorkspaceSwitcherMenu.vue'
import { NAV_ITEMS } from '@/design-system/navigation'
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n({ useScope: 'global' })
const drawerRef = ref<HTMLElement | null>(null)

function emitClose(): void {
  emit('close')
}

function focusDrawerEntry(): void {
  const root = drawerRef.value
  if (!root) {
    return
  }

  const firstFocusable = root.querySelector<HTMLElement>(
    'button:not([disabled]), a[href], [tabindex]:not([tabindex="-1"]), input, select, textarea',
  )
  firstFocusable?.focus()
}

watch(
  () => props.open,
  async (value) => {
    if (!value) {
      return
    }
    await nextTick()
    focusDrawerEntry()
  },
  { flush: 'post' },
)
</script>
