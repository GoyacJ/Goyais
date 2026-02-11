<template>
  <section class="ui-page">
    <PageHeader :title="t('page.contextBundle.title')" :subtitle="t('page.contextBundle.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isRefreshing" @click="onRefresh">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <WindowBoard route-key="context-bundles" :panes="windowPanes">
      <template #context-list>
        <SectionCard :title="t('page.contextBundle.listTitle')" :subtitle="String(rows.length)">
          <div class="mb-3 grid gap-2 md:grid-cols-[0.9fr_1fr_0.8fr_auto]">
            <Select v-model="scopeType" :options="scopeTypeOptions" />
            <Input v-model="scopeId" :placeholder="t('page.contextBundle.fieldScopeId')" />
            <Select v-model="scopeVisibility" :options="visibilityOptions" />
            <Button :disabled="isActionBusy || scopeId.trim().length === 0" @click="onRebuildBundle">
              {{ t('page.contextBundle.actionRebuild') }}
            </Button>
          </div>

          <Table
            :columns="columns"
            :rows="rows"
            :state="tableState"
            :error-message="tableErrorMessage"
            :caption="t('page.contextBundle.listTitle')"
            interactive-rows
            row-key="id"
            :selected-row-key="selectedBundleId"
            @row-click="onRowClick"
          />
        </SectionCard>
      </template>

      <template #context-detail>
        <SectionCard :title="t('page.contextBundle.detailTitle')" :subtitle="selectedBundle?.id ?? '-'">
          <div v-if="selectedBundle" class="space-y-3 text-xs">
            <div class="rounded-lg border border-ui-border bg-ui-panel p-3 text-ui-muted">
              <div>{{ t('page.contextBundle.fieldScopeType') }}: {{ selectedBundle.scopeType }}</div>
              <div>{{ t('page.contextBundle.fieldScopeId') }}: {{ selectedBundle.scopeId }}</div>
              <div>{{ t('page.contextBundle.fieldVisibility') }}: {{ selectedBundle.visibility }}</div>
            </div>

            <div class="grid gap-2 md:grid-cols-3">
              <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
                <div class="text-[11px] text-ui-muted">{{ t('page.contextBundle.metricCoverage') }}</div>
                <div class="mt-1 text-sm font-semibold text-ui-text">{{ factCoverage }}</div>
              </div>
              <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
                <div class="text-[11px] text-ui-muted">{{ t('page.contextBundle.metricCommandCount') }}</div>
                <div class="mt-1 text-sm font-semibold text-ui-text">{{ factCommandCount }}</div>
              </div>
              <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
                <div class="text-[11px] text-ui-muted">{{ t('page.contextBundle.metricRunCount') }}</div>
                <div class="mt-1 text-sm font-semibold text-ui-text">{{ factRunCount }}</div>
              </div>
              <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
                <div class="text-[11px] text-ui-muted">{{ t('page.contextBundle.metricSessionCount') }}</div>
                <div class="mt-1 text-sm font-semibold text-ui-text">{{ factSessionCount }}</div>
              </div>
              <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
                <div class="text-[11px] text-ui-muted">{{ t('page.contextBundle.metricAssetCount') }}</div>
                <div class="mt-1 text-sm font-semibold text-ui-text">{{ factAssetCount }}</div>
              </div>
              <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
                <div class="text-[11px] text-ui-muted">{{ t('page.contextBundle.metricWarningCount') }}</div>
                <div class="mt-1 text-sm font-semibold text-ui-text">{{ warningItems.length }}</div>
              </div>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 font-medium text-ui-text">{{ t('page.contextBundle.highlightsTitle') }}</div>
              <ul v-if="highlightItems.length > 0" class="space-y-1 text-[11px] text-ui-muted">
                <li v-for="item in highlightItems" :key="item">{{ item }}</li>
              </ul>
              <div v-else class="text-[11px] text-ui-muted">-</div>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 font-medium text-ui-text">{{ t('page.contextBundle.recommendationsTitle') }}</div>
              <ul v-if="recommendationItems.length > 0" class="space-y-1 text-[11px] text-ui-muted">
                <li v-for="item in recommendationItems" :key="item">{{ item }}</li>
              </ul>
              <div v-else class="text-[11px] text-ui-muted">-</div>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 font-medium text-ui-text">{{ t('page.contextBundle.recentFailuresTitle') }}</div>
              <div class="grid gap-2 md:grid-cols-2">
                <div class="space-y-1">
                  <p class="text-[11px] font-semibold text-ui-text">commands</p>
                  <ul v-if="recentFailedCommands.length > 0" class="space-y-1 text-[11px] text-ui-muted">
                    <li v-for="item in recentFailedCommands" :key="`cmd:${item}`">{{ item }}</li>
                  </ul>
                  <p v-else class="text-[11px] text-ui-muted">-</p>
                </div>
                <div class="space-y-1">
                  <p class="text-[11px] font-semibold text-ui-text">runs</p>
                  <ul v-if="recentFailedRuns.length > 0" class="space-y-1 text-[11px] text-ui-muted">
                    <li v-for="item in recentFailedRuns" :key="`run:${item}`">{{ item }}</li>
                  </ul>
                  <p v-else class="text-[11px] text-ui-muted">-</p>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 font-medium text-ui-text">{{ t('page.contextBundle.timelineTitle') }}</div>
              <ul v-if="timelineDigestItems.length > 0" class="space-y-1 text-[11px] text-ui-muted">
                <li v-for="item in timelineDigestItems" :key="item.id">
                  {{ item.ts }} · {{ item.type }} · {{ item.scope }} · {{ item.refId }}
                </li>
              </ul>
              <p v-else class="text-[11px] text-ui-muted">{{ t('page.contextBundle.timelineEmpty') }}</p>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 font-medium text-ui-text">{{ t('page.contextBundle.rawPayloadTitle') }}</div>
              <pre class="ui-scrollbar max-h-52 overflow-auto rounded border border-ui-border bg-ui-bg p-2">{{
                formatJSON({
                  facts: selectedBundle.facts,
                  summaries: selectedBundle.summaries,
                  refs: selectedBundle.refs,
                  timeline: selectedBundle.timeline,
                })
              }}</pre>
            </div>
          </div>

          <div v-else class="text-sm text-ui-muted">{{ t('page.contextBundle.emptyDetail') }}</div>
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
import { createCommand } from '@/api/commands'
import { getContextBundle, listContextBundles } from '@/api/context-bundles'
import { ApiHttpError } from '@/api/http'
import type { ApiError, ContextBundleDTO } from '@/api/types'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import { useToast } from '@/composables/useToast'
import type { TableState } from '@/design-system/types'
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const bundles = ref<ContextBundleDTO[]>([])
const selectedBundleId = ref<string | null>(null)
const selectedBundle = ref<ContextBundleDTO | null>(null)

