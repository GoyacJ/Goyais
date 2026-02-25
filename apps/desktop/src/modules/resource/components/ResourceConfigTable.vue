<template>
  <section class="card grid gap-[var(--global-space-12)] border border-[var(--semantic-border)] rounded-[var(--global-radius-12)] bg-[var(--semantic-surface)] p-[var(--component-card-padding)]">
    <div class="card-head flex items-center justify-between gap-[var(--global-space-8)]">
      <h3 class="m-0">{{ title }}</h3>
      <BaseButton
        v-if="showAdd"
        variant="secondary"
        :disabled="addDisabled"
        @click="emit('add')"
      >
        {{ addLabel }}
      </BaseButton>
    </div>

    <div class="toolbar grid grid-cols-[minmax(0,1fr)_auto] gap-[var(--global-space-8)]">
      <BaseInput
        v-if="showSearch"
        :model-value="search"
        :placeholder="searchPlaceholder"
        :disabled="loading"
        @update:model-value="(value) => emit('update:search', value)"
      />
      <slot name="toolbar-right" />
    </div>

    <div class="table-wrap overflow-x-auto border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] p-[var(--global-space-2)]">
      <table class="table w-full min-w-[760px] border-separate border-spacing-0">
        <thead>
          <tr>
            <th
              v-for="column in columns"
              :key="column.key"
              class="border-b border-[var(--semantic-border)] px-[var(--global-space-12)] py-[var(--global-space-10)] text-left align-middle whitespace-nowrap text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)] [font-weight:600]"
            >
              {{ column.label }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.id as string">
            <td
              v-for="column in columns"
              :key="column.key"
              class="border-b border-[var(--semantic-border)] px-[var(--global-space-12)] py-[var(--global-space-10)] text-left align-middle text-[var(--global-font-size-13)] leading-[1.45] text-[var(--semantic-text)]"
            >
              <slot :name="`cell-${column.key}`" :row="row">
                {{ row[column.key] }}
              </slot>
            </td>
          </tr>
          <tr v-if="rows.length === 0">
            <td
              :colspan="columns.length"
              class="empty border-b border-[var(--semantic-border)] px-[var(--global-space-12)] py-[var(--global-space-10)] text-center text-[var(--semantic-text-subtle)]"
            >
              {{ emptyText }}
            </td>
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
