<template>
  <div class="space-y-3">
    <header class="flex flex-wrap items-center justify-between gap-2">
      <div class="grid gap-1 text-xs text-ui-muted md:grid-cols-3 md:gap-3">
        <span>{{ t('page.canvas.registryCapabilitiesCount', { count: capabilities.length }) }}</span>
        <span>{{ t('page.canvas.registryAlgorithmsCount', { count: algorithms.length }) }}</span>
        <span>{{ t('page.canvas.registryProvidersCount', { count: providers.length }) }}</span>
      </div>
      <Button variant="ghost" :disabled="busy" @click="$emit('refresh')">{{ t('common.refresh') }}</Button>
    </header>

    <Table
      :columns="columns"
      :rows="rows"
      :state="tableState"
      :caption="t('page.canvas.registryTitle')"
    >
      <template #cell-actions="{ row }">
        <Button
          :disabled="busy || Boolean(row.runDisabled)"
          @click="$emit('runAlgorithm', String(row.id))"
        >
          {{ t('page.canvas.actionRunAlgorithm') }}
        </Button>
      </template>
    </Table>
  </div>
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
import type { AlgorithmDTO, CapabilityDTO, ProviderDTO } from '@/api/types'
import Button from '@/components/ui/Button.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import type { TableState } from '@/design-system/types'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    capabilities: CapabilityDTO[]
    algorithms: AlgorithmDTO[]
    providers: ProviderDTO[]
    busy?: boolean
    loading?: boolean
  }>(),
  {
    busy: false,
    loading: false,
  },
)

defineEmits<{
  (e: 'runAlgorithm', algorithmId: string): void
  (e: 'refresh'): void
}>()

const { t } = useI18n({ useScope: 'global' })

const columns = computed<TableColumn[]>(() => [
  { key: 'name', label: t('page.canvas.fieldAlgorithmName') },
  { key: 'version', label: t('page.canvas.fieldVersion'), width: '8rem' },
  { key: 'templateRef', label: t('page.canvas.fieldTemplateRef'), mono: true, width: '16rem' },
  { key: 'actions', label: t('page.canvas.fieldActions'), width: '8rem', align: 'center' },
])

const rows = computed(() =>
  props.algorithms.map((item) => ({
    id: item.id,
    name: item.name,
    version: item.version,
    templateRef: item.templateRef,
    runDisabled: !item.id,
  })),
)

const tableState = computed<TableState>(() => {
  if (props.loading) {
    return 'loading'
  }
  return rows.value.length > 0 ? 'ready' : 'empty'
})
</script>
