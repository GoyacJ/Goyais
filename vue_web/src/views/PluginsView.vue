<template>
  <section class="ui-page">
    <PageHeader :title="t('page.plugins.title')" :subtitle="t('page.plugins.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isLoading" @click="onRefresh">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <WindowBoard route-key="plugins" :panes="windowPanes">
      <template #plugin-catalog>
        <SectionCard :title="t('page.plugins.catalogTitle')" :subtitle="catalogSubtitle">
          <div class="ui-surface mb-3 p-3">
            <h3 class="text-sm font-semibold text-ui-fg">{{ t('page.plugins.uploadTitle') }}</h3>
            <div class="mt-3 grid gap-2">
              <Input v-model="uploadName" :placeholder="t('page.plugins.fieldPackageName')" />
              <div class="grid gap-2 md:grid-cols-2">
                <Input v-model="uploadVersion" :placeholder="t('page.plugins.fieldVersion')" />
                <Select v-model="uploadPackageType" :options="packageTypeOptions" />
              </div>
              <Textarea v-model="uploadManifestText" :placeholder="t('page.plugins.fieldManifest')" />
              <div class="grid gap-2 md:grid-cols-2">
                <Select v-model="uploadVisibility" :options="visibilityOptions" />
                <Button :disabled="isUploading" @click="onUploadPackage">
                  {{ t('page.plugins.actionUpload') }}
                </Button>
              </div>
            </div>
          </div>

          <div v-if="tableState === 'error'" class="ui-surface p-3 text-sm text-ui-danger">
            {{ tableErrorMessage }}
          </div>
          <EmptyState
            v-else-if="tableState === 'ready' && packageCards.length === 0"
            variant="commands-empty"
            :title="t('page.plugins.catalogEmptyTitle')"
            :description="t('page.plugins.catalogEmptyDescription')"
          />
          <div v-else class="grid gap-[var(--ui-page-gap)] md:grid-cols-2">
            <PluginPackageCard
              v-for="item in packageCards"
              :key="item.id"
              :item="item"
              :busy="pendingInstallActionId === item.installId || pendingPackageActionId === item.id"
              @install="onInstallPackage"
              @enable="onEnableInstall"
              @disable="onDisableInstall"
              @rollback="onRollbackInstall"
              @upgrade="onUpgradeInstall"
              @download="onDownloadPackage"
            />
          </div>
        </SectionCard>
      </template>

      <template #plugin-activity>
        <SectionCard :title="t('page.plugins.activityTitle')" :subtitle="t('page.plugins.activitySubtitle')">
          <PluginCommandTimeline :events="pluginTimeline" />
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
  downloadPluginPackage,
  disablePluginInstall,
  enablePluginInstall,
  installPlugin,
  listPluginPackages,
  rollbackPluginInstall,
  upgradePluginInstall,
  uploadPluginPackage,
} from '@/api/plugins'
import type { ApiError, ApiObject, CommandDTO, PluginPackageDTO } from '@/api/types'
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import PluginCommandTimeline, { type PluginCommandTimelineEvent } from '@/components/runtime/PluginCommandTimeline.vue'
import PluginPackageCard, { type PluginPackageCardItem } from '@/components/runtime/PluginPackageCard.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Textarea from '@/components/ui/Textarea.vue'
import { useToast } from '@/composables/useToast'
import type { TableState } from '@/design-system/types'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

interface InstallSnapshot {
  installId: string
  packageId: string
  installStatus: string
  commandId: string
  acceptedAt: string
}

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const useMock = isMockEnabled()
const tableState = ref<TableState>('loading')
const apiError = ref<ApiError | null>(null)
const isUploading = ref(false)
const pendingInstallActionId = ref<string | null>(null)
const pendingPackageActionId = ref<string | null>(null)
const packages = ref<PluginPackageDTO[]>([])
const pluginCommands = ref<CommandDTO[]>([])

