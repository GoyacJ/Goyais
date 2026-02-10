<template>
  <nav class="ui-surface ui-bg-content flex items-center gap-2 overflow-auto border-x-0 border-t-0 px-4 py-2">
    <RouterLink
      v-for="item in NAV_ITEMS"
      :key="item.to"
      :to="item.to"
      class="ui-control ui-focus-ring ui-pressable inline-flex shrink-0 items-center gap-2 border-transparent text-sm"
      active-class="!border-primary-500 !bg-primary-500/10 !text-primary-700 dark:!text-primary-500"
    >
      <Icon :name="item.icon" :size="16" decorative />
      <span>{{ t(item.label) }}</span>
    </RouterLink>
  </nav>
</template>

<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'
import { NAV_ITEMS } from '@/design-system/navigation'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const { t } = useI18n({ useScope: 'global' })
</script>
