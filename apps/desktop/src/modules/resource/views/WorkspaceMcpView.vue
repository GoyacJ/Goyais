<template>
  <WorkspaceSharedPage
    active-key="workspace_mcp"
    title="MCP 配置（共享）"
    account-subtitle="Workspace Config / MCP (Shared)"
    settings-subtitle="Local Settings / MCP (Shared)"
    :cards="workspaceMcpCards"
  />
</template>

<script setup lang="ts">
import WorkspaceSharedPage from "@/modules/resource/components/WorkspaceSharedPage.vue";
import { workspaceMcpCards } from "@/modules/resource/schemas/pageContent";
</script>
