<template>
  <section class="ui-page">
    <PageHeader :title="t('page.assets.title')" :subtitle="t('page.assets.subtitle')">
      <template #actions>
        <Button variant="secondary">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
        <Button @click="onUploadPlaceholder">
          <Icon name="upload" :size="14" decorative />
          {{ t('page.assets.actionUpload') }}
        </Button>
      </template>
    </PageHeader>

    <SectionCard :title="t('page.assets.filtersTitle')" :subtitle="t('page.assets.filtersSubtitle')">
      <div class="grid gap-3 md:grid-cols-[1.4fr_1fr_1fr]">
        <Input v-model="searchQuery" :placeholder="t('page.assets.searchPlaceholder')" />
        <Select v-model="typeFilter" :options="typeOptions" />
        <Select v-model="visibilityFilter" :options="visibilityOptions" />
      </div>
    </SectionCard>

    <div class="grid gap-[var(--ui-page-gap)] xl:grid-cols-[1.35fr_1fr]">
      <SectionCard :title="t('page.assets.listTitle')" :subtitle="`${filteredAssets.length}`">
        <EmptyState
          v-if="filteredAssets.length === 0"
          variant="assets-empty"
          :title="t('empty_state.assets.title')"
          :description="t('empty_state.assets.description')"
        />
        <Table
          v-else
          :columns="columns"
          :rows="tableRows"
          :caption="t('page.assets.listTitle')"
          interactive-rows
          :selected-row-index="selectedRowIndex"
          @row-click="onRowClick"
        />
      </SectionCard>

      <SectionCard :title="t('page.assets.detailTitle')" :subtitle="selectedAsset?.assetId ?? '-'">
        <div v-if="selectedAsset" class="space-y-3">
          <dl class="grid gap-3 text-xs text-ui-muted md:grid-cols-2">
            <div>
              <dt>{{ t('page.assets.fieldName') }}</dt>
              <dd class="mt-1 text-sm text-ui-fg">{{ selectedAsset.name }}</dd>
            </div>
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
            <div class="md:col-span-2">
              <dt>URI</dt>
              <dd class="ui-monospace mt-1 break-all text-ui-fg">{{ selectedAsset.uri }}</dd>
            </div>
            <div class="md:col-span-2">
              <dt>Hash</dt>
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
    </div>
  </section>
</template>

<script setup lang="ts">
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import { useToast } from '@/composables/useToast'
import { mockAssets } from '@/mocks/assets'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const searchQuery = ref('')
const typeFilter = ref('all')
const visibilityFilter = ref('all')
const selectedAssetId = ref<string | null>(mockAssets[0]?.assetId ?? null)

const columns = computed<TableColumn[]>(() => [
  { key: 'name', label: t('page.assets.fieldName') },
  { key: 'type', label: t('page.assets.fieldType') },
  { key: 'size', label: t('page.assets.fieldSize') },
  { key: 'visibility', label: t('page.assets.fieldVisibility') },
  { key: 'createdAt', label: t('page.assets.fieldCreatedAt'), mono: true },
])

const typeOptions = computed(() => [
  { value: 'all', label: t('common.all') },
  ...Array.from(new Set(mockAssets.map((item) => item.type))).map((type) => ({
    value: type,
    label: type,
  })),
])

const visibilityOptions = computed(() => [
  { value: 'all', label: t('common.all') },
  ...Array.from(new Set(mockAssets.map((item) => item.visibility))).map((visibility) => ({
    value: visibility,
    label: visibility,
  })),
])

const filteredAssets = computed(() =>
  mockAssets.filter((item) => {
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

watch(
  filteredAssets,
  (items) => {
    if (!items.some((item) => item.assetId === selectedAssetId.value)) {
      selectedAssetId.value = items[0]?.assetId ?? null
    }
  },
  { immediate: true },
)

const selectedAsset = computed(() =>
  filteredAssets.value.find((item) => item.assetId === selectedAssetId.value),
)

const selectedRowIndex = computed(() =>
  filteredAssets.value.findIndex((item) => item.assetId === selectedAssetId.value),
)

const tableRows = computed(() =>
  filteredAssets.value.map((item) => ({
    name: item.name,
    type: item.type,
    size: item.size,
    visibility: item.visibility,
    createdAt: item.createdAt,
  })),
)

function onRowClick(payload: { index: number }): void {
  const target = filteredAssets.value[payload.index]
  if (target) {
    selectedAssetId.value = target.assetId
  }
}

function onUploadPlaceholder(): void {
  pushToast({
    title: t('page.assets.actionUpload'),
    message: t('page.assets.uploadPlaceholderHint'),
    tone: 'info',
  })
}
</script>
