<template>
  <WorkspaceSharedPage
    active-key="workspace_model"
    title="模型配置（共享）"
    remote-subtitle="Workspace Config / Models (Shared)"
    local-subtitle="Local Settings / Models (Shared)"
    :cards="workspaceModelCards"
  />
</template>

<script setup lang="ts">
import WorkspaceSharedPage from "@/modules/resource/components/WorkspaceSharedPage.vue";
import { workspaceModelCards } from "@/modules/resource/schemas/pageContent";
</script>
