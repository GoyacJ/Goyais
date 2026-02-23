<template>
  <RemoteConfigLayout
    active-key="remote_members_roles"
    :menu-entries="menuEntries"
    :scope-hint="scopeHint"
    title="成员与角色"
    subtitle="Remote Workspace / Members & Roles"
  >
    <ConfigSectionCard
      v-for="card in remoteMembersRolesCards"
      :key="card.title"
      :title="card.title"
      :lines="card.lines"
      :tone="card.tone"
    />
  </RemoteConfigLayout>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";

import { remoteMembersRolesCards } from "@/modules/admin/schemas/pageContent";
import { refreshAdminData } from "@/modules/admin/store";
import RemoteConfigLayout from "@/shared/layouts/RemoteConfigLayout.vue";
import { useRemoteConfigMenu } from "@/shared/navigation/pageMenus";
import ConfigSectionCard from "@/shared/ui/ConfigSectionCard.vue";

const menuEntries = useRemoteConfigMenu();
const scopeHint = computed(() => "Remote 视图：显示成员与角色、权限与审计，并按 RBAC+ABAC 控制");

onMounted(async () => {
  await refreshAdminData();
});
</script>
