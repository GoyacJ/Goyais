<template>
  <header
    class="flex h-[40px] cursor-grab items-center justify-between rounded-t-[var(--global-radius-12)] bg-transparent px-[var(--global-space-8)]"
    data-tauri-drag-region
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
import { handleDragMouseDown, toggleMaximizeCurrentWindow } from "@/shared/services/windowControls";

function onTopbarMouseDown(event: MouseEvent): void {
  void handleDragMouseDown(event);
}

function onTopbarDoubleClick(event: MouseEvent): void {
  if ((event.target as HTMLElement | null)?.closest("button,a,input,select,textarea,[role='button'],[data-no-drag='true']")) {
    return;
  }
  void toggleMaximizeCurrentWindow();
}
</script>
