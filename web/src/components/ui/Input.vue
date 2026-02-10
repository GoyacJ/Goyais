<template>
  <input
    :value="modelValue"
    :type="type"
    :placeholder="placeholder"
    :disabled="disabled"
    :aria-busy="loading || undefined"
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
    type?: string
    disabled?: boolean
    loading?: boolean
  }>(),
  {
    modelValue: '',
    placeholder: '',
    type: 'text',
    disabled: false,
    loading: false,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

function onInput(event: Event): void {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
}

const classes = computed(() =>
  cn(
    'ui-control ui-focus-ring ui-pressable w-full text-sm',
    props.disabled && 'ui-disabled',
    props.loading && 'ui-loading',
  ),
)
</script>
