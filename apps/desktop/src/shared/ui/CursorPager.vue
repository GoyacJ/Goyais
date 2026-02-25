<template>
  <div class="inline-flex items-center gap-[var(--global-space-8)]">
    <button
      type="button"
      :disabled="!canPrev || loading"
      class="border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-surface-2)] px-[var(--global-space-8)] py-[var(--global-space-2px)] text-[var(--global-font-size-11)] text-[var(--semantic-text)] disabled:opacity-50"
      @click="$emit('prev')"
    >
      上一页
    </button>
    <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">{{ stateLabel }}</span>
    <button
      type="button"
      :disabled="!canNext || loading"
      class="border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-surface-2)] px-[var(--global-space-8)] py-[var(--global-space-2px)] text-[var(--global-font-size-11)] text-[var(--semantic-text)] disabled:opacity-50"
      @click="$emit('next')"
    >
      下一页
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  canPrev: boolean;
  canNext: boolean;
  loading?: boolean;
}>();

defineEmits<{
  (event: "prev"): void;
  (event: "next"): void;
}>();

const stateLabel = computed(() => {
  if (props.loading) {
    return "加载中";
  }
  if (!props.canPrev && !props.canNext) {
    return "单页";
  }
  return "分页";
});
</script>
