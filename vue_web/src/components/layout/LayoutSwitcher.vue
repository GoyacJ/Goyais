<template>
  <label class="flex items-center gap-2 text-xs text-ui-muted">
    <span>{{ t('common.layout') }}</span>
    <div class="min-w-36">
      <Select v-model="layoutModel" :options="layoutOptions" />
    </div>
  </label>
</template>

<script setup lang="ts">
import Select from '@/components/ui/Select.vue'
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

const layoutOptions = computed(() => [
  { value: 'auto', label: t('common.layoutAuto') },
  { value: 'console', label: t('common.layoutConsole') },
  { value: 'topnav', label: t('common.layoutTopnav') },
  { value: 'focus', label: t('common.layoutFocus') },
])
</script>
