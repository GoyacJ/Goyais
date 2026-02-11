<template>
  <section class="ui-page">
    <PageHeader :title="t('page.streams.title')" :subtitle="t('page.streams.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isRefreshing" @click="onRefresh">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <WindowBoard route-key="streams" :panes="windowPanes">
      <template #stream-overview>
        <SectionCard :title="t('page.streams.overviewTitle')" :subtitle="overviewSubtitle">
          <div class="grid gap-3 xl:grid-cols-[1.4fr_1fr]">
            <Table
              :columns="columns"
              :rows="rows"
              :state="tableState"
              :error-message="tableErrorMessage"
              :caption="t('page.streams.overviewTitle')"
              interactive-rows
              row-key="id"
              :selected-row-key="selectedStreamId"
              @row-click="onStreamRowClick"
            />
            <StreamControlPanel
              :selected-stream="selectedStream"
              :busy="isActionBusy"
              @create="onCreateStream"
              @record-start="onRecordStart"
              @record-stop="onRecordStop"
              @kick="onKickStream"
              @update-auth="onUpdateAuth"
              @delete="onDeleteStream"
            />
          </div>
        </SectionCard>
      </template>

      <template #stream-logs>
        <SectionCard :title="t('page.streams.logsTitle')" :subtitle="t('page.streams.logsSubtitle')">
          <StreamCommandLog :events="streamEvents" />
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
import { ApiHttpError, isMockEnabled } from '@/api/http'
import {
  createStream,
  deleteStream,
  getStream,
  kickStream,
  listStreams,
  startStreamRecording,
  stopStreamRecording,
  updateStreamAuth,
} from '@/api/streams'
import type { ApiError, ApiObject, CommandDTO, StreamCreateRequest, StreamDTO } from '@/api/types'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import StreamCommandLog, { type StreamCommandLogEvent } from '@/components/runtime/StreamCommandLog.vue'
import StreamControlPanel, {
  type StreamAuthFormValue,
  type StreamCreateFormValue,
} from '@/components/runtime/StreamControlPanel.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import { useToast } from '@/composables/useToast'
import type { CommandStatus, TableState } from '@/design-system/types'
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const useMock = isMockEnabled()
const streams = ref<StreamDTO[]>([])
const streamCommands = ref<CommandDTO[]>([])
const selectedStreamId = ref<string | null>(null)
const tableState = ref<TableState>('loading')
const apiError = ref<ApiError | null>(null)
const isActionBusy = ref(false)

const windowPanes = computed(() => [
  { id: 'stream-overview', title: t('page.streams.overviewTitle') },
  { id: 'stream-logs', title: t('page.streams.logsTitle') },
])

const columns = computed<TableColumn[]>(() => [
  { key: 'path', label: t('page.streams.fieldPath') },
  { key: 'protocol', label: t('page.streams.fieldProtocol'), width: '8rem' },
  { key: 'status', label: t('page.streams.fieldStatus'), width: '9rem' },
  { key: 'id', label: t('page.streams.fieldStreamId'), mono: true, width: '17rem' },
])

const rows = computed(() =>
  streams.value.map((item) => ({
    id: item.id,
    path: item.path,
    protocol: item.protocol,
    status: item.status,
  })),
)

const selectedStream = computed(() => streams.value.find((item) => item.id === selectedStreamId.value) ?? null)

const isRefreshing = computed(() => tableState.value === 'loading')

const overviewSubtitle = computed(() => {
  if (tableState.value === 'loading') {
    return t('common.loading')
  }
  return String(rows.value.length)
})

const tableErrorMessage = computed(() => {
  if (!apiError.value) {
    return t('error.common.internal')
  }
  return t(apiError.value.messageKey || 'error.common.internal', apiError.value.details ?? {})
})

