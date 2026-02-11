<template>
  <section class="ui-page">
    <PageHeader :title="t('page.algorithmLibrary.title')" :subtitle="t('page.algorithmLibrary.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isRefreshing" @click="onRefresh">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <WindowBoard route-key="algorithm-library" :panes="windowPanes">
      <template #algorithm-list>
        <SectionCard :title="t('page.algorithmLibrary.listTitle')" :subtitle="String(rows.length)">
          <Table
            :columns="columns"
            :rows="rows"
            :state="tableState"
            :error-message="tableErrorMessage"
            :caption="t('page.algorithmLibrary.listTitle')"
            interactive-rows
            row-key="id"
            :selected-row-key="selectedAlgorithmId"
            @row-click="onRowClick"
          />
        </SectionCard>
      </template>

      <template #algorithm-detail>
        <SectionCard :title="t('page.algorithmLibrary.detailTitle')" :subtitle="selectedAlgorithm?.id ?? '-'">
          <div v-if="selectedAlgorithm" class="space-y-3 text-xs">
            <div class="rounded-lg border border-ui-border bg-ui-panel p-3 text-ui-muted">
              <div>{{ t('page.canvas.fieldAlgorithmName') }}: {{ selectedAlgorithm.name }}</div>
              <div>{{ t('page.canvas.fieldVersion') }}: {{ selectedAlgorithm.version }}</div>
              <div>{{ t('page.canvas.fieldTemplateRef') }}: {{ selectedAlgorithm.templateRef }}</div>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 font-medium text-ui-text">defaults</div>
              <pre class="ui-scrollbar max-h-44 overflow-auto rounded border border-ui-border bg-ui-bg p-2">{{
                formatJSON(selectedAlgorithm.defaults)
              }}</pre>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 font-medium text-ui-text">constraints</div>
              <pre class="ui-scrollbar max-h-44 overflow-auto rounded border border-ui-border bg-ui-bg p-2">{{
                formatJSON(selectedAlgorithm.constraints)
              }}</pre>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 font-medium text-ui-text">dependencies</div>
              <pre class="ui-scrollbar max-h-44 overflow-auto rounded border border-ui-border bg-ui-bg p-2">{{
                formatJSON(selectedAlgorithm.dependencies)
              }}</pre>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 font-medium text-ui-text">{{ t('page.algorithmLibrary.runTitle') }}</div>
              <p class="mb-2 text-[11px] text-ui-muted">{{ t('page.algorithmLibrary.runSubtitle') }}</p>
              <Textarea
                v-model="runInputsRaw"
                :rows="6"
                :placeholder="t('page.algorithmLibrary.runInputPlaceholder')"
              />
              <div class="mt-2 grid gap-2 md:grid-cols-[1fr_auto]">
                <Select v-model="runMode" :options="runModeOptions" />
                <Button :disabled="runSubmitting" @click="onRunAlgorithm">
                  {{ t('page.algorithmLibrary.actionRun') }}
                </Button>
              </div>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3 text-ui-muted">
              <div class="mb-2 font-medium text-ui-text">{{ t('page.algorithmLibrary.runResultTitle') }}</div>
              <div v-if="lastRunResult" class="space-y-1">
                <div>runId: {{ lastRunResult.id }}</div>
                <div>{{ t('page.algorithmLibrary.runFieldStatus') }}: {{ lastRunResult.status }}</div>
                <div>{{ t('page.algorithmLibrary.runFieldWorkflowRunId') }}: {{ lastRunResult.workflowRunId }}</div>
                <div>{{ t('page.algorithmLibrary.runFieldAssetIds') }}: {{ lastRunResult.assetIds.join(', ') || '-' }}</div>
                <div>{{ t('page.algorithmLibrary.runFieldCommandId') }}: {{ lastRunResult.commandId }}</div>
              </div>
              <div v-else class="text-sm">{{ t('page.algorithmLibrary.runResultEmpty') }}</div>
            </div>
          </div>

          <div v-else class="text-sm text-ui-muted">{{ t('page.algorithmLibrary.emptyDetail') }}</div>
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
import { runAlgorithm } from '@/api/algorithms'
import { listAlgorithms } from '@/api/registry'
import type { AlgorithmDTO, ApiError } from '@/api/types'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Select from '@/components/ui/Select.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import Textarea from '@/components/ui/Textarea.vue'
import { useToast } from '@/composables/useToast'
import type { TableState } from '@/design-system/types'
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const algorithms = ref<AlgorithmDTO[]>([])
const selectedAlgorithmId = ref<string | null>(null)
const tableState = ref<TableState>('loading')
const apiError = ref<ApiError | null>(null)
const runInputsRaw = ref('{\n  \n}')
const runMode = ref<'sync' | 'running' | 'fail'>('sync')
const runSubmitting = ref(false)
const lastRunResult = ref<{
  id: string
  status: string
  workflowRunId: string
  assetIds: string[]
  commandId: string
} | null>(null)

