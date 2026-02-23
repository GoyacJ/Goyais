<template>
  <div class="pager">
    <button type="button" :disabled="!canPrev || loading" @click="$emit('prev')">上一页</button>
    <span class="pager-state">{{ stateLabel }}</span>
    <button type="button" :disabled="!canNext || loading" @click="$emit('next')">下一页</button>
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

<style scoped>
.pager {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}

.pager button {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  font-size: var(--global-font-size-11);
  padding: 2px var(--global-space-8);
}

.pager button:disabled {
  opacity: 0.5;
}

.pager-state {
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}
</style>