const streamEvents = computed<StreamCommandLogEvent[]>(() => {
  const events: StreamCommandLogEvent[] = []
  for (const cmd of streamCommands.value) {
    events.push({
      commandId: cmd.id,
      commandType: cmd.commandType,
      acceptedAt: cmd.acceptedAt,
      status: cmd.status,
      summary: summarizeStreamCommand(cmd),
    })
  }
  return events
})

watch(
  streams,
  (items) => {
    if (!items.some((item) => item.id === selectedStreamId.value)) {
      selectedStreamId.value = items[0]?.id ?? null
    }
  },
  { immediate: true },
)

onMounted(() => {
  void loadData()
})

async function loadData(): Promise<void> {
  tableState.value = 'loading'
  apiError.value = null
  try {
    if (useMock) {
      streams.value = []
      streamCommands.value = []
      tableState.value = 'ready'
      return
    }

    const [streamResp, commandResp] = await Promise.all([
      listStreams({ page: 1, pageSize: 200 }),
      listCommands({ page: 1, pageSize: 400 }),
    ])
    streams.value = streamResp.items
    streamCommands.value = commandResp.items.filter((item) => {
      if (item.commandType.startsWith('stream.')) {
        return true
      }
      if (item.commandType !== 'workflow.run') {
        return false
      }
      const payload = asObject(item.payload)
      const inputs = asObject(payload.inputs)
      return inputs.trigger === 'stream.onPublish'
    })
    tableState.value = 'ready'
  } catch (error) {
    streams.value = []
    streamCommands.value = []
    apiError.value = asApiError(error)
    tableState.value = 'error'
  }
}

async function onRefresh(): Promise<void> {
  await loadData()
}

function onStreamRowClick(payload: { rowKey: string }): void {
  selectedStreamId.value = payload.rowKey
  void refreshSelectedStream()
}

async function refreshSelectedStream(): Promise<void> {
  const streamId = selectedStreamId.value
  if (!streamId || useMock) {
    return
  }
  try {
    const detail = await getStream(streamId)
    streams.value = streams.value.map((item) => (item.id === detail.id ? detail : item))
  } catch {
    // keep list snapshot if detail fetch fails
  }
}