const scopeType = ref<'workspace' | 'run' | 'session'>('workspace')
const scopeId = ref('workspace-alpha')
const scopeVisibility = ref<'PRIVATE' | 'WORKSPACE'>('PRIVATE')

const tableState = ref<TableState>('loading')
const apiError = ref<ApiError | null>(null)
const isActionBusy = ref(false)

const windowPanes = computed(() => [
  { id: 'context-list', title: t('page.contextBundle.listTitle') },
  { id: 'context-detail', title: t('page.contextBundle.detailTitle') },
])

const scopeTypeOptions = computed(() => [
  { value: 'workspace', label: 'workspace' },
  { value: 'run', label: 'run' },
  { value: 'session', label: 'session' },
])

const visibilityOptions = computed(() => [
  { value: 'PRIVATE', label: 'PRIVATE' },
  { value: 'WORKSPACE', label: 'WORKSPACE' },
])

const columns = computed<TableColumn[]>(() => [
  { key: 'scopeType', label: t('page.contextBundle.fieldScopeType'), width: '8rem' },
  { key: 'scopeId', label: t('page.contextBundle.fieldScopeId') },
  { key: 'visibility', label: t('page.contextBundle.fieldVisibility'), width: '9rem' },
  { key: 'id', label: 'id', mono: true, width: '17rem' },
])

const rows = computed(() =>
  bundles.value.map((item) => ({
    id: item.id,
    scopeType: item.scopeType,
    scopeId: item.scopeId,
    visibility: item.visibility,
  })),
)

const tableErrorMessage = computed(() => {
  if (!apiError.value) {
    return t('error.common.internal')
  }
  return t(apiError.value.messageKey || 'error.common.internal', apiError.value.details ?? {})
})

const isRefreshing = computed(() => tableState.value === 'loading')
const factsMap = computed(() => asObject(selectedBundle.value?.facts))
const summariesMap = computed(() => asObject(selectedBundle.value?.summaries))
const refsMap = computed(() => asObject(selectedBundle.value?.refs))

const factCoverage = computed(() => readString(factsMap.value, 'coverage') || '-')
const factCommandCount = computed(() => readNumber(factsMap.value, 'commandCount'))
const factRunCount = computed(() => readNumber(factsMap.value, 'runCount'))
const factSessionCount = computed(() => readNumber(factsMap.value, 'sessionCount'))
const factAssetCount = computed(() => readNumber(factsMap.value, 'assetCount'))
const warningItems = computed(() => readStringArray(factsMap.value, 'warnings'))
const highlightItems = computed(() => readStringArray(summariesMap.value, 'highlights'))
const recommendationItems = computed(() => readStringArray(summariesMap.value, 'recommendations'))

