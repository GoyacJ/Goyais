<template>
  <RemoteConfigLayout
    active-key="remote_permissions_audit"
    :menu-entries="menuEntries"
    :scope-hint="scopeHint"
    title="权限与审计"
    subtitle="Remote Workspace / Permissions & Audit"
  >
    <ToastAlert tone="403" message="ABAC 拒绝示例: Forbidden (403)" />
    <ConfigSectionCard
      v-for="card in remotePermissionsAuditCards"
      :key="card.title"
      :title="card.title"
      :lines="card.lines"
      :tone="card.tone"
    />
  </RemoteConfigLayout>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";

import { remotePermissionsAuditCards } from "@/modules/admin/schemas/pageContent";
import { refreshAdminData } from "@/modules/admin/store";
import RemoteConfigLayout from "@/shared/layouts/RemoteConfigLayout.vue";
import { useRemoteConfigMenu } from "@/shared/navigation/pageMenus";
import ConfigSectionCard from "@/shared/ui/ConfigSectionCard.vue";
import ToastAlert from "@/shared/ui/ToastAlert.vue";

const menuEntries = useRemoteConfigMenu();
const scopeHint = computed(() => "Remote 视图：显示成员与角色、权限与审计，并按 RBAC+ABAC 控制");

onMounted(async () => {
  await refreshAdminData();
});
</script>