const uploadName = ref('demo-plugin')
const uploadVersion = ref('1.0.0')
const uploadPackageType = ref('tool-provider')
const uploadManifestText = ref('{"entry":"main"}')
const uploadVisibility = ref('PRIVATE')

const packageTypeOptions = computed(() => [
  { value: 'tool-provider', label: 'tool-provider' },
  { value: 'skill-pack', label: 'skill-pack' },
  { value: 'algo-pack', label: 'algo-pack' },
  { value: 'mcp-provider', label: 'mcp-provider' },
])

const visibilityOptions = computed(() => [
  { value: 'PRIVATE', label: 'PRIVATE' },
  { value: 'WORKSPACE', label: 'WORKSPACE' },
])

const isLoading = computed(() => tableState.value === 'loading')

const windowPanes = computed(() => [
  { id: 'plugin-catalog', title: t('page.plugins.catalogTitle') },
  { id: 'plugin-activity', title: t('page.plugins.activityTitle') },
])

const catalogSubtitle = computed(() => {
  if (tableState.value === 'loading') {
    return t('common.loading')
  }
  return String(packageCards.value.length)
})

const tableErrorMessage = computed(() => {
  if (!apiError.value) {
    return t('error.common.internal')
  }
  return t(apiError.value.messageKey || 'error.common.internal', apiError.value.details ?? {})
})

const installSnapshotsByPackage = computed<Record<string, InstallSnapshot>>(() => {
  const snapshots: Record<string, InstallSnapshot> = {}
  for (const cmd of pluginCommands.value) {
    const install = getInstallResult(cmd.result)
    if (!install) {
      continue
    }
    if (snapshots[install.packageId]) {
      continue
    }
    snapshots[install.packageId] = {
      installId: install.id,
      packageId: install.packageId,
      installStatus: install.status,
      commandId: cmd.id,
      acceptedAt: cmd.acceptedAt,
    }
  }
  return snapshots
})

const packageCards = computed<PluginPackageCardItem[]>(() =>
  packages.value.map((item) => {
    const install = installSnapshotsByPackage.value[item.id]
    return {
      id: item.id,
      name: item.name,
      version: item.version,
      packageType: item.packageType,
      installId: install?.installId ?? null,
      installStatus: install?.installStatus ?? 'uninstalled',
      lastCommandId: install?.commandId ?? null,
    }
  }),
)

const pluginTimeline = computed<PluginCommandTimelineEvent[]>(() =>
  pluginCommands.value.map((cmd) => ({
    commandId: cmd.id,
    commandType: cmd.commandType,
    acceptedAt: cmd.acceptedAt,
    status: cmd.status,
    summary: summarizePluginCommand(cmd),
  })),
)

onMounted(() => {
  void loadData()
})

async function loadData(): Promise<void> {
  tableState.value = 'loading'
  apiError.value = null

  try {
    if (useMock) {
      packages.value = []
      pluginCommands.value = []
      tableState.value = 'ready'
      return
    }

    const [packageResp, commandResp] = await Promise.all([
      listPluginPackages({ page: 1, pageSize: 200 }),
      listCommands({ page: 1, pageSize: 400 }),
    ])

    packages.value = packageResp.items
    pluginCommands.value = commandResp.items.filter((item) => item.commandType.startsWith('plugin.'))
    tableState.value = 'ready'
  } catch (error) {
    apiError.value = asApiError(error)
    packages.value = []
    pluginCommands.value = []
    tableState.value = 'error'
  }
}

async function onRefresh(): Promise<void> {
  await loadData()
}

