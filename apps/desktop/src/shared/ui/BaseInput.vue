<template>
  <div class="ui-input-wrap">
    <input
      class="ui-input"
      :class="{ error, disabled }"
      :disabled="disabled"
      :placeholder="placeholder"
      :value="modelValue"
      @input="emitValue"
    />
    <p v-if="hint !== ''" class="hint" :class="{ error }">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: string;
    placeholder?: string;
    hint?: string;
    error?: boolean;
    disabled?: boolean;
  }>(),
  {
    placeholder: "",
    hint: "",
    error: false,
    disabled: false
  }
);

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

function emitValue(event: Event): void {
  const target = event.target as HTMLInputElement;
  emit("update:modelValue", target.value);
}
</script>

<style scoped>
.ui-input-wrap {
  display: grid;
  gap: var(--global-space-4);
}

.ui-input {
  height: var(--component-input-height);
  border-radius: var(--component-input-radius);
  background: var(--component-input-bg);
  color: var(--component-input-fg);
  border: 1px solid var(--component-input-border);
  padding: 0 var(--global-space-12);
  font-family: var(--global-font-family-ui);
  font-size: var(--global-font-size-13);
}

.ui-input::placeholder {
  color: var(--component-input-placeholder);
}

.ui-input:focus-visible {
  outline: 1px solid var(--component-input-border-focus);
}

.ui-input.error {
  border-color: var(--component-input-border-error);
}

.ui-input.disabled {
  background: var(--component-input-disabled-bg);
  color: var(--component-input-disabled-fg);
}

.hint {
  margin: 0;
  color: var(--component-input-hint);
  font-size: var(--global-font-size-12);
}

.hint.error {
  color: var(--semantic-danger);
}
</style>
