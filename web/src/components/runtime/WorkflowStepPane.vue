<template>
  <div class="space-y-3">
    <p class="text-xs text-ui-muted">
      {{ runId ? `${t('page.canvas.fieldRunId')}: ${runId}` : t('page.canvas.stepsEmptyDescription') }}
    </p>
    <Table
      :columns="columns"
      :rows="rows"
      :state="tableState"
      :caption="t('page.canvas.stepsTitle')"
    />
  </div>
</template>

<script setup lang="ts">
import type { StepRunDTO } from '@/api/types'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import type { TableState } from '@/design-system/types'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    runId: string | null
    steps: StepRunDTO[]
    loading?: boolean
  }>(),
  {
    loading: false,
  },
)

const { t } = useI18n({ useScope: 'global' })

const columns = computed<TableColumn[]>(() => [
  { key: 'stepKey', label: t('page.canvas.fieldStepKey') },
  { key: 'status', label: t('page.canvas.fieldStatus'), width: '9rem' },
  { key: 'durationMs', label: t('page.canvas.fieldDurationMs'), align: 'right', width: '8rem' },
  { key: 'traceId', label: t('page.canvas.fieldTraceId'), mono: true, width: '16rem' },
])

const rows = computed(() =>
  props.steps.map((item) => ({
    id: item.id,
    stepKey: item.stepKey,
    status: item.status,
    durationMs: item.durationMs ?? '-',
    traceId: item.traceId || '-',
  })),
)

const tableState = computed<TableState>(() => {
  if (props.loading) {
    return 'loading'
  }
  if (!props.runId) {
    return 'empty'
  }
  return rows.value.length > 0 ? 'ready' : 'empty'
})
</script>
