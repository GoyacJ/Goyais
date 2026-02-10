<template>
  <div class="ui-surface overflow-hidden">
    <table class="w-full border-collapse text-left text-sm">
      <caption v-if="caption" class="sr-only">{{ caption }}</caption>
      <thead class="bg-ui-surface2 text-xs uppercase tracking-[0.05em] text-ui-muted">
        <tr>
          <th
            v-for="column in columns"
            :key="column.key"
            class="border-b border-ui-borderSubtle px-3 py-2 font-semibold"
          >
            {{ column.label }}
          </th>
        </tr>
      </thead>

      <tbody v-if="state === 'ready' && rows.length > 0">
        <tr
          v-for="(row, index) in rows"
          :key="index"
          class="ui-table-row border-b border-ui-borderSubtle last:border-b-0"
          :class="[
            interactiveRows ? 'ui-pressable ui-table-row--interactive' : '',
            selectedRowIndex === index ? 'ui-table-row--selected' : '',
          ]"
          :role="interactiveRows ? 'button' : undefined"
          :tabindex="interactiveRows ? 0 : undefined"
          :aria-selected="interactiveRows ? selectedRowIndex === index : undefined"
          @click="onRowActivate(row, index)"
          @keydown.enter.prevent="onRowActivate(row, index)"
          @keydown.space.prevent="onRowActivate(row, index)"
        >
          <td
            v-for="column in columns"
            :key="column.key"
            class="px-3 align-middle"
            :class="column.mono ? 'ui-monospace text-[13px]' : ''"
          >
            {{ row[column.key] ?? '-' }}
          </td>
        </tr>
      </tbody>

      <tbody v-else-if="state === 'loading'">
        <tr
          v-for="index in loadingRows"
          :key="`loading-${index}`"
          class="ui-table-row border-b border-ui-borderSubtle last:border-b-0"
        >
          <td v-for="column in columns" :key="`${column.key}-${index}`" class="px-3 align-middle">
            <div class="h-2.5 w-2/3 animate-pulse rounded bg-ui-border-subtle" />
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
import type { TableState } from '@/design-system/types'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export interface TableColumn {
  key: string
  label: string
  mono?: boolean
}

const emit = defineEmits<{
  (e: 'rowClick', payload: { row: Record<string, string | number>; index: number }): void
}>()

function onRowActivate(row: Record<string, string | number>, index: number): void {
  if (!props.interactiveRows) {
    return
  }
  emit('rowClick', { row, index })
}
const props = withDefaults(
  defineProps<{
    columns: TableColumn[]
    rows: Array<Record<string, string | number>>
    state?: TableState
    loadingRows?: number
    emptyMessage?: string
    errorMessage?: string
    caption?: string
    interactiveRows?: boolean
    selectedRowIndex?: number
  }>(),
  {
    state: 'ready',
    loadingRows: 4,
    emptyMessage: '',
    errorMessage: '',
    caption: '',
    interactiveRows: false,
    selectedRowIndex: -1,
  },
)

const { t } = useI18n({ useScope: 'global' })

const resolvedEmptyMessage = computed(() => props.emptyMessage || t('common.empty'))
const resolvedErrorMessage = computed(() => props.errorMessage || t('error.common.internal'))
</script>
