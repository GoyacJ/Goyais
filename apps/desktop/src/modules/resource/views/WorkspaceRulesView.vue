<template>
  <WorkspaceSharedPage
    active-key="workspace_rules"
    title="Rules 配置（共享）"
    account-subtitle="Workspace Config / Rules (Shared)"
    settings-subtitle="Local Settings / Rules (Shared)"
    :cards="workspaceRulesCards"
  />
</template>

<script setup lang="ts">
import WorkspaceSharedPage from "@/modules/resource/components/WorkspaceSharedPage.vue";
import { workspaceRulesCards } from "@/modules/resource/schemas/pageContent";
</script>
