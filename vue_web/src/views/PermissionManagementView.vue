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
        <SectionCard :title="t('page.permissionManagement.overviewTitle')" :subtitle="String(shareRows.length)">
          <div class="mb-3 rounded-lg border border-ui-border bg-ui-panel p-3 text-xs text-ui-muted">
            {{ t('page.permissionManagement.overviewHint') }}
          </div>

          <div class="mb-3 rounded-lg border border-ui-border bg-ui-panel p-3">
            <div class="mb-2 text-xs font-medium text-ui-text">{{ t('page.permissionManagement.editorTitle') }}</div>
            <div class="grid gap-2 md:grid-cols-2">
              <div>
                <div class="mb-1 text-[11px] text-ui-muted">{{ t('page.permissionManagement.fieldResourceType') }}</div>
                <Select v-model="shareForm.resourceType" :options="resourceTypeOptions" />
              </div>
              <div>
                <div class="mb-1 text-[11px] text-ui-muted">{{ t('page.permissionManagement.fieldResourceId') }}</div>
                <Input v-model="shareForm.resourceId" :placeholder="t('page.permissionManagement.placeholderResourceId')" />
              </div>
              <div>
                <div class="mb-1 text-[11px] text-ui-muted">{{ t('page.permissionManagement.fieldSubjectType') }}</div>
                <Select v-model="shareForm.subjectType" :options="subjectTypeOptions" />
              </div>
              <div>
                <div class="mb-1 text-[11px] text-ui-muted">{{ t('page.permissionManagement.fieldSubjectId') }}</div>
                <Input v-model="shareForm.subjectId" :placeholder="t('page.permissionManagement.placeholderSubjectId')" />
              </div>
              <div class="md:col-span-2">
                <div class="mb-1 text-[11px] text-ui-muted">{{ t('page.permissionManagement.fieldPermissions') }}</div>
                <Input
                  v-model="shareForm.permissionsRaw"
                  :placeholder="t('page.permissionManagement.placeholderPermissions')"
                />
                <p class="mt-1 text-[11px] text-ui-muted">{{ t('page.permissionManagement.permissionsHint') }}</p>
              </div>
              <div class="md:col-span-2">
                <div class="mb-1 text-[11px] text-ui-muted">{{ t('page.permissionManagement.fieldExpiresAt') }}</div>
                <Input v-model="shareForm.expiresAt" :placeholder="t('page.permissionManagement.placeholderExpiresAt')" />
              </div>
            </div>

            <div class="mt-3 flex flex-wrap gap-2">
              <Button :disabled="isActionBusy" @click="onGrantShare">{{ t('page.permissionManagement.actionGrant') }}</Button>
              <Button variant="secondary" :disabled="isActionBusy || !selectedShareId" @click="onRevokeShare">
                {{ t('page.permissionManagement.actionRevoke') }}
              </Button>
            </div>
          </div>

          <div class="mb-2 text-xs font-medium text-ui-text">{{ t('page.permissionManagement.policyTableTitle') }}</div>
          <Table
            :columns="shareColumns"
            :rows="shareRows"
            :state="tableState"
            :error-message="tableErrorMessage"
            :caption="t('page.permissionManagement.policyTableTitle')"
            interactive-rows
            row-key="id"
            :selected-row-key="selectedShareId"
            @row-click="onShareRowClick"
          />

          <div class="mb-2 mt-4 text-xs font-medium text-ui-text">{{ t('page.permissionManagement.auditTitle') }}</div>
          <Table
            :columns="commandColumns"
            :rows="commandRows"
            :state="tableState"
            :error-message="tableErrorMessage"
            :caption="t('page.permissionManagement.auditTitle')"
            interactive-rows
            row-key="id"
            :selected-row-key="selectedCommandId"
            @row-click="onCommandRowClick"
          />
        </SectionCard>
      </template>

      <template #permission-detail>
        <SectionCard
          :title="t('page.permissionManagement.detailTitle')"
          :subtitle="selectedShare?.id ?? selectedCommand?.id ?? '-'"
        >
          <div class="space-y-3">
            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 text-xs font-medium text-ui-text">{{ t('page.permissionManagement.policyDetailTitle') }}</div>
              <div v-if="selectedShare" class="space-y-1 text-xs text-ui-muted">
                <div>{{ t('page.permissionManagement.fieldResourceType') }}: {{ selectedShare.resourceType }}</div>
                <div>{{ t('page.permissionManagement.fieldResourceId') }}: {{ selectedShare.resourceId }}</div>
                <div>{{ t('page.permissionManagement.fieldSubjectType') }}: {{ selectedShare.subjectType }}</div>
                <div>{{ t('page.permissionManagement.fieldSubjectId') }}: {{ selectedShare.subjectId }}</div>
                <div>{{ t('page.permissionManagement.fieldPermissions') }}: {{ selectedShare.permissions.join(', ') }}</div>
                <div>{{ t('page.permissionManagement.fieldCreatedBy') }}: {{ selectedShare.createdBy }}</div>
                <div>{{ t('page.permissionManagement.fieldCreatedAt') }}: {{ selectedShare.createdAt }}</div>
                <div>{{ t('page.permissionManagement.fieldExpiresAt') }}: {{ selectedShare.expiresAt || '-' }}</div>
              </div>
              <div v-else class="text-sm text-ui-muted">{{ t('page.permissionManagement.emptyPolicyDetail') }}</div>
            </div>

            <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
              <div class="mb-2 text-xs font-medium text-ui-text">{{ t('page.permissionManagement.auditDetailTitle') }}</div>
              <div v-if="selectedCommand" class="space-y-3">
                <div class="rounded-lg border border-ui-border bg-ui-bg p-3 text-xs text-ui-muted">
                  <div>{{ t('page.commands.fieldType') }}: {{ selectedCommand.commandType }}</div>
                  <div>{{ t('page.commands.fieldStatus') }}: {{ selectedCommand.status }}</div>
                  <div>{{ t('page.commands.fieldAcceptedAt') }}: {{ selectedCommand.acceptedAt }}</div>
                  <div>{{ t('page.commands.fieldOwner') }}: {{ selectedCommand.ownerId }}</div>
                </div>

                <div class="rounded-lg border border-ui-border bg-ui-bg p-3 text-xs">
                  <div class="mb-2 font-medium text-ui-text">payload</div>
                  <pre class="ui-scrollbar max-h-52 overflow-auto rounded border border-ui-border bg-ui-panel p-2">{{
                    formatJSON(selectedCommand.payload)
                  }}</pre>
                </div>

                <div class="rounded-lg border border-ui-border bg-ui-bg p-3 text-xs">
                  <div class="mb-2 font-medium text-ui-text">result</div>
                  <pre class="ui-scrollbar max-h-52 overflow-auto rounded border border-ui-border bg-ui-panel p-2">{{
                    formatJSON(selectedCommand.result || {})
                  }}</pre>
                </div>
              </div>
              <div v-else class="text-sm text-ui-muted">{{ t('page.permissionManagement.emptyDetail') }}</div>
            </div>
          </div>
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
import { createShare, deleteShare, listShares } from '@/api/shares'
import type { ApiError, CommandDTO, ShareCreateRequest, ShareDTO, ShareSubjectType } from '@/api/types'
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const commands = ref<CommandDTO[]>([])
const shares = ref<ShareDTO[]>([])
const selectedCommandId = ref<string | null>(null)
const selectedShareId = ref<string | null>(null)
const tableState = ref<TableState>('loading')
const apiError = ref<ApiError | null>(null)
const isActionBusy = ref(false)

