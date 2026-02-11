<template>
  <section class="ui-page">
    <PageHeader :title="t('page.permissionManagement.title')" :subtitle="t('page.permissionManagement.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isRefreshing" @click="onRefresh">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <WindowBoard route-key="permission-management" :panes="windowPanes">
      <template #permission-overview>
        <SectionCard :title="t('page.permissionManagement.overviewTitle')" :subtitle="String(rows.length)">
          <div class="mb-3 rounded-lg border border-ui-border bg-ui-panel p-3 text-xs text-ui-muted">
            {{ t('page.permissionManagement.overviewHint') }}
          </div>
          <Table
            :columns="columns"
            :rows="rows"
            :state="tableState"
            :error-message="tableErrorMessage"
            :caption="t('page.permissionManagement.overviewTitle')"
            interactive-rows
            row-key="id"
            :selected-row-key="selectedCommandId"
            @row-click="onRowClick"
          />
        </SectionCard>
      </template>

      <template #permission-logs>
        <SectionCard :title="t('page.permissionManagement.detailTitle')" :subtitle="selectedCommand?.id ?? '-'">
          <div v-if="selectedCommand" class="space-y-3">
            <div class="rounded-lg border border-ui-border bg-ui-panel p-3 text-xs text-ui-muted">
              <div>{{ t('page.commands.fieldType') }}: {{ selectedCommand.commandType }}</div>
              <div>{{ t('page.commands.fieldStatus') }}: {{ selectedCommand.status }}</div>
              <div>{{ t('page.commands.fieldAcceptedAt') }}: {{ selectedCommand.acceptedAt }}</div>
              <div>{{ t('page.commands.fieldOwner') }}: {{ selectedCommand.ownerId }}</div>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3 text-xs">
              <div class="mb-2 font-medium text-ui-text">payload</div>
              <pre class="ui-scrollbar max-h-52 overflow-auto rounded border border-ui-border bg-ui-bg p-2">{{
                formatJSON(selectedCommand.payload)
              }}</pre>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3 text-xs">
              <div class="mb-2 font-medium text-ui-text">result</div>
              <pre class="ui-scrollbar max-h-52 overflow-auto rounded border border-ui-border bg-ui-bg p-2">{{
                formatJSON(selectedCommand.result || {})
              }}</pre>
            </div>
          </div>

          <div v-else class="text-sm text-ui-muted">{{ t('page.permissionManagement.emptyDetail') }}</div>
        </SectionCard>
      </template>
    </WindowBoard>
  </section>
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
import { listCommands } from '@/api/commands'
import { ApiHttpError } from '@/api/http'
import type { ApiError, CommandDTO } from '@/api/types'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import type { TableState } from '@/design-system/types'
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })

const commands = ref<CommandDTO[]>([])
const selectedCommandId = ref<string | null>(null)
const tableState = ref<TableState>('loading')
const apiError = ref<ApiError | null>(null)

const windowPanes = computed(() => [
  { id: 'permission-overview', title: t('page.permissionManagement.overviewTitle') },
  { id: 'permission-logs', title: t('page.permissionManagement.detailTitle') },
])

const columns = computed<TableColumn[]>(() => [
  { key: 'commandType', label: t('page.commands.fieldType') },
  { key: 'status', label: t('page.commands.fieldStatus'), width: '9rem' },
  { key: 'ownerId', label: t('page.commands.fieldOwner'), width: '11rem' },
  { key: 'id', label: t('page.commands.fieldCommandId'), mono: true, width: '17rem' },
])

const rows = computed(() =>
  commands.value.map((item) => ({
    id: item.id,
    commandType: item.commandType,
    status: item.status,
    ownerId: item.ownerId,
  })),
)

const selectedCommand = computed(
  () => commands.value.find((item) => item.id === selectedCommandId.value) ?? null,
)

const tableErrorMessage = computed(() => {
  if (!apiError.value) {
    return t('error.common.internal')
  }
  return t(apiError.value.messageKey || 'error.common.internal', apiError.value.details ?? {})
})

const isRefreshing = computed(() => tableState.value === 'loading')

watch(
  commands,
  (items) => {
    if (!items.some((item) => item.id === selectedCommandId.value)) {
      selectedCommandId.value = items[0]?.id ?? null
    }
  },
  { immediate: true },
)

onMounted(() => {
  void loadCommands()
})

async function loadCommands(): Promise<void> {
  tableState.value = 'loading'
  apiError.value = null
  try {
    const response = await listCommands({ page: 1, pageSize: 400 })
    commands.value = response.items.filter((item) => item.commandType.startsWith('share.'))
    tableState.value = 'ready'
  } catch (error) {
    commands.value = []
    apiError.value = asApiError(error)
    tableState.value = 'error'
  }
}

function onRowClick(payload: { rowKey: string }): void {
  selectedCommandId.value = payload.rowKey
}

async function onRefresh(): Promise<void> {
  await loadCommands()
}

function asApiError(error: unknown): ApiError {
  if (error instanceof ApiHttpError) {
    return error.error
  }
  return {
    code: 'INTERNAL_ERROR',
    messageKey: 'error.common.internal',
  }
}

function formatJSON(value: unknown): string {
  try {
    return JSON.stringify(value ?? {}, null, 2)
  } catch {
    return '{}'
  }
}
</script>
