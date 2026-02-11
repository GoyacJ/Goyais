<template>
  <section class="ui-card border-dashed text-center">
    <div class="ui-empty-tone mx-auto mb-4 max-w-[24rem] [&_svg]:h-auto [&_svg]:w-full" v-html="illustration" />
    <h3 v-if="title" class="text-sm font-semibold text-ui-fg">
      <span class="ui-empty-tone-dot" :class="illustrationToneDot" />
      {{ title }}
    </h3>
    <p v-if="description" class="mt-1 text-sm text-ui-muted">{{ description }}</p>
    <div v-if="$slots.actions" class="mt-3 flex justify-center">
      <slot name="actions" />
    </div>
  </section>
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
import assetsEmptySvg from '@/assets/illustrations/states/empty-assets.svg?raw'
import commandsEmptySvg from '@/assets/illustrations/states/empty-commands.svg?raw'
import errorSvg from '@/assets/illustrations/states/error.svg?raw'
import forbiddenSvg from '@/assets/illustrations/states/forbidden-403.svg?raw'
import loadingSvg from '@/assets/illustrations/states/loading.svg?raw'
import notFoundSvg from '@/assets/illustrations/states/not-found-404.svg?raw'
import { computed } from 'vue'

export type EmptyStateVariant =
  | 'generic'
  | 'commands-empty'
  | 'assets-empty'
  | 'forbidden-403'
  | 'not-found-404'
  | 'loading'
  | 'error'

const props = withDefaults(
  defineProps<{
    variant?: EmptyStateVariant
    title?: string
    description?: string
  }>(),
  {
    variant: 'generic',
    title: '',
    description: '',
  },
)

const illustrations: Record<EmptyStateVariant, string> = {
  generic: commandsEmptySvg,
  'commands-empty': commandsEmptySvg,
  'assets-empty': assetsEmptySvg,
  'forbidden-403': forbiddenSvg,
  'not-found-404': notFoundSvg,
  loading: loadingSvg,
  error: errorSvg,
}

const toneClasses: Record<EmptyStateVariant, string> = {
  generic: '',
  'commands-empty': 'ui-empty-tone-dot--primary',
  'assets-empty': 'ui-empty-tone-dot--info',
  'forbidden-403': 'ui-empty-tone-dot--error',
  'not-found-404': 'ui-empty-tone-dot--warn',
  loading: 'ui-empty-tone-dot--primary',
  error: 'ui-empty-tone-dot--error',
}

const illustration = computed(() => illustrations[props.variant])
const illustrationToneDot = computed(() => toneClasses[props.variant])
</script>
