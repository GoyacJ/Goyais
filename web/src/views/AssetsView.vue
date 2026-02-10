<template>
  <section class="ui-page">
    <PageHeader :title="t('page.assets.title')" :subtitle="t('page.assets.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isRefreshing" @click="onRefresh">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
        <Button :disabled="isUploading" @click="onUploadClick">
          <Icon name="upload" :size="14" decorative />
          {{ t('page.assets.actionUpload') }}
        </Button>
      </template>
    </PageHeader>

    <input
      ref="fileInput"
      type="file"
      class="hidden"
      @change="onUploadSelected"
    />

    <WindowBoard route-key="assets" :panes="windowPanes">
      <template #filters>
        <SectionCard :title="t('page.assets.filtersTitle')" :subtitle="t('page.assets.filtersSubtitle')">
          <div class="grid gap-3 md:grid-cols-[1.4fr_1fr_1fr]">
            <Input v-model="searchQuery" :placeholder="t('page.assets.searchPlaceholder')" />
            <Select v-model="typeFilter" :options="typeOptions" />
            <Select v-model="visibilityFilter" :options="visibilityOptions" />
          </div>
        </SectionCard>
      </template>

      <template #list>
        <SectionCard :title="t('page.assets.listTitle')" :subtitle="listSubtitle">
          <EmptyState
            v-if="tableState === 'ready' && filteredAssets.length === 0"
            variant="assets-empty"
            :title="t('empty_state.assets.title')"
            :description="t('empty_state.assets.description')"
          />
          <Table
            v-else
            :columns="columns"
            :rows="tableRows"
            :state="tableState"
            :error-message="tableErrorMessage"
            :caption="t('page.assets.listTitle')"
            interactive-rows
            row-key="assetId"
            :selected-row-key="selectedAssetId"
            @row-click="onRowClick"
          >
            <template #cell-name="{ row }">
              <div class="flex flex-col gap-0.5 leading-tight">
                <span class="text-sm text-ui-fg">{{ String(row.name) }}</span>
                <span class="ui-monospace text-[11px] text-ui-muted">{{ String(row.assetId) }}</span>
              </div>
            </template>
          </Table>
        </SectionCard>
      </template>

      <template #detail>
        <SectionCard :title="t('page.assets.detailTitle')" :subtitle="selectedAsset?.assetId ?? '-'">
          <div v-if="selectedAsset" class="ui-detail-block">
            <header class="ui-detail-header">
              <div class="min-w-0">
                <p class="truncate text-sm font-semibold text-ui-fg">{{ selectedAsset.name }}</p>
                <p class="mt-1 text-xs text-ui-muted">{{ selectedAsset.assetId }}</p>
              </div>
              <span class="ui-monospace text-xs text-ui-muted">{{ selectedAsset.visibility }}</span>
            </header>

            <dl class="ui-detail-meta text-xs text-ui-muted md:grid-cols-2">
              <div>
                <dt>{{ t('page.assets.fieldType') }}</dt>
                <dd class="mt-1 text-sm text-ui-fg">{{ selectedAsset.type }}</dd>
              </div>
              <div>
                <dt>{{ t('page.assets.fieldSize') }}</dt>
                <dd class="mt-1 text-sm text-ui-fg">{{ selectedAsset.size }}</dd>
              </div>
              <div>
                <dt>{{ t('page.assets.fieldVisibility') }}</dt>
                <dd class="mt-1 text-sm text-ui-fg">{{ selectedAsset.visibility }}</dd>
              </div>
              <div class="md:col-span-2">
                <dt>{{ t('page.assets.fieldCreatedAt') }}</dt>
                <dd class="ui-monospace mt-1 text-ui-fg">{{ selectedAsset.createdAt }}</dd>
              </div>
              <div class="md:col-span-2 ui-detail-mono">
                <dt>{{ t('page.assets.fieldUri') }}</dt>
                <dd class="ui-monospace mt-1 break-all text-ui-fg">{{ selectedAsset.uri }}</dd>
              </div>
              <div class="md:col-span-2 ui-detail-mono">
                <dt>{{ t('page.assets.fieldHash') }}</dt>
                <dd class="ui-monospace mt-1 break-all text-ui-fg">{{ selectedAsset.hash }}</dd>
              </div>
              <div>
                <dt>{{ t('page.assets.fieldOwner') }}</dt>
                <dd class="ui-monospace mt-1 text-ui-fg">{{ selectedAsset.owner }}</dd>
              </div>
            </dl>
          </div>
          <EmptyState
            v-else
            variant="assets-empty"
            :title="t('empty_state.assets.detailTitle')"
            :description="t('empty_state.assets.detailDescription')"
          />
        </SectionCard>
      </template>
    </WindowBoard>
  </section>
