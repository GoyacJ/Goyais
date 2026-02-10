<template>
  <div class="ui-surface overflow-hidden">
    <table class="w-full border-collapse text-left text-sm">
      <thead class="bg-ui-hover text-xs uppercase tracking-[0.05em] text-ui-muted">
        <tr>
          <th
            v-for="column in columns"
            :key="column.key"
            class="border-b border-ui-border px-3 py-2 font-semibold"
          >
            {{ column.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, index) in rows" :key="index" class="ui-table-row border-b border-ui-border/70 last:border-b-0">
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
    </table>
  </div>
</template>

<script setup lang="ts">
export interface TableColumn {
  key: string
  label: string
  mono?: boolean
}

defineProps<{
  columns: TableColumn[]
  rows: Array<Record<string, string | number>>
}>()
</script>
