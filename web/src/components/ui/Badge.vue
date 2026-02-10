<template>
  <span :class="classes">
    <slot />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/utils/cn'

const props = withDefaults(
  defineProps<{
    tone?: 'neutral' | 'primary' | 'success' | 'warn' | 'error' | 'info'
  }>(),
  {
    tone: 'neutral',
  },
)

const toneClass: Record<NonNullable<typeof props.tone>, string> = {
  neutral: 'border-ui-borderSubtle bg-ui-surface2 text-ui-fg',
  primary: 'ui-tone-surface-primary',
  success: 'ui-tone-surface-success',
  warn: 'ui-tone-surface-warn',
  error: 'ui-tone-surface-error',
  info: 'ui-tone-surface-info',
}

const classes = computed(() =>
  cn(
    'inline-flex h-6 items-center rounded-button border px-2 text-xs font-semibold uppercase tracking-[0.05em]',
    toneClass[props.tone],
  ),
)
</script>
