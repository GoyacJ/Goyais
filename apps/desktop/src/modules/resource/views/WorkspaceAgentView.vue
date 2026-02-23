<template>
  <WorkspaceSharedPage
    active-key="workspace_agent"
    title="Agent 配置（共享）"
    remote-subtitle="Workspace Config / Agent (Shared)"
    local-subtitle="Local Settings / Agent (Shared)"
    :cards="workspaceAgentCards"
  />
</template>

<script setup lang="ts">
import WorkspaceSharedPage from "@/modules/resource/components/WorkspaceSharedPage.vue";
import { workspaceAgentCards } from "@/modules/resource/schemas/pageContent";
</script>
