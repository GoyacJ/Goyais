<template>
  <header class="ui-surface flex flex-wrap items-center justify-between gap-3 rounded-none border-x-0 border-t-0 px-4 py-3">
    <div class="min-w-0">
      <p class="ui-monospace text-[11px] uppercase tracking-[0.15em] text-ui-muted">Console-first</p>
      <h1 class="truncate text-lg font-semibold">{{ t('common.appName') }}</h1>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <button type="button" class="ui-control ui-focus-ring ui-pressable text-sm text-ui-muted">
        {{ t('common.searchPlaceholder') }}
      </button>

      <label class="flex items-center gap-2 text-xs text-ui-muted">
        <span>{{ t('common.theme') }}</span>
        <select v-model="themeModel" class="ui-control ui-focus-ring ui-pressable min-w-28 bg-ui-panel text-sm">
          <option value="system">{{ t('common.system') }}</option>
          <option value="light">{{ t('common.light') }}</option>
          <option value="dark">{{ t('common.dark') }}</option>
        </select>
      </label>

      <label class="flex items-center gap-2 text-xs text-ui-muted">
        <span>{{ t('common.language') }}</span>
        <select v-model="localeModel" class="ui-control ui-focus-ring ui-pressable min-w-24 bg-ui-panel text-sm">
          <option value="zh-CN">zh-CN</option>
          <option value="en-US">en-US</option>
        </select>
      </label>
    </div>
  </header>
</template>

<script setup lang="ts">
import type { SupportedLocale, ThemeMode } from '@/design-system/types'
import { useThemeStore } from '@/design-system/theme'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n({ useScope: 'global' })
const { themeMode, setThemeMode } = useThemeStore()

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
</script>