const shareForm = reactive({
  resourceType: 'asset',
  resourceId: '',
  subjectType: 'user' as ShareSubjectType,
  subjectId: '',
  permissionsRaw: 'READ',
  expiresAt: '',
})

const windowPanes = computed(() => [
  { id: 'permission-overview', title: t('page.permissionManagement.overviewTitle') },
  { id: 'permission-detail', title: t('page.permissionManagement.detailTitle') },
])

const resourceTypeOptions = computed(() => [
  { value: 'asset', label: 'asset' },
  { value: 'command', label: 'command' },
])

const subjectTypeOptions = computed(() => [
  { value: 'user', label: 'user' },
  { value: 'role', label: 'role' },
])

const shareColumns = computed<TableColumn[]>(() => [
  { key: 'resourceType', label: t('page.permissionManagement.fieldResourceType'), width: '8rem' },
  { key: 'resourceId', label: t('page.permissionManagement.fieldResourceId'), mono: true, width: '14rem' },
  { key: 'subjectType', label: t('page.permissionManagement.fieldSubjectType'), width: '8rem' },
  { key: 'subjectId', label: t('page.permissionManagement.fieldSubjectId'), mono: true, width: '12rem' },
  { key: 'permissions', label: t('page.permissionManagement.fieldPermissions') },
  { key: 'id', label: 'id', mono: true, width: '15rem' },
])

