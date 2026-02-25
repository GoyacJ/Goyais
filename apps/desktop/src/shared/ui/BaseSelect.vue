<template>
  <div>
    <select
      class="h-[var(--component-input-height)] w-full border rounded-[var(--component-input-radius)] border-[var(--component-select-trigger-border)] bg-[var(--component-select-trigger-bg)] px-[var(--global-space-12)] text-[var(--global-font-size-13)] text-[var(--component-select-trigger-fg)] [font-family:var(--global-font-family-ui)] focus-visible:outline focus-visible:outline-1 focus-visible:outline-[var(--semantic-focus-ring)]"
      :value="modelValue"
      :disabled="disabled"
      @change="emitValue"
    >
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
