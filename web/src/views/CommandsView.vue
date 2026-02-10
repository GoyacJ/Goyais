<template>
  <section class="ui-page">
    <PageHeader :title="t('page.commands.title')" :subtitle="t('page.commands.subtitle')">
      <template #actions>
        <Button variant="secondary">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
        <Button>
          <Icon name="commands" :size="14" decorative />
          {{ t('page.commands.actionRun') }}
        </Button>
      </template>
    </PageHeader>

    <WindowBoard route-key="commands" :panes="windowPanes">
      <template #filters>
        <SectionCard :title="t('page.commands.filtersTitle')" :subtitle="t('page.commands.filtersSubtitle')">
          <div class="grid gap-3 md:grid-cols-[1.4fr_1fr_1fr]">
            <Input v-model="searchQuery" :placeholder="t('page.commands.searchPlaceholder')" />
            <Select v-model="statusFilter" :options="statusOptions" />
            <Select v-model="ownerFilter" :options="ownerOptions" />
          </div>
        </SectionCard>
      </template>

      <template #list>
        <SectionCard :title="t('page.commands.listTitle')" :subtitle="`${filteredCommands.length}`">
          <div v-if="filteredCommands.length === 0">
            <EmptyState
              variant="commands-empty"
              :title="t('empty_state.commands.title')"
              :description="t('empty_state.commands.description')"
            />
          </div>
          <Table
            v-else
            :columns="columns"
            :rows="tableRows"
            :caption="t('page.commands.listTitle')"
            interactive-rows
            row-key="commandId"
            :selected-row-key="selectedCommandId"
            @row-click="onCommandRowClick"
          >
            <template #cell-status="{ row }">
              <StatusBadge :status="asCommandStatus(row.status)" />
            </template>
            <template #cell-commandId="{ row }">
              <div class="flex flex-col gap-0.5 leading-tight">
                <span class="ui-monospace text-[12px] text-ui-fg">{{ String(row.commandId) }}</span>
                <span class="text-[11px] text-ui-muted">{{ String(row.owner) }}</span>
              </div>
            </template>
          </Table>
        </SectionCard>
      </template>

      <template #detail>
        <SectionCard :title="t('page.commands.detailTitle')" :subtitle="selectedCommand?.commandId ?? '-'">
          <div v-if="selectedCommand" class="space-y-3">
            <Tabs v-model="detailTab" :tabs="detailTabs" :aria-label="t('page.commands.detailTitle')" />

            <div v-if="detailTab === 'summary'" class="ui-detail-block">
              <header class="ui-detail-header">
                <div class="min-w-0">
                  <p class="text-sm font-semibold text-ui-fg">{{ selectedCommand.commandType }}</p>
                  <p class="mt-1 text-xs text-ui-muted">{{ selectedCommand.resultSummary }}</p>
                </div>
                <StatusBadge :status="selectedCommand.status" />
              </header>

              <dl class="ui-detail-meta text-xs text-ui-muted md:grid-cols-2">
                <div>
                  <dt>{{ t('page.commands.fieldAcceptedAt') }}</dt>
                  <dd class="ui-monospace mt-1 text-ui-fg">{{ selectedCommand.acceptedAt }}</dd>
                </div>
                <div>
                  <dt>{{ t('page.commands.fieldOwner') }}</dt>
                  <dd class="ui-monospace mt-1 text-ui-fg">{{ selectedCommand.owner }}</dd>
                </div>
                <div class="md:col-span-2 ui-detail-mono">
                  <dt>{{ t('page.commands.fieldTraceId') }}</dt>
                  <dd class="ui-monospace mt-1 break-all text-ui-fg">{{ selectedCommand.traceId }}</dd>
                </div>
              </dl>
            </div>

            <LogPanel v-else :lines="selectedCommand.logs" />
          </div>
          <EmptyState
            v-else
            variant="commands-empty"
            :title="t('empty_state.commands.detailTitle')"
            :description="t('empty_state.commands.detailDescription')"
          />
        </SectionCard>
      </template>
    </WindowBoard>
  </section>
</template>

<script setup lang="ts">
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import StatusBadge from '@/components/runtime/StatusBadge.vue'
import LogPanel from '@/components/runtime/LogPanel.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import Tabs from '@/components/ui/Tabs.vue'
import type { CommandStatus } from '@/design-system/types'
import { mockCommands } from '@/mocks/commands'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })

const searchQuery = ref('')
const statusFilter = ref('all')
const ownerFilter = ref('all')
const selectedCommandId = ref<string | null>(mockCommands[0]?.commandId ?? null)
const detailTab = ref('summary')

const windowPanes = computed(() => [
  { id: 'filters', title: t('page.commands.filtersTitle') },
  { id: 'list', title: t('page.commands.listTitle') },
  { id: 'detail', title: t('page.commands.detailTitle') },
])

const statusOptions = computed(() => [
  { value: 'all', label: t('common.all') },
  { value: 'accepted', label: t('status.accepted') },
  { value: 'running', label: t('status.running') },
  { value: 'succeeded', label: t('status.succeeded') },
  { value: 'failed', label: t('status.failed') },
  { value: 'canceled', label: t('status.canceled') },
])

const ownerOptions = computed(() => [
  { value: 'all', label: t('common.all') },
  ...Array.from(new Set(mockCommands.map((item) => item.owner))).map((owner) => ({
    value: owner,
    label: owner,
  })),
])

const filteredCommands = computed(() =>
  mockCommands.filter((item) => {
    const matchStatus = statusFilter.value === 'all' || item.status === statusFilter.value
    const matchOwner = ownerFilter.value === 'all' || item.owner === ownerFilter.value
    const q = searchQuery.value.trim().toLowerCase()
    const matchQuery =
      q.length === 0 ||
      item.commandId.toLowerCase().includes(q) ||
      item.commandType.toLowerCase().includes(q) ||
      item.traceId.toLowerCase().includes(q)

    return matchStatus && matchOwner && matchQuery
  }),
)

watch(
  filteredCommands,
  (items) => {
    if (!items.some((item) => item.commandId === selectedCommandId.value)) {
      selectedCommandId.value = items[0]?.commandId ?? null
    }
  },
  { immediate: true },
)

const selectedCommand = computed(() =>
  filteredCommands.value.find((item) => item.commandId === selectedCommandId.value),
)

const columns = computed<TableColumn[]>(() => [
  { key: 'commandType', label: t('page.commands.fieldType') },
  { key: 'status', label: t('page.commands.fieldStatus'), align: 'center', width: '8.5rem' },
  { key: 'acceptedAt', label: t('page.commands.fieldAcceptedAt'), mono: true, width: '12rem' },
  { key: 'commandId', label: t('page.commands.fieldCommandId'), mono: true, width: '17rem' },
])

const tableRows = computed(() =>
  filteredCommands.value.map((item) => ({
    commandId: item.commandId,
    commandType: item.commandType,
    status: item.status,
    acceptedAt: item.acceptedAt,
    owner: item.owner,
  })),
)

const detailTabs = computed(() => [
  { id: 'summary', label: t('page.commands.tabSummary') },
  { id: 'logs', label: t('page.commands.tabLogs') },
])

function onCommandRowClick(payload: { rowKey: string }): void {
  selectedCommandId.value = payload.rowKey
}

function asCommandStatus(value: unknown): CommandStatus {
  if (value === 'running' || value === 'succeeded' || value === 'failed' || value === 'canceled') {
    return value
  }

  return 'accepted'
}
</script>