</template>

<script setup lang="ts">
import { createAsset, listAssets } from '@/api/assets'
import { ApiHttpError, isMockEnabled } from '@/api/http'
import type { ApiError, AssetDTO } from '@/api/types'
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import { useToast } from '@/composables/useToast'
import type { TableState, Visibility } from '@/design-system/types'
import { mockAssets, type MockAsset } from '@/mocks/assets'
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

interface AssetViewItem {
  assetId: string
  name: string
  type: string
  size: string
  sizeBytes: number
  visibility: Visibility
  createdAt: string
  uri: string
  hash: string
  owner: string
}

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const searchQuery = ref('')
const typeFilter = ref('all')
const visibilityFilter = ref('all')
const selectedAssetId = ref<string | null>(null)
const tableState = ref<TableState>('loading')
const assets = ref<AssetViewItem[]>([])
const apiError = ref<ApiError | null>(null)
const isUploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const useMock = isMockEnabled()

const isRefreshing = computed(() => tableState.value === 'loading')

const windowPanes = computed(() => [
  { id: 'filters', title: t('page.assets.filtersTitle') },
  { id: 'list', title: t('page.assets.listTitle') },
  { id: 'detail', title: t('page.assets.detailTitle') },
])

const columns = computed<TableColumn[]>(() => [
  { key: 'name', label: t('page.assets.fieldName') },
  { key: 'type', label: t('page.assets.fieldType') },
  { key: 'size', label: t('page.assets.fieldSize'), align: 'right', width: '8rem' },
  { key: 'visibility', label: t('page.assets.fieldVisibility'), align: 'center', width: '9rem' },
  { key: 'createdAt', label: t('page.assets.fieldCreatedAt'), mono: true, width: '13rem' },
])

const typeOptions = computed(() => [
  { value: 'all', label: t('common.all') },
  ...Array.from(new Set(assets.value.map((item) => item.type))).map((type) => ({
    value: type,
    label: type,
  })),
])

const visibilityOptions = computed(() => [
  { value: 'all', label: t('common.all') },
  ...Array.from(new Set(assets.value.map((item) => item.visibility))).map((visibility) => ({
    value: visibility,
    label: visibility,
  })),
])

const filteredAssets = computed(() =>
  assets.value.filter((item) => {
    const matchType = typeFilter.value === 'all' || item.type === typeFilter.value
    const matchVisibility = visibilityFilter.value === 'all' || item.visibility === visibilityFilter.value
    const q = searchQuery.value.trim().toLowerCase()
    const matchQuery =
      q.length === 0 ||
      item.assetId.toLowerCase().includes(q) ||
      item.name.toLowerCase().includes(q) ||
      item.uri.toLowerCase().includes(q)

    return matchType && matchVisibility && matchQuery
  }),
)

const selectedAsset = computed(() =>
  filteredAssets.value.find((item) => item.assetId === selectedAssetId.value),
)

const tableRows = computed(() =>
  filteredAssets.value.map((item) => ({
    assetId: item.assetId,
    name: item.name,
    type: item.type,
    size: item.size,
    visibility: item.visibility,
    createdAt: item.createdAt,
  })),
)

