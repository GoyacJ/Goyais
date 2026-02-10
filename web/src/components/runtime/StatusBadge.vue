<template>
  <Badge :tone="tone">
    {{ t(`status.${status}`) }}
  </Badge>
</template>

<script setup lang="ts">
import Badge from '@/components/ui/Badge.vue'
import type { CommandStatus } from '@/design-system/types'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  status: CommandStatus
}>()

const { t } = useI18n({ useScope: 'global' })

const tone = computed(() => {
  const map: Record<CommandStatus, 'primary' | 'success' | 'warn' | 'error' | 'neutral'> = {
    accepted: 'primary',
    running: 'warn',
    succeeded: 'success',
    failed: 'error',
    canceled: 'neutral',
  }

  return map[props.status]
})
</script>
