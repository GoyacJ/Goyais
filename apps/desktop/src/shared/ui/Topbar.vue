<template>
  <header class="topbar" data-tauri-drag-region @mousedown="onTopbarMouseDown" @dblclick="onTopbarDoubleClick">
    <div class="left">
      <slot name="left" />
    </div>
    <div class="right">
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

<style scoped>
.topbar {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border: 1px solid var(--component-topbar-border);
  background: var(--component-topbar-bg);
  border-radius: var(--global-radius-12);
  padding: 0 var(--global-space-12);
  cursor: grab;
}

.left,
.right {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}
</style>
