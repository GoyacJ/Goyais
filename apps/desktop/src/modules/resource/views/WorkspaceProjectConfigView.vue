<template>
  <WorkspaceSharedShell
    active-key="workspace_project_config"
    title="项目配置（共享）"
    account-subtitle="Workspace Config / Project Config (Shared)"
    settings-subtitle="Local Settings / Project Config (Shared)"
  >
    <section class="card">
      <h3>项目配置绑定</h3>
      <div class="row">
        <label>
          项目
          <select v-model="form.projectId" @change="loadProjectConfig">
            <option value="">请选择项目</option>
            <option v-for="project in projectStore.projects" :key="project.id" :value="project.id">
              {{ project.name }}
            </option>
          </select>
        </label>
        <label>
          模型
          <input v-model="form.modelId" type="text" placeholder="gpt-4.1" />
        </label>
      </div>

      <div class="row">
        <label>
          规则 IDs（逗号分隔）
          <input v-model="form.ruleIds" type="text" placeholder="rule_secure,rule_repo_guard" />
        </label>
      </div>

      <div class="row">
        <label>
          技能 IDs（逗号分隔）
          <input v-model="form.skillIds" type="text" placeholder="skill_review,skill_test" />
        </label>
      </div>

      <div class="row">
        <label>
          MCP IDs（逗号分隔）
          <input v-model="form.mcpIds" type="text" placeholder="mcp_git,mcp_fs" />
        </label>
      </div>

      <div class="actions">
        <button type="button" :disabled="form.projectId === ''" @click="saveConfig">保存项目配置</button>
      </div>
    </section>

    <section class="card tips">
      <h3>语义约束</h3>
      <ul>
        <li>Conversation 创建时自动继承 ProjectConfig。</li>
        <li>Conversation 可覆盖，但不会反写 ProjectConfig。</li>
        <li>本地资源共享到远程后，仍需工作区管理员审批。</li>
      </ul>
    </section>
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import { reactive, watch } from "vue";

import { projectStore, refreshProjects, updateProjectBinding } from "@/modules/project/store";
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import { workspaceStore } from "@/shared/stores/workspaceStore";

const form = reactive({
  projectId: "",
  modelId: "gpt-4.1",
  ruleIds: "",
  skillIds: "",
  mcpIds: ""
});

watch(
  () => workspaceStore.currentWorkspaceId,
  async () => {
    form.projectId = "";
    form.modelId = "gpt-4.1";
    form.ruleIds = "";
    form.skillIds = "";
    form.mcpIds = "";
    await refreshProjects();
    form.projectId = projectStore.activeProjectId || projectStore.projects[0]?.id || "";
    loadProjectConfig();
  },
  { immediate: true }
);

function loadProjectConfig(): void {
  if (form.projectId === "") {
    return;
  }

  const config = projectStore.projectConfigsByProjectId[form.projectId];
  if (!config) {
    form.modelId = "gpt-4.1";
    form.ruleIds = "";
    form.skillIds = "";
    form.mcpIds = "";
    return;
  }

  form.modelId = config.model_id ?? "";
  form.ruleIds = config.rule_ids.join(",");
  form.skillIds = config.skill_ids.join(",");
  form.mcpIds = config.mcp_ids.join(",");
}

async function saveConfig(): Promise<void> {
  if (form.projectId === "") {
    return;
  }

  await updateProjectBinding(form.projectId, {
    model_id: form.modelId.trim() || null,
    rule_ids: toList(form.ruleIds),
    skill_ids: toList(form.skillIds),
    mcp_ids: toList(form.mcpIds)
  });
}

function toList(raw: string): string[] {
  return raw
    .split(",")
    .map((item) => item.trim())
    .filter((item) => item !== "");
}
</script>

<style scoped>
.card {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-8);
}

.card h3 {
  margin: 0;
}

.row {
  display: grid;
  gap: var(--global-space-8);
}

label {
  display: grid;
  gap: var(--global-space-4);
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

input,
select {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  padding: var(--global-space-8);
}

.actions {
  display: flex;
  justify-content: flex-end;
}

.actions button {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  padding: var(--global-space-8) var(--global-space-12);
}

.tips ul {
  margin: 0;
  padding-left: var(--global-space-16);
  color: var(--semantic-text-muted);
  display: grid;
  gap: var(--global-space-4);
}
</style>