async function onUploadPackage(): Promise<void> {
  if (isUploading.value) {
    return
  }
  if (useMock) {
    pushToast({
      title: t('page.plugins.actionUpload'),
      message: t('common.placeholderAction', { value: t('page.plugins.actionUpload') }),
      tone: 'info',
    })
    return
  }

  isUploading.value = true
  try {
    const manifest = parseManifest(uploadManifestText.value)
    const response = await uploadPluginPackage({
      name: uploadName.value.trim(),
      version: uploadVersion.value.trim(),
      packageType: uploadPackageType.value,
      visibility: uploadVisibility.value as 'PRIVATE' | 'WORKSPACE' | 'TENANT' | 'PUBLIC',
      manifest,
    })

    pushToast({
      title: t('page.plugins.actionUpload'),
      message: `${t('page.plugins.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadData()
  } catch (error) {
    const apiErr = asApiError(error)
    pushToast({
      title: t('page.plugins.actionUpload'),
      message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
      tone: 'error',
    })
  } finally {
    isUploading.value = false
  }
}

async function onInstallPackage(packageId: string): Promise<void> {
  if (useMock) {
    return
  }
  pendingPackageActionId.value = packageId
  try {
    const response = await installPlugin({ packageId, scope: 'workspace' })
    pushToast({
      title: t('page.plugins.actionInstall'),
      message: `${t('page.plugins.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadData()
  } catch (error) {
    notifyActionError(t('page.plugins.actionInstall'), error)
  } finally {
    pendingPackageActionId.value = null
  }
}

async function onEnableInstall(installId: string): Promise<void> {
  await runInstallAction(installId, t('common.enable'), enablePluginInstall)
}

async function onDisableInstall(installId: string): Promise<void> {
  await runInstallAction(installId, t('common.disable'), disablePluginInstall)
}

async function onRollbackInstall(installId: string): Promise<void> {
  await runInstallAction(installId, t('page.plugins.actionRollback'), rollbackPluginInstall)
}

async function onUpgradeInstall(installId: string): Promise<void> {
  await runInstallAction(installId, t('page.plugins.actionUpgrade'), upgradePluginInstall)
}

async function onDownloadPackage(packageId: string): Promise<void> {
  if (useMock) {
    return
  }
  pendingPackageActionId.value = packageId
  try {
    const result = await downloadPluginPackage(packageId)
    pushToast({
      title: t('page.plugins.actionDownload'),
      message: `${result.filename} (${result.content.length} bytes)`,
      tone: 'success',
    })
  } catch (error) {
    notifyActionError(t('page.plugins.actionDownload'), error)
  } finally {
    pendingPackageActionId.value = null
  }
}

async function runInstallAction(
  installId: string,
  title: string,
  action: (installId: string) => Promise<{ commandRef: { commandId: string } }>,
): Promise<void> {
  if (useMock) {
    return
  }
  pendingInstallActionId.value = installId
  try {
    const response = await action(installId)
    pushToast({
      title,
      message: `${t('page.plugins.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadData()
  } catch (error) {
    notifyActionError(title, error)
  } finally {
    pendingInstallActionId.value = null
  }
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

function parseManifest(raw: string): ApiObject {
  const input = raw.trim()
  if (input.length === 0) {
    return {}
  }
  try {
    const parsed = JSON.parse(input)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as ApiObject
    }
  } catch {
    // fall through to default
  }
  return {}
}

function summarizePluginCommand(cmd: CommandDTO): string {
  if (cmd.error?.messageKey) {
    return t(cmd.error.messageKey, cmd.error.details ?? {})
  }
  const install = getInstallResult(cmd.result)
  if (install) {
    return `${install.packageId} -> ${install.status}`
  }
  return cmd.commandType
}

function getInstallResult(result: CommandDTO['result']): { id: string; packageId: string; status: string } | null {
  if (!result || typeof result !== 'object') {
    return null
  }
  const install = (result as ApiObject).install
  if (!install || typeof install !== 'object') {
    return null
  }
  const installObj = install as ApiObject
  const id = typeof installObj.id === 'string' ? installObj.id : ''
  const packageId = typeof installObj.packageId === 'string' ? installObj.packageId : ''
  const status = typeof installObj.status === 'string' ? installObj.status : ''
  if (!id || !packageId || !status) {
    return null
  }
  return { id, packageId, status }
}
</script>
