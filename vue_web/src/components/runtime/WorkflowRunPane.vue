<template>
  <div class="space-y-3">
    <header class="flex flex-wrap items-center gap-2">
      <Select v-model="mode" :options="modeOptions" />
      <Button :disabled="busy || !selectedTemplateId" @click="onCreateRun">{{ t('page.canvas.actionRunTemplate') }}</Button>
      <Button variant="secondary" :disabled="busy || !selectedRunId" @click="$emit('cancel', selectedRunId as string)">
        {{ t('page.canvas.actionCancelRun') }}
      </Button>
      <Button variant="ghost" :disabled="busy" @click="$emit('refresh')">{{ t('common.refresh') }}</Button>
    </header>

    <Table
      :columns="columns"
      :rows="rows"
      :state="state"
      :caption="t('page.canvas.runsTitle')"
      interactive-rows
      row-key="id"
      :selected-row-key="selectedRunId"
      @row-click="onRowClick"
    />
  </div>
</template>

<script setup lang="ts">
import type { WorkflowRunDTO } from '@/api/types'
import Button from '@/components/ui/Button.vue'
import Select from '@/components/ui/Select.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import type { TableState } from '@/design-system/types'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    runs: WorkflowRunDTO[]
    selectedRunId: string | null
    selectedTemplateId: string | null
    busy?: boolean
    state?: TableState
  }>(),
  {
    busy: false,
    state: 'ready',
  },
)

const emit = defineEmits<{
  (e: 'select', runId: string): void
  (e: 'create', payload: { mode: 'sync' | 'running' | 'fail' }): void
  (e: 'cancel', runId: string): void
  (e: 'refresh'): void
}>()

const { t } = useI18n({ useScope: 'global' })

const mode = ref<'sync' | 'running' | 'fail'>('sync')

const modeOptions = computed(() => [
  { value: 'sync', label: 'sync' },
  { value: 'running', label: 'running' },
  { value: 'fail', label: 'fail' },
])

const columns = computed<TableColumn[]>(() => [
  { key: 'status', label: t('page.canvas.fieldStatus'), width: '9rem' },
  { key: 'attempt', label: t('page.canvas.fieldAttempt'), width: '7rem', align: 'right' },
  { key: 'templateId', label: t('page.canvas.fieldTemplateId'), mono: true, width: '16rem' },
  { key: 'id', label: t('page.canvas.fieldRunId'), mono: true, width: '16rem' },
])

const rows = computed(() =>
  props.runs.map((item) => ({
    id: item.id,
    status: item.status,
    attempt: item.attempt,
    templateId: item.templateId,
  })),
)

function onCreateRun(): void {
  emit('create', { mode: mode.value })
}

function onRowClick(payload: { rowKey: string }): void {
  emit('select', payload.rowKey)
}
</script>
