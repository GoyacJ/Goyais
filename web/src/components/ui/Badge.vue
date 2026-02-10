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
  neutral: 'border-ui-border bg-ui-hover text-ui-fg',
  primary: 'border-primary-600/50 bg-primary-500/15 text-primary-700 dark:text-primary-500',
  success: 'border-success/50 bg-success/15 text-success',
  warn: 'border-warn/50 bg-warn/15 text-warn',
  error: 'border-error/50 bg-error/15 text-error',
  info: 'border-info/50 bg-info/15 text-info',
}

const classes = computed(() =>
  cn(
    'inline-flex h-6 items-center rounded-button border px-2 text-xs font-semibold uppercase tracking-[0.05em]',
    toneClass[props.tone],
  ),
)
</script>
