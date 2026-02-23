<template>
  <WorkspaceSharedPage
    active-key="workspace_skills"
    title="Skills 配置（共享）"
    account-subtitle="Workspace Config / Skills (Shared)"
    settings-subtitle="Local Settings / Skills (Shared)"
    :cards="workspaceSkillsCards"
  />
</template>

<script setup lang="ts">
import WorkspaceSharedPage from "@/modules/resource/components/WorkspaceSharedPage.vue";
import { workspaceSkillsCards } from "@/modules/resource/schemas/pageContent";
</script>
