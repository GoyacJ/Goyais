<template>
  <aside class="sidebar">
    <div class="sidebar-top">
      <div class="local-trigger">
        <IconSymbol name="settings" :size="14" />
        <span>Local Settings (shared modules)</span>
      </div>

      <p class="group-title">工作区配置（共享页面）</p>
      <nav class="menu-list">
        <RouterLink
          v-for="item in sharedEntries"
          :key="item.key"
          :to="item.path"
          class="menu-item"
          :class="{ active: item.key === activeKey, muted: item.visibility !== 'enabled' }"
          @click.prevent="onMenuClick(item)"
        >
          {{ item.label }}
        </RouterLink>
      </nav>

      <p class="group-title">软件通用设置</p>
      <nav class="menu-list">
        <RouterLink
          v-for="item in generalEntries"
          :key="item.key"
          :to="item.path"
          class="menu-item"
          :class="{ active: item.key === activeKey, muted: item.visibility !== 'enabled' }"
          @click.prevent="onMenuClick(item)"
        >
          {{ item.label }}
        </RouterLink>
      </nav>
    </div>

    <div class="local-panel">
      <p class="scope-title">本地工作区 Local</p>
      <p class="scope-desc">本地模式不显示账号信息。</p>
      <div class="panel-actions">
        <button class="link-btn" type="button" @click="openRemoteWorkspace">打开远程工作区</button>
        <button class="link-btn secondary" type="button">查看连接状态</button>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";

import type { MenuEntry } from "@/shared/navigation/pageMenus";
import IconSymbol from "@/shared/ui/IconSymbol.vue";

const props = defineProps<{
  activeKey: string;
  menuEntries: MenuEntry[];
}>();

const router = useRouter();

const sharedKeys = ["workspace_agent", "workspace_model", "workspace_rules", "workspace_skills", "workspace_mcp"];
const generalKeys = ["settings_theme", "settings_i18n", "settings_updates_diagnostics", "settings_general"];

const sharedEntries = computed(() =>
  props.menuEntries.filter((item) => sharedKeys.includes(item.key))
);

const generalEntries = computed(() =>
  props.menuEntries.filter((item) => generalKeys.includes(item.key))
);

function onMenuClick(item: MenuEntry): void {
  if (item.visibility === "enabled" || item.visibility === "readonly") {
    void router.push(item.path);
  }
}

function openRemoteWorkspace(): void {
  void router.push("/main");
}
</script>

<style scoped>
.sidebar {
  background: var(--semantic-surface);
  border-radius: var(--global-radius-12);
  padding: var(--global-space-12);
  display: grid;
  grid-template-rows: 1fr auto;
  gap: var(--global-space-12);
}
.sidebar-top {
  display: grid;
  gap: var(--global-space-8);
  align-content: start;
}
.local-trigger {
  height: 34px;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
  padding: 0 var(--global-space-12);
}
.group-title {
  margin: 0;
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
  font-weight: var(--global-font-weight-600);
}
.menu-list {
  display: grid;
  gap: var(--global-space-4);
}
.menu-item {
  color: var(--semantic-text-muted);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8) var(--global-space-12);
  font-size: var(--global-font-size-12);
}
.menu-item.active {
  color: var(--semantic-text);
  background: var(--component-sidebar-item-bg-active);
}
.menu-item.muted {
  opacity: var(--component-tree-item-disabled-opacity);
}
.local-panel {
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-8);
}
.scope-title,
.scope-desc {
  margin: 0;
}
.scope-title {
  color: var(--semantic-text);
  font-size: var(--global-font-size-12);
  font-weight: var(--global-font-weight-600);
}
.scope-desc {
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}
.panel-actions {
  display: inline-flex;
  gap: var(--global-space-8);
}
.link-btn {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface);
  color: var(--semantic-text);
  padding: var(--global-space-4) var(--global-space-8);
  font-size: var(--global-font-size-11);
}
.secondary {
  color: var(--semantic-text-muted);
}
</style>
