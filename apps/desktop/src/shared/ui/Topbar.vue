<template>
  <header
    class="topbar flex min-h-[44px] items-center justify-between rounded-t-[var(--global-radius-12)] bg-transparent px-[var(--global-space-8)]"
    :class="{ 'cursor-grab': supportsWindowControls, 'cursor-default': !supportsWindowControls }"
    :data-tauri-drag-region="supportsWindowControls ? '' : undefined"
    @mousedown="onTopbarMouseDown"
    @dblclick="onTopbarDoubleClick"
  >
    <div class="inline-flex items-center gap-[var(--global-space-8)]">
      <slot name="left" />
    </div>
    <div class="inline-flex items-center gap-[var(--global-space-8)]">
      <slot name="right" />
    </div>
  </header>
</template>

<script setup lang="ts">
import { isRuntimeCapabilitySupported } from "@/shared/runtime";
import { handleDragMouseDown, toggleMaximizeCurrentWindow } from "@/shared/services/windowControls";

const supportsWindowControls = isRuntimeCapabilitySupported("supportsWindowControls");

function onTopbarMouseDown(event: MouseEvent): void {
  if (!supportsWindowControls) {
    return;
  }
  void handleDragMouseDown(event);
}

function onTopbarDoubleClick(event: MouseEvent): void {
  if (!supportsWindowControls) {
    return;
  }
  if ((event.target as HTMLElement | null)?.closest("button,a,input,select,textarea,[role='button'],[data-no-drag='true']")) {
    return;
  }
  void toggleMaximizeCurrentWindow();
}
</script>

<style scoped>
@media (max-width: 768px) {
  .topbar {
    min-height: 52px;
    padding-left: calc(var(--global-space-8) + var(--safe-area-left));
    padding-right: calc(var(--global-space-8) + var(--safe-area-right));
  }
}
</style>
