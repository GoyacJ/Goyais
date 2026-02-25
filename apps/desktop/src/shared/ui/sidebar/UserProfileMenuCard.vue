<template>
  <div
    class="user-panel relative grid gap-[var(--global-space-8)] rounded-[var(--global-radius-8)] bg-[var(--semantic-surface-2)] p-[var(--global-space-8)]"
  >
    <button
      class="user-trigger flex min-h-[34px] w-full items-center justify-between border-0 rounded-[var(--global-radius-8)] bg-transparent px-[var(--global-space-4)] text-[var(--semantic-text)]"
      type="button"
      @click="menuOpen = !menuOpen"
    >
      <span class="user-left inline-flex items-center gap-[var(--global-space-8)]">
        <span
          class="avatar inline-flex h-[22px] w-[22px] items-center justify-center rounded-[var(--global-radius-full)] bg-[var(--semantic-primary)] text-[var(--global-font-size-11)] text-[var(--semantic-bg)] [font-weight:var(--global-font-weight-700)]"
        >
          {{ avatar }}
        </span>
        <span v-if="!collapsed" class="user-meta grid gap-[var(--global-space-2px)] text-left">
          <strong>{{ title }}</strong>
          <small v-if="subtitle" class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">{{ subtitle }}</small>
        </span>
      </span>
      <AppIcon v-if="!collapsed" name="chevron-up" :size="12" />
    </button>

    <div
      v-if="menuOpen && !collapsed"
      class="user-menu absolute bottom-[calc(100%+var(--global-space-8))] left-[var(--global-space-8)] right-[var(--global-space-8)] z-12 grid gap-[var(--global-space-4)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]"
    >
      <button
        v-for="item in items"
        :key="item.key"
        class="menu-item inline-flex min-h-[32px] items-center gap-[var(--global-space-8)] border-0 rounded-[var(--global-radius-8)] bg-transparent px-[var(--global-space-8)] text-left text-[var(--semantic-text)]"
        type="button"
        @click="onSelect(item.key)"
      >
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
