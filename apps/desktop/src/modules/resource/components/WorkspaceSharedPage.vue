<template>
  <component
    :is="layoutComponent"
    :active-key="activeKey"
    :menu-entries="menuEntries"
    :scope-hint="scopeHint"
    :title="title"
    :subtitle="subtitle"
  >
    <ConfigSectionCard
      v-for="card in cards"
      :key="card.title"
      :title="card.title"
      :lines="card.lines"
      :tone="card.tone"
      :mono="card.mono"
    />
  </component>
</template>

<script setup lang="ts">
import { computed } from "vue";

import LocalSettingsLayout from "@/shared/layouts/LocalSettingsLayout.vue";
import RemoteConfigLayout from "@/shared/layouts/RemoteConfigLayout.vue";
import {
  useLocalSettingsMenu,
  useRemoteConfigMenu
} from "@/shared/navigation/pageMenus";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import ConfigSectionCard from "@/shared/ui/ConfigSectionCard.vue";

type CardSpec = {
  title: string;
  lines: string[];
  tone: "default" | "info" | "warning" | "danger" | "success";
  mono?: boolean;
};

const props = defineProps<{
  activeKey: string;
  title: string;
  remoteSubtitle: string;
  localSubtitle: string;
  cards: CardSpec[];
}>();

const remoteMenuEntries = useRemoteConfigMenu();
const localMenuEntries = useLocalSettingsMenu();

const layoutComponent = computed(() =>
  workspaceStore.mode === "remote" ? RemoteConfigLayout : LocalSettingsLayout
);

const menuEntries = computed(() =>
  workspaceStore.mode === "remote" ? remoteMenuEntries.value : localMenuEntries.value
);

const subtitle = computed(() =>
  workspaceStore.mode === "remote" ? props.remoteSubtitle : props.localSubtitle
);

const scopeHint = computed(() => "共享模块：可从账号信息或设置进入；根据当前工作区与权限显示不同能力。");
</script>