const windowPanes = computed(() => [
  { id: 'algorithm-list', title: t('page.algorithmLibrary.listTitle') },
  { id: 'algorithm-detail', title: t('page.algorithmLibrary.detailTitle') },
])

const columns = computed<TableColumn[]>(() => [
  { key: 'name', label: t('page.canvas.fieldAlgorithmName') },
  { key: 'version', label: t('page.canvas.fieldVersion'), width: '9rem' },
  { key: 'templateRef', label: t('page.canvas.fieldTemplateRef') },
  { key: 'id', label: 'id', mono: true, width: '17rem' },
])

const rows = computed(() =>
  algorithms.value.map((item) => ({
    id: item.id,
    name: item.name,
    version: item.version,
    templateRef: item.templateRef,
  })),
)

const selectedAlgorithm = computed(
  () => algorithms.value.find((item) => item.id === selectedAlgorithmId.value) ?? null,
)
const runModeOptions = computed(() => [
  { value: 'sync', label: 'sync' },
  { value: 'running', label: 'running' },
  { value: 'fail', label: 'fail' },
])

const tableErrorMessage = computed(() => {
  if (!apiError.value) {
    return t('error.common.internal')
  }
  return t(apiError.value.messageKey || 'error.common.internal', apiError.value.details ?? {})
})

const isRefreshing = computed(() => tableState.value === 'loading')

watch(
  algorithms,
  (items) => {
    if (!items.some((item) => item.id === selectedAlgorithmId.value)) {
      selectedAlgorithmId.value = items[0]?.id ?? null
    }
  },
  { immediate: true },
)

onMounted(() => {
  void loadAlgorithms()
})

async function loadAlgorithms(): Promise<void> {
  tableState.value = 'loading'
  apiError.value = null
  try {
    const response = await listAlgorithms({ page: 1, pageSize: 200 })
    algorithms.value = response.items
    tableState.value = 'ready'
  } catch (error) {
    algorithms.value = []
    apiError.value = asApiError(error)
    tableState.value = 'error'
  }
}

function onRowClick(payload: { rowKey: string }): void {
  selectedAlgorithmId.value = payload.rowKey
  runInputsRaw.value = '{\n  \n}'
  lastRunResult.value = null
}

async function onRefresh(): Promise<void> {
  await loadAlgorithms()
}

async function onRunAlgorithm(): Promise<void> {
  if (!selectedAlgorithm.value || runSubmitting.value) {
    return
  }
  let inputs: Record<string, unknown>
  try {
    const parsed = JSON.parse(runInputsRaw.value || '{}') as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error('inputs_not_object')
    }
    inputs = parsed as Record<string, unknown>
  } catch {
    pushToast({
      tone: 'error',
      title: t('page.algorithmLibrary.actionRun'),
      message: t('error.request.invalid_json'),
    })
    return
  }

  runSubmitting.value = true
  try {
    const response = await runAlgorithm(selectedAlgorithm.value.id, {
      inputs,
      mode: runMode.value,
    })
    lastRunResult.value = {
      id: response.resource.id,
      status: response.resource.status,
      workflowRunId: response.resource.workflowRunId,
      assetIds: Array.isArray(response.resource.assetIds) ? response.resource.assetIds : [],
      commandId: response.commandRef.commandId,
    }
    pushToast({
      tone: 'success',
      title: t('page.algorithmLibrary.actionRun'),
      message: `${t('page.algorithmLibrary.runFieldCommandId')}: ${response.commandRef.commandId}`,
    })
  } catch (error) {
    const apiErr = asApiError(error)
    pushToast({
      tone: 'error',
      title: t('page.algorithmLibrary.actionRun'),
      message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
    })
  } finally {
    runSubmitting.value = false
  }
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
