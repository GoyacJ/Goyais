<template>
  <div class="ui-surface overflow-hidden">
    <table class="w-full border-collapse text-left text-sm">
      <caption v-if="caption" class="sr-only">{{ caption }}</caption>
      <thead class="bg-ui-surface2 text-xs uppercase tracking-[0.05em] text-ui-muted">
        <tr>
          <th
            v-for="column in columns"
            :key="column.key"
            class="border-b border-ui-borderSubtle px-[var(--ui-list-row-px)] py-2 font-semibold"
            :class="resolveAlignClass(column.align)"
            :style="column.width ? { width: column.width } : undefined"
          >
            {{ column.label }}
          </th>
        </tr>
      </thead>

      <tbody v-if="state === 'ready' && rows.length > 0">
        <tr
          v-for="(row, index) in rows"
          :key="resolveRowKey(row, index)"
          class="ui-table-row ui-list-row"
          :class="[
            interactiveRows ? 'ui-pressable ui-list-row--interactive ui-table-row--interactive' : '',
            isRowSelected(row, index) ? 'ui-list-row--selected ui-table-row--selected' : '',
            interactiveRows ? 'ui-list-row--focus' : '',
          ]"
          :role="interactiveRows ? 'button' : undefined"
          :tabindex="interactiveRows ? 0 : undefined"
          :aria-selected="interactiveRows ? isRowSelected(row, index) : undefined"
          @click="onRowActivate(row, index)"
          @keydown.enter.prevent="onRowActivate(row, index)"
          @keydown.space.prevent="onRowActivate(row, index)"
        >
          <td
            v-for="column in columns"
            :key="column.key"
            class="align-middle"
            :class="[column.mono ? 'ui-monospace text-[13px]' : '', resolveAlignClass(column.align), column.cellClass ?? '']"
          >
            <slot
              :name="`cell-${column.key}`"
              :row="row"
              :value="row[column.key]"
              :column="column"
              :index="index"
              :selected="isRowSelected(row, index)"
            >
              {{ row[column.key] ?? '-' }}
            </slot>
          </td>
        </tr>
      </tbody>

      <tbody v-else-if="state === 'loading'">
        <tr
          v-for="index in loadingRows"
          :key="`loading-${index}`"
          class="ui-table-row ui-list-row"
        >
          <td v-for="column in columns" :key="`${column.key}-${index}`" class="align-middle">
            <div class="h-2.5 w-2/3 animate-pulse rounded bg-ui-borderSubtle" />
          </td>
        </tr>
      </tbody>

      <tbody v-else>
        <tr>
          <td :colspan="columns.length" class="px-3 py-7 text-center text-sm text-ui-muted">
            <span v-if="state === 'error'">{{ resolvedErrorMessage }}</span>
            <span v-else>{{ resolvedEmptyMessage }}</span>
          </td>
        </tr>
      </tbody>
    </table>
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
import type { TableState } from '@/design-system/types'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

type TableRow = Record<string, unknown>

export interface TableColumn {
  key: string
  label: string
  mono?: boolean
  align?: 'left' | 'center' | 'right'
  width?: string
  cellClass?: string
}

const emit = defineEmits<{
  (e: 'rowClick', payload: { row: TableRow; index: number; rowKey: string }): void
}>()

function onRowActivate(row: TableRow, index: number): void {
  if (!props.interactiveRows) {
    return
  }
  emit('rowClick', { row, index, rowKey: resolveRowKey(row, index) })
}

const props = withDefaults(
  defineProps<{
    columns: TableColumn[]
    rows: TableRow[]
    state?: TableState
    loadingRows?: number
    emptyMessage?: string
    errorMessage?: string
    caption?: string
    interactiveRows?: boolean
    selectedRowIndex?: number
    rowKey?: string | ((row: TableRow, index: number) => string)
    selectedRowKey?: string | null
  }>(),
  {
    state: 'ready',
    loadingRows: 4,
    emptyMessage: '',
    errorMessage: '',
    caption: '',
    interactiveRows: false,
    selectedRowIndex: -1,
    rowKey: '',
    selectedRowKey: null,
  },
)

const { t } = useI18n({ useScope: 'global' })

const resolvedEmptyMessage = computed(() => props.emptyMessage || t('common.empty'))
const resolvedErrorMessage = computed(() => props.errorMessage || t('error.common.internal'))

function resolveAlignClass(align?: 'left' | 'center' | 'right'): string {
  if (align === 'center') {
    return 'text-center'
  }

  if (align === 'right') {
    return 'text-right'
  }

  return 'text-left'
}

function resolveRowKey(row: TableRow, index: number): string {
  if (typeof props.rowKey === 'function') {
    return String(props.rowKey(row, index))
  }

  if (typeof props.rowKey === 'string' && props.rowKey.length > 0) {
    const fromRow = row[props.rowKey]
    if (typeof fromRow === 'string' || typeof fromRow === 'number') {
      return String(fromRow)
    }
  }

  return String(index)
}

function isRowSelected(row: TableRow, index: number): boolean {
  if (props.selectedRowKey !== null && props.selectedRowKey !== undefined && props.selectedRowKey !== '') {
    return resolveRowKey(row, index) === props.selectedRowKey
  }

  return props.selectedRowIndex === index
}
</script>
