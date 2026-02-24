<template>
  <div class="ui-select-wrap">
    <select class="ui-select" :value="modelValue" :disabled="disabled" @change="emitValue">
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: string;
  disabled?: boolean;
  options: Array<{ value: string; label: string }>;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

function emitValue(event: Event): void {
  const target = event.target as HTMLSelectElement;
  emit("update:modelValue", target.value);
}
</script>

<style scoped>
.ui-select {
  height: var(--component-input-height);
  border-radius: var(--component-input-radius);
  border: 1px solid var(--component-select-trigger-border);
  background: var(--component-select-trigger-bg);
  color: var(--component-select-trigger-fg);
  padding: 0 var(--global-space-12);
  font-size: var(--global-font-size-13);
  font-family: var(--global-font-family-ui);
  width: 100%;
}

.ui-select:focus-visible {
  outline: 1px solid var(--semantic-focus-ring);
}
</style>
