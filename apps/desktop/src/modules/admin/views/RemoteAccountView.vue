<template>
  <RemoteConfigLayout
    active-key="remote_account"
    :menu-entries="menuEntries"
    :scope-hint="scopeHint"
    title="账号信息"
    subtitle="Remote Workspace / Account"
  >
    <ConfigSectionCard
      v-for="card in remoteAccountCards"
      :key="card.title"
      :title="card.title"
      :lines="card.lines"
      :tone="card.tone"
    />
  </RemoteConfigLayout>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";

import { refreshAdminData } from "@/modules/admin/store";
import { remoteAccountCards } from "@/modules/admin/schemas/pageContent";
import { useRemoteConfigMenu } from "@/shared/navigation/pageMenus";
import RemoteConfigLayout from "@/shared/layouts/RemoteConfigLayout.vue";
import ConfigSectionCard from "@/shared/ui/ConfigSectionCard.vue";

const menuEntries = useRemoteConfigMenu();
const scopeHint = computed(() => "Remote 视图：显示成员与角色、权限与审计，并按 RBAC+ABAC 控制");

onMounted(async () => {
  await refreshAdminData();
});
</script>
