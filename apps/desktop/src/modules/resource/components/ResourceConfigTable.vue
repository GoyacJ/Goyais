<template>
  <section class="card">
    <div class="card-head">
      <h3>{{ title }}</h3>
      <BaseButton
        v-if="showAdd"
        variant="secondary"
        :disabled="addDisabled"
        @click="emit('add')"
      >
        {{ addLabel }}
      </BaseButton>
    </div>

    <div class="toolbar">
      <BaseInput
        v-if="showSearch"
        :model-value="search"
        :placeholder="searchPlaceholder"
        :disabled="loading"
        @update:model-value="(value) => emit('update:search', value)"
      />
      <slot name="toolbar-right" />
    </div>

    <div class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th v-for="column in columns" :key="column.key">
              {{ column.label }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.id as string">
            <td v-for="column in columns" :key="column.key">
              <slot :name="`cell-${column.key}`" :row="row">
                {{ row[column.key] }}
              </slot>
            </td>
          </tr>
          <tr v-if="rows.length === 0">
            <td :colspan="columns.length" class="empty">{{ emptyText }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <CursorPager
      :can-prev="canPrev"
      :can-next="canNext"
      :loading="pagingLoading"
      @prev="emit('prev')"
      @next="emit('next')"
    />
  </section>
</template>

<script setup lang="ts">
import BaseInput from "@/shared/ui/BaseInput.vue";
import BaseButton from "@/shared/ui/BaseButton.vue";
import CursorPager from "@/shared/ui/CursorPager.vue";

withDefaults(
  defineProps<{
    title: string;
    columns: Array<{ key: string; label: string }>;
    rows: Array<Record<string, unknown>>;
    emptyText?: string;
    search?: string;
    searchPlaceholder?: string;
    showSearch?: boolean;
    showAdd?: boolean;
    addLabel?: string;
    addDisabled?: boolean;
    loading?: boolean;
    canPrev?: boolean;
    canNext?: boolean;
    pagingLoading?: boolean;
  }>(),
  {
    emptyText: "暂无数据",
    search: "",
    searchPlaceholder: "搜索...",
    showSearch: true,
    showAdd: true,
    addLabel: "新增",
    addDisabled: false,
    loading: false,
    canPrev: false,
    canNext: false,
    pagingLoading: false
  }
);

const emit = defineEmits<{
  (event: "update:search", value: string): void;
  (event: "add"): void;
  (event: "prev"): void;
  (event: "next"): void;
}>();
</script>

<style scoped>
.card {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--component-card-padding);
  display: grid;
  gap: var(--global-space-12);
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--global-space-8);
}

.card-head h3 {
  margin: 0;
}

.toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--global-space-8);
}

.table-wrap {
  overflow-x: auto;
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-2);
}

.table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  min-width: 760px;
}

.table th,
.table td {
  text-align: left;
  padding: var(--global-space-10) var(--global-space-12);
  border-bottom: 1px solid var(--semantic-border);
  vertical-align: middle;
}

.table th {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
  font-weight: 600;
  white-space: nowrap;
}

.table td {
  color: var(--semantic-text);
  font-size: var(--global-font-size-13);
  line-height: 1.45;
}

.empty {
  text-align: center;
  color: var(--semantic-text-subtle);
}

</style>
