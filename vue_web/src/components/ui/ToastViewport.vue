<template>
  <div
    class="pointer-events-none fixed right-4 top-4 z-[60] flex w-80 flex-col gap-2"
    aria-live="polite"
    aria-atomic="true"
  >
    <transition-group name="toast" tag="div">
      <article
        v-for="item in items"
        :key="item.id"
        class="ui-overlay-panel pointer-events-auto p-3"
      >
        <header class="flex items-center justify-between gap-2">
          <p class="flex items-center gap-2 text-sm font-semibold text-ui-fg">
            <span class="ui-empty-tone-dot" :class="toneDotClass[item.tone]" />
            <span>{{ item.title }}</span>
          </p>
          <button
            type="button"
            class="ui-focus-ring ui-pressable rounded-button px-1 text-xs text-ui-muted"
            :aria-label="t('common.dismissNotification')"
            @click="removeToast(item.id)"
          >
            ×
          </button>
        </header>
        <p class="mt-1 text-sm text-ui-muted">{{ item.message }}</p>
      </article>
    </transition-group>
  </div>
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
import { useToast } from '@/composables/useToast'
import { useI18n } from 'vue-i18n'

const { items, removeToast } = useToast()
const { t } = useI18n({ useScope: 'global' })

const toneDotClass = {
  info: 'ui-empty-tone-dot--info',
  success: 'ui-empty-tone-dot--success',
  warn: 'ui-empty-tone-dot--warn',
  error: 'ui-empty-tone-dot--error',
}
</script>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 140ms ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
