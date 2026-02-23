<template>
  <WorkspaceSharedPage
    active-key="workspace_mcp"
    title="MCP 配置（共享）"
    remote-subtitle="Workspace Config / MCP (Shared)"
    local-subtitle="Local Settings / MCP (Shared)"
    :cards="workspaceMcpCards"
  />
</template>

<script setup lang="ts">
import WorkspaceSharedPage from "@/modules/resource/components/WorkspaceSharedPage.vue";
import { workspaceMcpCards } from "@/modules/resource/schemas/pageContent";
</script>
