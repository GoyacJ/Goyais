<template>
  <button
    ref="buttonRef"
    :type="type"
    :disabled="isDisabled"
    :aria-busy="loading || undefined"
    :data-block-loading="loading && blockWhileLoading ? 'true' : 'false'"
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
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */
import { computed, ref } from 'vue'
import { cn } from '@/utils/cn'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'ghost' | 'destructive'
    loading?: boolean
    disabled?: boolean
    blockWhileLoading?: boolean
    type?: 'button' | 'submit' | 'reset'
  }>(),
  {
    variant: 'secondary',
    loading: false,
    disabled: false,
    blockWhileLoading: true,
    type: 'button',
  },
)

const buttonRef = ref<HTMLButtonElement | null>(null)

const isDisabled = computed(() => props.disabled || (props.blockWhileLoading && props.loading))

const variantClasses: Record<NonNullable<typeof props.variant>, string> = {
  primary: 'ui-btn-primary',
  secondary: 'ui-btn-secondary',
  ghost: 'ui-btn-ghost',
  destructive: 'ui-btn-destructive',
}

const classes = computed(() =>
  cn(
    'ui-control ui-focus-ring ui-pressable inline-flex items-center justify-center gap-2 font-medium',
    variantClasses[props.variant],
    isDisabled.value && 'ui-disabled',
    props.loading && 'ui-loading',
    props.loading && props.blockWhileLoading && 'ui-loading-block',
  ),
)

defineExpose({
  focus: () => {
    buttonRef.value?.focus()
  },
})
</script>
