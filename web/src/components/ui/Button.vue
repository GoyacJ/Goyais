<template>
  <button
    ref="buttonRef"
    :type="type"
    :disabled="isDisabled"
    :class="classes"
  >
    <span
      v-if="loading"
      class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent"
      aria-hidden="true"
    />
    <span><slot /></span>
  </button>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { cn } from '@/utils/cn'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'ghost' | 'destructive'
    loading?: boolean
    disabled?: boolean
    type?: 'button' | 'submit' | 'reset'
  }>(),
  {
    variant: 'secondary',
    loading: false,
    disabled: false,
    type: 'button',
  },
)

const isDisabled = computed(() => props.disabled || props.loading)
const buttonRef = ref<HTMLButtonElement | null>(null)

const variantClasses: Record<NonNullable<typeof props.variant>, string> = {
  primary: 'border-primary-600 bg-primary-600 text-white hover:border-primary-700 hover:bg-primary-700',
  secondary: 'border-ui-border bg-ui-panel text-ui-fg',
  ghost: 'border-transparent bg-transparent text-ui-fg',
  destructive: 'border-error bg-error text-white hover:bg-error/90',
}

const classes = computed(() =>
  cn(
    'ui-control ui-focus-ring ui-pressable inline-flex items-center justify-center gap-2 font-medium',
    variantClasses[props.variant],
    isDisabled.value && 'ui-disabled',
    props.loading && 'ui-loading',
  ),
)

defineExpose({
  focus: () => {
    buttonRef.value?.focus()
  },
})
</script>
