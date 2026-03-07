<template>
  <Teleport to="body">
    <TransitionGroup
      name="global-toast"
      tag="div"
      class="global-toast-host"
      aria-live="polite"
      aria-atomic="true"
    >
      <div v-for="item in toastItems" :key="item.id" class="global-toast-item">
        <ToastAlert :tone="item.tone" :message="item.message" />
      </div>
    </TransitionGroup>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { toastStore } from "@/shared/stores/toastStore";
import ToastAlert from "@/shared/ui/ToastAlert.vue";

const toastItems = computed(() => toastStore.items);
</script>

<style scoped>
.global-toast-host {
  position: fixed;
  top: calc(var(--safe-area-top) + var(--global-space-24));
  left: 50%;
  transform: translateX(-50%);
  max-width: calc(100vw - var(--global-space-24));
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--global-space-8);
  pointer-events: none;
  z-index: 1100;
}

.global-toast-item {
  inline-size: fit-content;
  max-inline-size: min(72ch, calc(100vw - var(--global-space-24)));
  pointer-events: auto;
}

.global-toast-enter-active,
.global-toast-leave-active {
  transition: transform 0.18s ease, opacity 0.18s ease;
}

.global-toast-enter-from,
.global-toast-leave-to {
  transform: translateY(-6px);
  opacity: 0;
}
</style>
