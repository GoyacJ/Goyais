<template>
  <div class="space-y-2">
    <EmptyState
      v-if="events.length === 0"
      variant="commands-empty"
      :title="t('page.streams.logsEmptyTitle')"
      :description="t('page.streams.logsEmptyDescription')"
    />
    <article v-for="item in events" :key="item.commandId" class="ui-surface p-3">
      <header class="flex items-center justify-between gap-2">
        <p class="ui-monospace text-xs text-ui-muted">{{ item.commandId }}</p>
        <StatusBadge :status="item.status" />
      </header>
      <p class="mt-2 text-sm text-ui-fg">{{ item.commandType }}</p>
      <p class="mt-1 text-xs text-ui-muted">{{ item.acceptedAt }}</p>
      <p class="mt-2 text-sm text-ui-muted">{{ item.summary }}</p>
    </article>
  </div>
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
import EmptyState from '@/components/layout/EmptyState.vue'
import StatusBadge from '@/components/runtime/StatusBadge.vue'
import type { CommandStatus } from '@/design-system/types'
import { useI18n } from 'vue-i18n'

export interface StreamCommandLogEvent {
  commandId: string
  commandType: string
  acceptedAt: string
  status: CommandStatus
  summary: string
}

defineProps<{
  events: StreamCommandLogEvent[]
}>()

const { t } = useI18n({ useScope: 'global' })
</script>
