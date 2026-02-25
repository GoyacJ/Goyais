<template>
  <div class="screen" :class="{ 'sidebar-open': sidebarOpen }">
    <button
      class="mobile-backdrop"
      type="button"
      aria-label="关闭导航菜单"
      @click="sidebarOpen = false"
    ></button>

    <aside class="sidebar-slot" @click="onSidebarClick">
      <div class="sidebar-slot-fill">
        <slot name="sidebar" />
      </div>
    </aside>

    <section class="content">
      <button
        class="mobile-menu-button"
        type="button"
        aria-label="打开导航菜单"
        @click="sidebarOpen = true"
      >
        ≡
      </button>
      <slot name="header" />
      <slot name="main" />
      <slot name="footer" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

const sidebarOpen = ref(false);

function onSidebarClick(event: MouseEvent): void {
  const target = event.target as HTMLElement | null;
  if (!target) {
    return;
  }
  if (target.closest("a,[role='link']")) {
    sidebarOpen.value = false;
  }
}
</script>

<style scoped>
.screen {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--global-space-8);
  padding: 0;
  background: var(--component-shell-bg);
  overflow: hidden;
}

.sidebar-slot {
  min-height: 0;
  height: 100%;
}

.sidebar-slot-fill {
  height: 100%;
  display: grid;
}

.content {
  padding: 0 var(--global-space-8) 0 0;
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: var(--global-space-8);
  border-radius: var(--global-radius-12);
  min-height: 0;
  overflow: hidden;
  position: relative;
}

.mobile-backdrop,
.mobile-menu-button {
  display: none;
}

@media (max-width: 768px) {
  .screen {
    grid-template-columns: minmax(0, 1fr);
    gap: 0;
    overflow: visible;
    padding-top: var(--safe-area-top);
    padding-right: var(--safe-area-right);
    padding-bottom: var(--safe-area-bottom);
    padding-left: var(--safe-area-left);
  }

  .sidebar-slot {
    position: fixed;
    top: var(--safe-area-top);
    left: var(--safe-area-left);
    bottom: var(--safe-area-bottom);
    width: min(86vw, 340px);
    z-index: 40;
    transform: translateX(calc(-100% - var(--global-space-12)));
    transition: transform 0.2s ease;
  }

  .screen.sidebar-open .sidebar-slot {
    transform: translateX(0);
  }

  .content {
    padding: 0 var(--global-space-8);
    border-radius: 0;
    grid-template-rows: auto 1fr auto;
  }

  .mobile-menu-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    position: absolute;
    top: var(--global-space-8);
    left: var(--global-space-8);
    width: 44px;
    height: 44px;
    border: 1px solid var(--semantic-border);
    border-radius: var(--global-radius-12);
    background: var(--semantic-surface);
    color: var(--semantic-text);
    z-index: 12;
  }

  .mobile-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    border: 0;
    background: transparent;
    pointer-events: none;
    z-index: 32;
  }

  .screen.sidebar-open .mobile-backdrop {
    background: #00000066;
    pointer-events: auto;
  }
}
</style>
