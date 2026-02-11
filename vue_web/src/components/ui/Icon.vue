<template>
  <span class="inline-flex shrink-0 text-current" :class="spin ? 'animate-spin' : ''" v-html="markup" />
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
import { resolveIconSvg, type IconName } from '@/design-system/icon-registry'
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    name: IconName
    size?: number
    decorative?: boolean
    ariaLabel?: string
    spin?: boolean
  }>(),
  {
    size: 20,
    decorative: true,
    ariaLabel: '',
    spin: false,
  },
)

function escapeAttr(value: string): string {
  return value
    .split('&')
    .join('&amp;')
    .split('"')
    .join('&quot;')
    .split('<')
    .join('&lt;')
    .split('>')
    .join('&gt;')
}

const markup = computed(() => {
  const raw = resolveIconSvg(props.name)
  const ariaAttrs = props.decorative
    ? 'aria-hidden="true"'
    : `role="img" aria-label="${escapeAttr(props.ariaLabel || props.name)}"`

  return raw.replace(
    /<svg([^>]*)>/,
    `<svg$1 width="${props.size}" height="${props.size}" class="ui-icon" ${ariaAttrs}>`,
  )
})
</script>
