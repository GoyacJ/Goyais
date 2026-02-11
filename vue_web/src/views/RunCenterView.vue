<template>
  <section class="ui-page">
    <PageHeader :title="t('page.runCenter.title')" :subtitle="t('page.runCenter.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isRefreshing" @click="onRefresh">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <WindowBoard route-key="run-center" :panes="windowPanes">
      <template #run-center-list>
        <SectionCard :title="t('page.runCenter.listTitle')" :subtitle="String(runRows.length)">
          <Table
            :columns="runColumns"
            :rows="runRows"
            :state="tableState"
            :error-message="tableErrorMessage"
            :caption="t('page.runCenter.listTitle')"
            interactive-rows
            row-key="id"
            :selected-row-key="selectedRunId"
            @row-click="onRunRowClick"
          />
        </SectionCard>
      </template>

      <template #run-center-detail>
        <SectionCard :title="t('page.runCenter.detailTitle')" :subtitle="selectedRun?.id ?? '-'">
          <div v-if="selectedRun" class="space-y-3">
            <div class="rounded-lg border border-ui-border bg-ui-panel p-3 text-xs text-ui-muted">
              <div>{{ t('page.canvas.fieldTemplateId') }}: {{ selectedRun.templateId }}</div>
              <div>{{ t('page.canvas.fieldStatus') }}: {{ selectedRun.status }}</div>
              <div>{{ t('page.canvas.fieldAttempt') }}: {{ selectedRun.attempt }}</div>
              <div>{{ t('page.canvas.fieldTraceId') }}: {{ selectedRun.traceId || '-' }}</div>
            </div>

            <div>
              <div class="mb-2 text-xs font-medium text-ui-text">{{ t('page.runCenter.eventsTitle') }}</div>
              <div class="space-y-1 text-xs text-ui-muted">
                <div v-for="event in eventRows" :key="event.id" class="rounded border border-ui-border px-2 py-1">
                  <div class="flex items-center justify-between gap-2">
                    <span class="ui-monospace text-ui-fg">{{ event.type }}</span>
                    <span>{{ event.createdAt }}</span>
                  </div>
                  <div class="mt-1 text-[11px]">{{ event.summary }}</div>
                </div>
                <div v-if="eventRows.length === 0">{{ t('page.runCenter.eventsEmpty') }}</div>
              </div>
            </div>

            <div>
              <div class="mb-2 text-xs font-medium text-ui-text">{{ t('page.canvas.stepsTitle') }}</div>
              <Table
                :columns="stepColumns"
                :rows="stepRows"
                :state="detailState"
                :error-message="detailErrorMessage"
                :caption="t('page.canvas.stepsTitle')"
                row-key="id"
              />
            </div>
          </div>

          <div v-else class="text-sm text-ui-muted">{{ t('page.runCenter.emptyDetail') }}</div>
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
import { ApiHttpError } from '@/api/http'
import { getWorkflowRun, getWorkflowRunEvents, listWorkflowRuns, listWorkflowStepRuns } from '@/api/workflow'
import type { ApiError, ApiObject, StepRunDTO, WorkflowRunDTO, WorkflowRunEventDTO } from '@/api/types'
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

const runs = ref<WorkflowRunDTO[]>([])
const selectedRunId = ref<string | null>(null)
const selectedRun = ref<WorkflowRunDTO | null>(null)
const steps = ref<StepRunDTO[]>([])
const runEvents = ref<WorkflowRunEventDTO[]>([])

const tableState = ref<TableState>('loading')
const detailState = ref<TableState>('ready')
const apiError = ref<ApiError | null>(null)
const detailError = ref<ApiError | null>(null)

const windowPanes = computed(() => [
  { id: 'run-center-list', title: t('page.runCenter.listTitle') },
  { id: 'run-center-detail', title: t('page.runCenter.detailTitle') },
])

const runColumns = computed<TableColumn[]>(() => [
  { key: 'id', label: t('page.canvas.fieldRunId'), mono: true, width: '17rem' },
  { key: 'templateId', label: t('page.canvas.fieldTemplateId') },
  { key: 'status', label: t('page.canvas.fieldStatus'), width: '9rem' },
  { key: 'attempt', label: t('page.canvas.fieldAttempt'), width: '7rem', align: 'right' },
])

const runRows = computed(() =>
  runs.value.map((item) => ({
    id: item.id,
    templateId: item.templateId,
    status: item.status,
    attempt: item.attempt,
  })),
)

