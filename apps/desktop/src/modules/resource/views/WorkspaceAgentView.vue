<template>
  <WorkspaceSharedPage
    active-key="workspace_agent"
    title="Agent 配置（共享）"
    account-subtitle="Workspace Config / Agent (Shared)"
    settings-subtitle="Local Settings / Agent (Shared)"
    :cards="workspaceAgentCards"
  />
</template>

<script setup lang="ts">
import WorkspaceSharedPage from "@/modules/resource/components/WorkspaceSharedPage.vue";
import { workspaceAgentCards } from "@/modules/resource/schemas/pageContent";
</script>
