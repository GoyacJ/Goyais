<template>
  <div class="pointer-events-none fixed right-4 top-4 z-[60] flex w-80 flex-col gap-2">
    <transition-group name="toast" tag="div">
      <article
        v-for="item in items"
        :key="item.id"
        class="ui-overlay-panel pointer-events-auto p-3"
      >
        <header class="flex items-center justify-between gap-2">
          <p class="text-sm font-semibold" :class="toneClass[item.tone]">{{ item.title }}</p>
          <button
            type="button"
            class="ui-focus-ring ui-pressable rounded-button px-1 text-xs text-ui-muted"
            @click="removeToast(item.id)"
          >
            x
          </button>
        </header>
        <p class="mt-1 text-sm text-ui-muted">{{ item.message }}</p>
      </article>
    </transition-group>
  </div>
</template>

<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const { items, removeToast } = useToast()

const toneClass = {
  info: 'text-info',
  success: 'text-success',
  warn: 'text-warn',
  error: 'text-error',
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
