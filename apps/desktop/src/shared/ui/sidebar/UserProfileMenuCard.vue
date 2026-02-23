<template>
  <div class="user-panel">
    <button class="user-trigger" type="button" @click="menuOpen = !menuOpen">
      <span class="user-left">
        <span class="avatar">{{ avatar }}</span>
        <span v-if="!collapsed" class="user-meta">
          <strong>{{ title }}</strong>
          <small v-if="subtitle">{{ subtitle }}</small>
        </span>
      </span>
      <AppIcon v-if="!collapsed" name="chevron-up" :size="12" />
    </button>

    <div v-if="menuOpen && !collapsed" class="user-menu">
      <button v-for="item in items" :key="item.key" class="menu-item" type="button" @click="onSelect(item.key)">
        <AppIcon :name="item.icon" :size="12" />
        <span>{{ item.label }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

import AppIcon from "@/shared/ui/AppIcon.vue";

type MenuItem = {
  key: string;
  label: string;
  icon: string;
};

withDefaults(
  defineProps<{
    collapsed?: boolean;
    avatar: string;
    title: string;
    subtitle?: string;
    items: MenuItem[];
  }>(),
  {
    collapsed: false,
    subtitle: ""
  }
);

const emit = defineEmits<{
  (event: "select", key: string): void;
}>();

const menuOpen = ref(false);

function onSelect(key: string): void {
  emit("select", key);
  menuOpen.value = false;
}
</script>

<style scoped>
.user-panel {
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-8);
}

.user-trigger {
  border: 0;
  border-radius: var(--global-radius-8);
  background: transparent;
  color: var(--semantic-text);
  min-height: 34px;
  padding: 0 var(--global-space-4);
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
}

.user-left {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}

.avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--semantic-primary);
  color: var(--semantic-bg);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: var(--global-font-size-11);
  font-weight: var(--global-font-weight-700);
}

.user-meta {
  display: grid;
  gap: 2px;
  text-align: left;
}

.user-meta small {
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}

.user-menu {
  background: var(--semantic-bg);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-4);
}

.menu-item {
  border: 0;
  border-radius: var(--global-radius-8);
  background: transparent;
  color: var(--semantic-text);
  min-height: 32px;
  padding: 0 var(--global-space-8);
  display: inline-flex;
  gap: var(--global-space-8);
  align-items: center;
  text-align: left;
}
</style>
