<template>
  <textarea
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :rows="rows"
    :class="classes"
    @input="onInput"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/utils/cn'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    rows?: number
    disabled?: boolean
  }>(),
  {
    modelValue: '',
    placeholder: '',
    rows: 4,
    disabled: false,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

function onInput(event: Event): void {
  const target = event.target as HTMLTextAreaElement
  emit('update:modelValue', target.value)
}

const classes = computed(() =>
  cn(
    'ui-focus-ring ui-pressable w-full rounded-button border border-ui-border bg-ui-panel px-[var(--ui-control-px)] py-[var(--ui-control-py)] text-sm',
    'min-h-[calc(var(--ui-control-h)*2.2)]',
    props.disabled && 'ui-disabled',
  ),
)
</script>
