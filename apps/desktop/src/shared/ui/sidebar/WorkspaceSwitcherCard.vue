<template>
  <div
    class="workspace-switcher relative grid content-start pt-[var(--global-space-4)]"
    :class="{ 'gap-[var(--global-space-8)]': !collapsed, 'gap-[var(--global-space-6)]': collapsed }"
    data-tauri-drag-region
    @mousedown="onDragMouseDown"
    @dblclick="onDragRegionDoubleClick"
  >
    <div
      v-if="supportsWindowControls"
      class="mac-row inline-flex mt-[var(--global-space-neg-2px)] gap-[var(--global-space-8)]"
      :class="{ 'justify-center': collapsed }"
      data-tauri-drag-region
    >
      <button
        class="dot danger h-[12px] w-[12px] border-0 rounded-[var(--global-radius-pill)] bg-[var(--semantic-danger)] p-0 opacity-90 transition-all duration-[120ms] ease-linear hover:scale-105 hover:opacity-100 focus-visible:outline focus-visible:outline-1 focus-visible:outline-[var(--semantic-focus-ring)] focus-visible:outline-offset-1"
        data-no-drag="true"
        type="button"
        title="Close"
        aria-label="关闭窗口"
        @click.stop="onCloseWindow"
      ></button>
      <button
        class="dot warning h-[12px] w-[12px] border-0 rounded-[var(--global-radius-pill)] bg-[var(--semantic-warning)] p-0 opacity-90 transition-all duration-[120ms] ease-linear hover:scale-105 hover:opacity-100 focus-visible:outline focus-visible:outline-1 focus-visible:outline-[var(--semantic-focus-ring)] focus-visible:outline-offset-1"
        data-no-drag="true"
        type="button"
        title="Minimize"
        aria-label="最小化窗口"
        @click.stop="onMinimizeWindow"
      ></button>
      <button
        class="dot success h-[12px] w-[12px] border-0 rounded-[var(--global-radius-pill)] bg-[var(--semantic-success)] p-0 opacity-90 transition-all duration-[120ms] ease-linear hover:scale-105 hover:opacity-100 focus-visible:outline focus-visible:outline-1 focus-visible:outline-[var(--semantic-focus-ring)] focus-visible:outline-offset-1"
        data-no-drag="true"
        type="button"
        title="Toggle Maximize"
        aria-label="切换最大化"
        @click.stop="onToggleMaximizeWindow"
      ></button>
    </div>

    <div
      class="workspace-row mt-[var(--global-space-4)] grid grid-cols-[1fr_auto] gap-[var(--global-space-8)]"
      :class="{ 'gap-[var(--global-space-4)]': collapsed }"
    >
      <button
        class="workspace-btn inline-flex min-h-[44px] items-center justify-between border-0 rounded-[var(--global-radius-8)] bg-[var(--semantic-surface-2)] px-[var(--global-space-12)] text-[var(--semantic-text)]"
        :class="{ 'justify-center px-0': collapsed }"
        type="button"
        @click="menuOpen = !menuOpen"
      >
        <span
          class="workspace-label inline-flex items-center gap-[var(--global-space-8)]"
          :class="{ 'w-full justify-center gap-0': collapsed }"
        >
          <AppIcon :name="currentWorkspace?.mode === 'local' ? 'house' : 'briefcase-business'" :size="13" />
          <template v-if="!collapsed">{{ workspaceLabel }}</template>
        </span>
        <AppIcon v-if="!collapsed" name="chevron-down" :size="14" />
      </button>

      <button
        v-if="showCollapseToggle"
        class="icon-btn inline-flex h-[44px] w-[44px] items-center justify-center border-0 rounded-[var(--global-radius-8)] bg-[var(--semantic-surface-2)] text-[var(--semantic-text)]"
        type="button"
        aria-label="切换侧栏折叠"
        @click="$emit('toggleCollapse')"
      >
        <AppIcon :name="collapsed ? 'chevron-right' : 'chevron-left'" :size="14" />
      </button>
    </div>

    <div
      v-if="menuOpen && !collapsed"
      class="workspace-menu absolute left-0 right-0 top-[calc(100%+var(--global-space-8))] z-24 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)] shadow-[var(--global-shadow-2)]"
    >
      <button
        v-for="workspace in workspaceOptions"
        :key="workspace.id"
        class="workspace-option inline-flex min-h-[44px] items-center gap-[var(--global-space-8)] border-0 rounded-[var(--global-radius-8)] bg-transparent px-[var(--global-space-8)] text-left text-[var(--semantic-text-muted)] hover:(bg-[var(--component-sidebar-item-bg-active)] text-[var(--semantic-text)])"
        :class="{ 'bg-[var(--component-sidebar-item-bg-active)] text-[var(--semantic-text)]': workspace.id === currentWorkspaceId }"
        type="button"
        @click="onSwitchWorkspace(workspace.id)"
      >
        <AppIcon :name="workspace.mode === 'local' ? 'house' : 'briefcase-business'" :size="11" />
        <span>{{ workspace.name }}</span>
      </button>
      <button
        v-if="canCreateWorkspace"
        class="workspace-option add inline-flex min-h-[44px] items-center gap-[var(--global-space-8)] border-0 rounded-[var(--global-radius-8)] bg-transparent px-[var(--global-space-8)] text-left text-[var(--semantic-text)] hover:bg-[var(--component-sidebar-item-bg-active)]"
        type="button"
        @click="onCreateWorkspace"
      >
        <AppIcon name="plus" :size="11" />
        <span>新增工作区</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";

