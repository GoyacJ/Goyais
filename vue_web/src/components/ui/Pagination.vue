<template>
  <div class="flex items-center justify-between gap-2">
    <p class="ui-monospace text-xs text-ui-muted">{{ page }} / {{ totalPages }}</p>

    <div class="flex items-center gap-2">
      <Button :disabled="page <= 1" variant="ghost" @click="emit('update:page', page - 1)">
        {{ t('common.previous') }}
      </Button>
      <Button :disabled="page >= totalPages" variant="ghost" @click="emit('update:page', page + 1)">
        {{ t('common.next') }}
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Button from '@/components/ui/Button.vue'

const props = defineProps<{
  page: number
  pageSize: number
  total: number
}>()

const emit = defineEmits<{
  (e: 'update:page', value: number): void
}>()

const { t } = useI18n({ useScope: 'global' })

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
</script>