const commandColumns = computed<TableColumn[]>(() => [
  { key: 'commandType', label: t('page.commands.fieldType') },
  { key: 'status', label: t('page.commands.fieldStatus'), width: '9rem' },
  { key: 'ownerId', label: t('page.commands.fieldOwner'), width: '11rem' },
  { key: 'id', label: t('page.commands.fieldCommandId'), mono: true, width: '17rem' },
])

const shareRows = computed(() =>
  shares.value.map((item) => ({
    id: item.id,
    resourceType: item.resourceType,
    resourceId: item.resourceId,
    subjectType: item.subjectType,
    subjectId: item.subjectId,
    permissions: item.permissions.join(', '),
  })),
)

const commandRows = computed(() =>
  commands.value.map((item) => ({
    id: item.id,
    commandType: item.commandType,
    status: item.status,
    ownerId: item.ownerId,
  })),
)

const selectedCommand = computed(() => commands.value.find((item) => item.id === selectedCommandId.value) ?? null)
const selectedShare = computed(() => shares.value.find((item) => item.id === selectedShareId.value) ?? null)

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

watch(
  shares,
  (items) => {
    if (!items.some((item) => item.id === selectedShareId.value)) {
      selectedShareId.value = items[0]?.id ?? null
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
    const [commandResp, shareResp] = await Promise.all([listCommands({ page: 1, pageSize: 400 }), listShares({ page: 1, pageSize: 400 })])
    commands.value = commandResp.items.filter((item) => item.commandType.startsWith('share.'))
    shares.value = shareResp.items
    tableState.value = 'ready'
  } catch (error) {
    commands.value = []
    shares.value = []
    apiError.value = asApiError(error)
    tableState.value = 'error'
  }
}

function onShareRowClick(payload: { rowKey: string }): void {
  selectedShareId.value = payload.rowKey
}

function onCommandRowClick(payload: { rowKey: string }): void {
  selectedCommandId.value = payload.rowKey
}

async function onRefresh(): Promise<void> {
  await loadData()
}

async function onGrantShare(): Promise<void> {
  const resourceId = shareForm.resourceId.trim()
  const subjectId = shareForm.subjectId.trim()
  const permissions = parsePermissions(shareForm.permissionsRaw)

  if (!resourceId || !subjectId || permissions.length === 0 || isActionBusy.value) {
    pushToast({
      title: t('page.permissionManagement.actionGrant'),
      message: t('page.permissionManagement.formInvalid'),
      tone: 'warn',
    })
    return
  }

  isActionBusy.value = true
  try {
    const request: ShareCreateRequest = {
      resourceType: shareForm.resourceType,
      resourceId,
      subjectType: shareForm.subjectType,
      subjectId,
      permissions,
      expiresAt: shareForm.expiresAt.trim() || undefined,
    }
    const response = await createShare(request)
    pushToast({
      title: t('page.permissionManagement.actionGrant'),
      message: `${t('page.permissionManagement.fieldCommandRef')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadData()
    if (typeof response.resource.id === 'string' && response.resource.id.trim()) {
      selectedShareId.value = response.resource.id
    }
    selectedCommandId.value = response.commandRef.commandId
  } catch (error) {
    notifyActionError(t('page.permissionManagement.actionGrant'), error)
  } finally {
    isActionBusy.value = false
  }
}

async function onRevokeShare(): Promise<void> {
  if (!selectedShareId.value || isActionBusy.value) {
    return
  }

  isActionBusy.value = true
  try {
    const response = await deleteShare(selectedShareId.value)
    pushToast({
      title: t('page.permissionManagement.actionRevoke'),
      message: `${t('page.permissionManagement.fieldCommandRef')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    selectedShareId.value = null
    await loadData()
    selectedCommandId.value = response.commandRef.commandId
  } catch (error) {
    notifyActionError(t('page.permissionManagement.actionRevoke'), error)
  } finally {
    isActionBusy.value = false
  }
}

function parsePermissions(raw: string): string[] {
  const set = new Set<string>()
  for (const part of raw.split(',')) {
    const permission = part.trim().toUpperCase()
    if (!permission) {
      continue
    }
    set.add(permission)
  }
  return Array.from(set)
}

function notifyActionError(title: string, error: unknown): void {
  const apiErr = asApiError(error)
  pushToast({
    title,
    message: `${apiErr.code}: ${t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {})}`,
    tone: 'error',
  })
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