const stepColumns = computed<TableColumn[]>(() => [
  { key: 'stepKey', label: t('page.canvas.fieldStepKey'), mono: true },
  { key: 'status', label: t('page.canvas.fieldStatus'), width: '9rem' },
  { key: 'attempt', label: t('page.canvas.fieldAttempt'), width: '7rem', align: 'right' },
  { key: 'durationMs', label: t('page.canvas.fieldDurationMs'), width: '9rem', align: 'right' },
])

const stepRows = computed(() =>
  steps.value.map((item) => ({
    id: item.id,
    stepKey: item.stepKey,
    status: item.status,
    attempt: item.attempt,
    durationMs: typeof item.durationMs === 'number' ? item.durationMs : '-',
  })),
)

const eventRows = computed(() =>
  runEvents.value.map((item, index) => {
    const payload = asObject(item.data)
    return {
      id: item.id || `event_${index + 1}`,
      type: item.event || 'workflow.event',
      createdAt: readString(payload, 'createdAt') || readString(payload, 'ts') || '-',
      summary: summarizeEventPayload(payload),
    }
  }),
)

const tableErrorMessage = computed(() => {
  if (!apiError.value) {
    return t('error.common.internal')
  }
  return t(apiError.value.messageKey || 'error.common.internal', apiError.value.details ?? {})
})

const detailErrorMessage = computed(() => {
  if (!detailError.value) {
    return t('error.common.internal')
  }
  return t(detailError.value.messageKey || 'error.common.internal', detailError.value.details ?? {})
})

const isRefreshing = computed(() => tableState.value === 'loading')

watch(
  runs,
  (items) => {
    if (!items.some((item) => item.id === selectedRunId.value)) {
      selectedRunId.value = items[0]?.id ?? null
    }
  },
  { immediate: true },
)

watch(selectedRunId, (runId) => {
  if (!runId) {
    selectedRun.value = null
    steps.value = []
    runEvents.value = []
    detailState.value = 'ready'
    return
  }
  void loadRunDetail(runId)
})

onMounted(() => {
  void loadRuns()
})

async function loadRuns(): Promise<void> {
  tableState.value = 'loading'
  apiError.value = null
  try {
    const response = await listWorkflowRuns({ page: 1, pageSize: 200 })
    runs.value = response.items
    tableState.value = 'ready'
  } catch (error) {
    runs.value = []
    apiError.value = asApiError(error)
    tableState.value = 'error'
  }
}

async function loadRunDetail(runId: string): Promise<void> {
  detailState.value = 'loading'
  detailError.value = null
  try {
    const [run, stepsResp, events] = await Promise.all([
      getWorkflowRun(runId),
      listWorkflowStepRuns(runId, { page: 1, pageSize: 200 }),
      getWorkflowRunEvents(runId),
    ])
    selectedRun.value = run
    runs.value = [run, ...runs.value.filter((item) => item.id !== run.id)]
    steps.value = stepsResp.items
    runEvents.value = events
    detailState.value = 'ready'
  } catch (error) {
    selectedRun.value = null
    steps.value = []
    runEvents.value = []
    detailError.value = asApiError(error)
    detailState.value = 'error'
  }
}

function onRunRowClick(payload: { rowKey: string }): void {
  selectedRunId.value = payload.rowKey
}

async function onRefresh(): Promise<void> {
  await loadRuns()
  if (selectedRunId.value) {
    await loadRunDetail(selectedRunId.value)
  }
}

function summarizeEventPayload(payload: ApiObject): string {
  const parts: string[] = []
  const runId = readString(payload, 'runId')
  const stepKey = readString(payload, 'stepKey')
  const status = readString(payload, 'status')
  const commandType = readString(payload, 'commandType')
  if (runId) {
    parts.push(`runId=${runId}`)
  }
  if (stepKey) {
    parts.push(`stepKey=${stepKey}`)
  }
  if (status) {
    parts.push(`status=${status}`)
  }
  if (commandType) {
    parts.push(`commandType=${commandType}`)
  }
  if (parts.length > 0) {
    return parts.join(' · ')
  }
  return Object.keys(payload).length > 0 ? JSON.stringify(payload) : '-'
}

function readString(payload: ApiObject, key: string): string {
  const raw = payload[key]
  return typeof raw === 'string' ? raw.trim() : ''
}

function asObject(value: unknown): ApiObject {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {}
  }
  return value as ApiObject
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
</script>