const listSubtitle = computed(() => {
  if (tableState.value === 'loading') {
    return t('common.loading')
  }

  return String(filteredAssets.value.length)
})

const tableErrorMessage = computed(() => {
  if (!apiError.value) {
    return t('error.common.internal')
  }

  return t(apiError.value.messageKey || 'error.common.internal', apiError.value.details ?? {})
})

watch(
  filteredAssets,
  (items) => {
    if (!items.some((item) => item.assetId === selectedAssetId.value)) {
      selectedAssetId.value = items[0]?.assetId ?? null
    }
  },
  { immediate: true },
)

onMounted(() => {
  void loadAssets()
})

async function loadAssets(): Promise<void> {
  tableState.value = 'loading'
  apiError.value = null

  try {
    if (useMock) {
      assets.value = mockAssets.map(toAssetViewFromMock)
    } else {
      const response = await listAssets({ page: 1, pageSize: 200 })
      assets.value = response.items.map(toAssetViewFromApi)
    }
    tableState.value = 'ready'
  } catch (error) {
    apiError.value = asApiError(error)
    assets.value = []
    tableState.value = 'error'
  }
}

async function onRefresh(): Promise<void> {
  await loadAssets()
}

function onRowClick(payload: { rowKey: string }): void {
  selectedAssetId.value = payload.rowKey
}

function onUploadClick(): void {
  if (useMock) {
    pushToast({
      title: t('page.assets.actionUpload'),
      message: t('common.placeholderAction', { value: t('page.assets.actionUpload') }),
      tone: 'info',
    })
    return
  }

  fileInput.value?.click()
}

async function onUploadSelected(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) {
    return
  }

  isUploading.value = true
  try {
    const response = await createAsset({
      file,
      name: file.name,
      type: file.type,
      visibility: 'PRIVATE',
    })

    pushToast({
      title: t('page.assets.actionUpload'),
      message: `${t('page.commands.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })

    await loadAssets()

    const resourceID = typeof response.resource.id === 'string' ? response.resource.id : ''
    if (resourceID) {
      selectedAssetId.value = resourceID
    }
  } catch (error) {
    const apiErr = asApiError(error)
    pushToast({
      title: t('page.assets.actionUpload'),
      message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
      tone: 'error',
    })
  } finally {
    isUploading.value = false
    target.value = ''
  }
}

function toAssetViewFromApi(item: AssetDTO): AssetViewItem {
  return {
    assetId: item.id,
    name: item.name,
    type: item.type,
    size: formatBytes(item.size),
    sizeBytes: item.size,
    visibility: item.visibility,
    createdAt: item.createdAt,
    uri: item.uri,
    hash: item.hash,
    owner: item.ownerId,
  }
}

function toAssetViewFromMock(item: MockAsset): AssetViewItem {
  const numericSize = parseMockSizeToBytes(item.size)
  return {
    assetId: item.assetId,
    name: item.name,
    type: item.type,
    size: item.size,
    sizeBytes: numericSize,
    visibility: item.visibility,
    createdAt: item.createdAt,
    uri: item.uri,
    hash: item.hash,
    owner: item.owner,
  }
}

function parseMockSizeToBytes(size: string): number {
  const raw = size.trim().toUpperCase()
  const value = Number.parseFloat(raw)
  if (!Number.isFinite(value) || value <= 0) {
    return 0
  }
  if (raw.endsWith('KB')) {
    return Math.round(value * 1024)
  }
  if (raw.endsWith('MB')) {
    return Math.round(value * 1024 * 1024)
  }
  if (raw.endsWith('GB')) {
    return Math.round(value * 1024 * 1024 * 1024)
  }
  return Math.round(value)
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = bytes
  let unitIndex = 0
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }

  const fixed = size >= 10 || unitIndex === 0 ? size.toFixed(0) : size.toFixed(1)
  return `${fixed} ${units[unitIndex]}`
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
</script>