import { isRuntimeCapabilitySupported } from "@/shared/runtime";
import {
  closeCurrentWindow,
  handleDragMouseDown,
  minimizeCurrentWindow,
  toggleMaximizeCurrentWindow
} from "@/shared/services/windowControls";
import type { Workspace } from "@/shared/types/api";
import AppIcon from "@/shared/ui/AppIcon.vue";

const props = withDefaults(
  defineProps<{
    workspaces: Workspace[];
    currentWorkspaceId: string;
    collapsed?: boolean;
    showCollapseToggle?: boolean;
    canCreateWorkspace?: boolean;
    fallbackLabel?: string;
  }>(),
  {
    collapsed: false,
    showCollapseToggle: false,
    canCreateWorkspace: false,
    fallbackLabel: "工作区"
  }
);

const emit = defineEmits<{
  (event: "switchWorkspace", workspaceId: string): void;
  (event: "toggleCollapse"): void;
  (event: "createWorkspace"): void;
}>();

const menuOpen = ref(false);
const supportsWindowControls = isRuntimeCapabilitySupported("supportsWindowControls");

const currentWorkspace = computed(() => props.workspaces.find((workspace) => workspace.id === props.currentWorkspaceId));
const workspaceLabel = computed(() => currentWorkspace.value?.name ?? props.fallbackLabel);
const workspaceOptions = computed(() => {
  const local = props.workspaces.find((workspace) => workspace.mode === "local" || workspace.is_default_local);
  const remote = props.workspaces.filter((workspace) => workspace.mode === "remote" && workspace.id !== local?.id);
  return local ? [local, ...remote] : remote;
});

function onSwitchWorkspace(workspaceId: string): void {
  emit("switchWorkspace", workspaceId);
  menuOpen.value = false;
}

function onCreateWorkspace(): void {
  emit("createWorkspace");
  menuOpen.value = false;
}

function onCloseWindow(): void {
  if (!supportsWindowControls) {
    return;
  }
  void closeCurrentWindow();
}

function onMinimizeWindow(): void {
  if (!supportsWindowControls) {
    return;
  }
  void minimizeCurrentWindow();
}

function onToggleMaximizeWindow(): void {
  if (!supportsWindowControls) {
    return;
  }
  void toggleMaximizeCurrentWindow();
}

function onDragMouseDown(event: MouseEvent): void {
  if (!supportsWindowControls) {
    return;
  }
  void handleDragMouseDown(event);
}

function onDragRegionDoubleClick(event: MouseEvent): void {
  if (!supportsWindowControls) {
    return;
  }
  if ((event.target as HTMLElement | null)?.closest("button,a,input,select,textarea,[role='button'],[data-no-drag='true']")) {
    return;
  }

  void toggleMaximizeCurrentWindow();
}
</script>
