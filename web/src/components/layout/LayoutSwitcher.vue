<template>
  <label class="flex items-center gap-2 text-xs text-ui-muted">
    <span>{{ t('common.layout') }}</span>
    <select v-model="layoutModel" class="ui-control ui-focus-ring ui-pressable min-w-32 bg-ui-panel text-sm">
      <option value="auto">{{ t('common.layoutAuto') }}</option>
      <option value="console">{{ t('common.layoutConsole') }}</option>
      <option value="topnav">{{ t('common.layoutTopnav') }}</option>
      <option value="focus">{{ t('common.layoutFocus') }}</option>
    </select>
  </label>
</template>

<script setup lang="ts">
import { useLayoutStore } from '@/design-system/layout'
import type { LayoutPreference } from '@/design-system/types'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { layoutPreference, setLayoutPreference } = useLayoutStore()

const layoutModel = computed<LayoutPreference>({
  get: () => layoutPreference.value,
  set: (value) => {
    setLayoutPreference(value)
  },
})
</script>