const recentFailedCommands = computed(() => {
  const refs = asObject(refsMap.value.recentFailures)
  const commands = asArray(refs.commands)
  return commands
    .map((item) => readString(asObject(item), 'id') || readString(asObject(item), 'commandId'))
    .filter((item) => item.length > 0)
    .slice(0, 8)
})

const recentFailedRuns = computed(() => {
  const refs = asObject(refsMap.value.recentFailures)
  const runs = asArray(refs.runs)
  return runs
    .map((item) => readString(asObject(item), 'id') || readString(asObject(item), 'runId'))
    .filter((item) => item.length > 0)
    .slice(0, 8)
})

type TimelineDigestItem = {
  id: string
  ts: string
  type: string
  scope: string
  refId: string
}

const timelineDigestItems = computed<TimelineDigestItem[]>(() => {
  if (!selectedBundle.value) {
    return []
  }
  return asArray(selectedBundle.value.timeline)
    .map((item, index) => {
      const payload = asObject(item)
      const ts = readString(payload, 'ts')
      const type = readString(payload, 'type')
      const scope = readString(payload, 'scope')
      const refId = readString(payload, 'refId')
      return {
        id: `${index}:${ts}:${type}:${refId}`,
        ts: ts || '-',
        type: type || '-',
        scope: scope || '-',
        refId: refId || '-',
      }
    })
    .slice(0, 20)
})

watch(
  bundles,
  (items) => {
    if (!items.some((item) => item.id === selectedBundleId.value)) {
      selectedBundleId.value = items[0]?.id ?? null
    }
  },
  { immediate: true },
)

watch(selectedBundleId, (bundleId) => {
  if (!bundleId) {
    selectedBundle.value = null
    return
  }
  void loadBundleDetail(bundleId)
})

onMounted(() => {
  void loadBundles()
})

async function loadBundles(): Promise<void> {
  tableState.value = 'loading'
  apiError.value = null
  try {
    const response = await listContextBundles({ page: 1, pageSize: 200 })
    bundles.value = response.items
    tableState.value = 'ready'
  } catch (error) {
    bundles.value = []
    apiError.value = asApiError(error)
    tableState.value = 'error'
  }
}

async function loadBundleDetail(bundleId: string): Promise<void> {
  try {
    selectedBundle.value = await getContextBundle(bundleId)
  } catch (error) {
    selectedBundle.value = null
    apiError.value = asApiError(error)
  }
}

function onRowClick(payload: { rowKey: string }): void {
  selectedBundleId.value = payload.rowKey
}

async function onRefresh(): Promise<void> {
  await loadBundles()
  if (selectedBundleId.value) {
    await loadBundleDetail(selectedBundleId.value)
  }
}

async function onRebuildBundle(): Promise<void> {
  if (isActionBusy.value || scopeId.value.trim().length === 0) {
    return
  }
  isActionBusy.value = true
  try {
    const response = await createCommand({
      commandType: 'context.bundle.rebuild',
      payload: {
        scopeType: scopeType.value,
        scopeId: scopeId.value.trim(),
        visibility: scopeVisibility.value,
      },
      visibility: scopeVisibility.value,
    })
    pushToast({
      title: t('page.contextBundle.actionRebuild'),
      message: `${t('page.streams.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadBundles()
    const commandResult = response.resource.result as Record<string, unknown> | undefined
    const bundle = commandResult?.bundle as Record<string, unknown> | undefined
    const rebuiltBundleID = typeof bundle?.id === 'string' ? bundle.id.trim() : ''
    if (rebuiltBundleID) {
      selectedBundleId.value = rebuiltBundleID
      await loadBundleDetail(rebuiltBundleID)
    }
  } catch (error) {
    const apiErr = asApiError(error)
    pushToast({
      title: t('page.contextBundle.actionRebuild'),
      message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
      tone: 'error',
    })
  } finally {
    isActionBusy.value = false
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

function asObject(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {}
  }
  return value as Record<string, unknown>
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}

function readString(payload: Record<string, unknown>, key: string): string {
  const value = payload[key]
  return typeof value === 'string' ? value : ''
}

function readNumber(payload: Record<string, unknown>, key: string): number {
  const value = payload[key]
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim().length > 0) {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) {
      return parsed
    }
  }
  return 0
}

function readStringArray(payload: Record<string, unknown>, key: string): string[] {
  return asArray(payload[key])
    .map((item) => (typeof item === 'string' ? item.trim() : ''))
    .filter((item) => item.length > 0)
}
</script>
