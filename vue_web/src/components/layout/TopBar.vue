<template>
  <header class="ui-topbar ui-surface flex flex-wrap items-center justify-between gap-3 rounded-none border-x-0 border-t-0 px-4 py-3">
    <div class="ui-topbar-brand flex min-w-0 items-center gap-2">
      <button
        v-if="showMobileNavButton"
        type="button"
        class="ui-control ui-focus-ring ui-pressable inline-flex h-8 min-h-0 items-center gap-2 px-2 py-1 text-xs"
        :aria-label="t('common.openNavigation')"
        @click="emit('toggleMobileNav')"
      >
        <Icon name="canvas" :size="14" decorative />
        <span>{{ t('common.navShort') }}</span>
      </button>

      <div class="min-w-0">
        <p class="ui-monospace text-[11px] uppercase tracking-[0.15em] text-ui-muted">{{ t('common.consoleFirst') }}</p>
        <h1 class="truncate text-lg font-semibold">{{ t('common.appName') }}</h1>
      </div>
    </div>

    <div class="ui-topbar-controls flex min-w-0 flex-wrap items-center gap-2">
      <span v-if="focusMode" class="ui-control ui-tone-surface-primary ui-monospace inline-flex h-8 min-h-0 items-center px-2 py-0 text-xs leading-none">
        {{ t('common.layoutFocus') }}
      </span>

      <span class="ui-control ui-monospace inline-flex h-8 min-h-0 items-center bg-ui-surface2 px-2 py-0 text-xs leading-none text-ui-muted">
        {{ t('common.workspace') }}
      </span>

      <button type="button" class="ui-control ui-focus-ring ui-pressable inline-flex items-center gap-2 text-sm text-ui-fgSubtle">
        <Icon name="search" :size="14" decorative />
        <span>{{ t('common.searchPlaceholder') }}</span>
      </button>

      <LayoutSwitcher />

      <label class="flex items-center gap-2 text-xs text-ui-muted">
        <span>{{ t('common.theme') }}</span>
        <div class="min-w-28">
          <Select v-model="themeModel" :options="themeOptions" />
        </div>
      </label>

      <label class="flex items-center gap-2 text-xs text-ui-muted">
        <span>{{ t('common.language') }}</span>
        <div class="min-w-28">
          <Select v-model="localeModel" :options="localeOptions" />
        </div>
      </label>

      <Dropdown :label="t('common.userMenu')" :items="userMenuItems" @select="onUserMenuAction" />
    </div>
  </header>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */
import Dropdown, { type DropdownItem } from '@/components/ui/Dropdown.vue'
import Icon from '@/components/ui/Icon.vue'
import LayoutSwitcher from '@/components/layout/LayoutSwitcher.vue'
import Select from '@/components/ui/Select.vue'
import { useToast } from '@/composables/useToast'
import { useThemeStore } from '@/design-system/theme'
import type { SupportedLocale, ThemeMode } from '@/design-system/types'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

withDefaults(
  defineProps<{
    showMobileNavButton?: boolean
    focusMode?: boolean
  }>(),
  {
    showMobileNavButton: false,
    focusMode: false,
  },
)

const emit = defineEmits<{
  (e: 'toggleMobileNav'): void
}>()

const { t, locale } = useI18n({ useScope: 'global' })
const { themeMode, setThemeMode } = useThemeStore()
const { pushToast } = useToast()

const userMenuItems = computed<DropdownItem[]>(() => [
  { label: t('common.profile'), value: 'profile', hint: 'P' },
  { label: t('common.preferences'), value: 'preferences', hint: 'S' },
  { label: t('common.signOut'), value: 'signOut', danger: true },
])

const themeModel = computed<ThemeMode>({
  get: () => themeMode.value,
  set: (value) => {
    setThemeMode(value)
  },
})

const localeModel = computed<SupportedLocale>({
  get: () => locale.value as SupportedLocale,
  set: (value) => {
    locale.value = value
  },
})

const themeOptions = computed(() => [
  { value: 'system', label: t('common.system') },
  { value: 'light', label: t('common.light') },
  { value: 'dark', label: t('common.dark') },
])

const localeOptions = computed(() => [
  { value: 'zh-CN', label: t('common.localeZhCN') },
  { value: 'en-US', label: t('common.localeEnUS') },
])

function onUserMenuAction(value: string): void {
  pushToast({
    title: t('common.userMenu'),
    message: t('common.placeholderAction', { value }),
    tone: 'info',
  })
}
</script>
