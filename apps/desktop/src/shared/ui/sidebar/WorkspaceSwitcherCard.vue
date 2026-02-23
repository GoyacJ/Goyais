<template>
  <div class="workspace-switcher" :class="{ collapsed }">
    <div class="mac-row">
      <span class="dot danger"></span>
      <span class="dot warning"></span>
      <span class="dot success"></span>
    </div>

    <div class="workspace-row">
      <button class="workspace-btn" type="button" @click="menuOpen = !menuOpen">
        <span class="workspace-label">
          <AppIcon :name="currentWorkspace?.mode === 'local' ? 'house' : 'briefcase-business'" :size="13" />
          <template v-if="!collapsed">{{ workspaceLabel }}</template>
        </span>
        <AppIcon v-if="!collapsed" name="chevron-down" :size="14" />
      </button>

      <button v-if="showCollapseToggle" class="icon-btn" type="button" @click="$emit('toggleCollapse')">
        <AppIcon :name="collapsed ? 'chevron-right' : 'chevron-left'" :size="14" />
      </button>
    </div>

    <div v-if="menuOpen && !collapsed" class="workspace-menu">
      <button
        v-for="workspace in workspaces"
        :key="workspace.id"
        class="workspace-option"
        :class="{ active: workspace.id === currentWorkspaceId }"
        type="button"
        @click="onSwitchWorkspace(workspace.id)"
      >
        <AppIcon :name="workspace.mode === 'local' ? 'house' : 'briefcase-business'" :size="11" />
        <span>{{ workspace.name }}</span>
      </button>
      <button v-if="canCreateWorkspace" class="workspace-option add" type="button" @click="onCreateWorkspace">
        <AppIcon name="plus" :size="11" />
        <span>新增工作区</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";

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

const currentWorkspace = computed(() => props.workspaces.find((workspace) => workspace.id === props.currentWorkspaceId));
const workspaceLabel = computed(() => currentWorkspace.value?.name ?? props.fallbackLabel);

function onSwitchWorkspace(workspaceId: string): void {
  emit("switchWorkspace", workspaceId);
  menuOpen.value = false;
}

function onCreateWorkspace(): void {
  emit("createWorkspace");
  menuOpen.value = false;
}
</script>

<style scoped>
.workspace-switcher {
  display: grid;
  gap: var(--global-space-8);
  align-content: start;
}

.workspace-switcher.collapsed {
  gap: var(--global-space-6);
}

.mac-row {
  display: inline-flex;
  gap: var(--global-space-8);
}

.dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
}

.danger {
  background: var(--semantic-danger);
}

.warning {
  background: var(--semantic-warning);
}

.success {
  background: var(--semantic-success);
}

.workspace-row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: var(--global-space-8);
}

.workspace-btn,
.icon-btn,
.workspace-option {
  border: 0;
  border-radius: var(--global-radius-8);
  color: var(--semantic-text);
}

.workspace-btn {
  background: var(--semantic-surface-2);
  min-height: 34px;
  padding: 0 var(--global-space-12);
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
}

.workspace-label {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}

.icon-btn {
  width: 34px;
  height: 34px;
  background: var(--semantic-surface-2);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.workspace-menu {
  background: var(--semantic-bg);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-4);
}

.workspace-option {
  background: transparent;
  color: var(--semantic-text-muted);
  min-height: 32px;
  padding: 0 var(--global-space-8);
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
  text-align: left;
}

.workspace-option.active,
.workspace-option:hover {
  background: var(--component-sidebar-item-bg-active);
  color: var(--semantic-text);
}

.workspace-option.add {
  color: var(--semantic-text);
}
</style>