async function onCreateStream(form: StreamCreateFormValue): Promise<void> {
  if (useMock || isActionBusy.value) {
    return
  }
  isActionBusy.value = true
  try {
    const request: StreamCreateRequest = {
      path: form.path,
      protocol: form.protocol,
      source: form.source,
      visibility: form.visibility as StreamCreateRequest['visibility'],
      metadata: form.onPublishTemplateId ? { onPublishTemplateId: form.onPublishTemplateId } : {},
    }
    const response = await createStream(request)
    pushToast({
      title: t('page.streams.actionCreate'),
      message: `${t('page.streams.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadData()
    if (typeof response.resource.id === 'string' && response.resource.id) {
      selectedStreamId.value = response.resource.id
    }
  } catch (error) {
    notifyActionError(t('page.streams.actionCreate'), error)
  } finally {
    isActionBusy.value = false
  }
}

async function onRecordStart(): Promise<void> {
  await runStreamAction(t('page.streams.actionRecordStart'), startStreamRecording)
}

async function onRecordStop(): Promise<void> {
  await runStreamAction(t('page.streams.actionRecordStop'), stopStreamRecording)
}

async function onKickStream(): Promise<void> {
  await runStreamAction(t('page.streams.actionKick'), kickStream)
}

async function onUpdateAuth(form: StreamAuthFormValue): Promise<void> {
  const streamId = selectedStreamId.value
  if (!streamId || useMock || isActionBusy.value) {
    return
  }
  let authRule: ApiObject
  try {
    authRule = parseAuthRule(form.raw)
  } catch {
    pushToast({
      title: t('page.streams.actionUpdateAuth'),
      message: t('error.request.invalid_json'),
      tone: 'error',
    })
    return
  }

  isActionBusy.value = true
  try {
    const response = await updateStreamAuth(streamId, authRule)
    pushToast({
      title: t('page.streams.actionUpdateAuth'),
      message: `${t('page.streams.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadData()
    await refreshSelectedStream()
  } catch (error) {
    notifyActionError(t('page.streams.actionUpdateAuth'), error)
  } finally {
    isActionBusy.value = false
  }
}

async function onDeleteStream(): Promise<void> {
  const stream = selectedStream.value
  if (!stream || useMock || isActionBusy.value) {
    return
  }
  if (!isStreamDeletable(stream.status)) {
    pushToast({
      title: t('page.streams.actionDelete'),
      message: t('page.streams.deleteStatusGuard', { status: stream.status }),
      tone: 'error',
    })
    return
  }

  isActionBusy.value = true
  try {
    const response = await deleteStream(stream.id)
    pushToast({
      title: t('page.streams.actionDelete'),
      message: `${t('page.streams.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadData()
    await refreshSelectedStream()
  } catch (error) {
    notifyActionError(t('page.streams.actionDelete'), error)
  } finally {
    isActionBusy.value = false
  }
}

async function runStreamAction(
  title: string,
  action: (streamId: string) => Promise<{ commandRef: { commandId: string } }>,
): Promise<void> {
  const streamId = selectedStreamId.value
  if (!streamId || useMock || isActionBusy.value) {
    return
  }
  isActionBusy.value = true
  try {
    const response = await action(streamId)
    pushToast({
      title,
      message: `${t('page.streams.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadData()
    await refreshSelectedStream()
  } catch (error) {
    notifyActionError(title, error)
  } finally {
    isActionBusy.value = false
  }
}

function summarizeStreamCommand(cmd: CommandDTO): string {
  if (cmd.error?.messageKey) {
    return t(cmd.error.messageKey, cmd.error.details ?? {})
  }
  const result = asObject(cmd.result)
  if (cmd.commandType === 'stream.record.stop') {
    const assetId = typeof result.assetId === 'string' ? result.assetId : ''
    return assetId ? `${t('page.streams.fieldAssetId')}: ${assetId}` : cmd.commandType
  }
  if (cmd.commandType === 'stream.record.start') {
    const onPublish = asObject(result.onPublish)
    const onPublishCommandId = typeof onPublish.commandId === 'string' ? onPublish.commandId : ''
    return onPublishCommandId
      ? `${t('page.streams.fieldOnPublishCommandId')}: ${onPublishCommandId}`
      : cmd.commandType
  }
  if (cmd.commandType === 'workflow.run') {
    const payload = asObject(cmd.payload)
    const inputs = asObject(payload.inputs)
    const streamId = typeof inputs.streamId === 'string' ? inputs.streamId : ''
    const recordingId = typeof inputs.recordingId === 'string' ? inputs.recordingId : ''
    return streamId || recordingId ? `${streamId} / ${recordingId}` : cmd.commandType
  }
  const stream = asObject(result.stream)
  const path = typeof stream.path === 'string' ? stream.path : ''
  return path || cmd.commandType
}

function notifyActionError(title: string, error: unknown): void {
  const apiErr = asApiError(error)
  pushToast({
    title,
    message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
    tone: 'error',
  })
}

function asApiError(value: unknown): ApiError {
  if (value instanceof ApiHttpError) {
    return value.error
  }
  return {
    code: 'INTERNAL_ERROR',
    messageKey: 'error.common.internal',
  }
}

function parseAuthRule(raw: string): ApiObject {
  const trimmed = raw.trim()
  if (trimmed.length === 0) {
    return {}
  }
  const parsed = JSON.parse(trimmed) as unknown
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('invalid_auth_rule')
  }
  return parsed as ApiObject
}

function isStreamDeletable(status: string): boolean {
  const normalized = status.trim().toLowerCase()
  return normalized === 'offline' || normalized === 'error'
}

function asObject(value: unknown): ApiObject {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {}
  }
  return value as ApiObject
}
</script>
