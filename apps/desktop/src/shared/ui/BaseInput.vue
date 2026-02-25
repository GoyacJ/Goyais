<template>
  <div class="grid gap-[var(--global-space-4)]">
    <input
      class="h-[var(--component-input-height)] border rounded-[var(--component-input-radius)] border-[var(--component-input-border)] bg-[var(--component-input-bg)] px-[var(--global-space-12)] text-[var(--global-font-size-13)] text-[var(--component-input-fg)] [font-family:var(--global-font-family-ui)] [&::placeholder]:text-[var(--component-input-placeholder)] focus-visible:outline focus-visible:outline-1 focus-visible:outline-[var(--component-input-border-focus)]"
      :class="{
        'border-[var(--component-input-border-error)]': error,
        'bg-[var(--component-input-disabled-bg)] text-[var(--component-input-disabled-fg)]': disabled
      }"
      :disabled="disabled"
      :placeholder="placeholder"
      :value="modelValue"
      @input="emitValue"
    />
    <p v-if="hint !== ''" class="m-0 text-[var(--global-font-size-12)] text-[var(--component-input-hint)]" :class="{ 'text-[var(--semantic-danger)]': error }">
      {{ hint }}
    </p>
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
