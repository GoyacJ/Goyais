<template>
  <WorkspaceSharedShell
    active-key="workspace_project_config"
    title="项目配置"
    account-subtitle="Workspace Config / Project Config"
    settings-subtitle="Local Settings / Project Config"
  >
    <section class="project-config-page">
      <div class="import-row">
        <BaseButton :disabled="!canWrite || !supportsDirectoryImport" variant="secondary" @click="addProject">添加项目</BaseButton>
      </div>

      <ResourceConfigTable
        title="项目导入与绑定"
        :columns="columns"
        :rows="projectRows as Array<Record<string, unknown>>"
        :empty-text="tableEmptyText"
        :show-search="false"
        :show-add="false"
      >
        <template #cell-actions="{ row }">
          <div class="table-actions">
            <BaseButton :disabled="!canWrite" variant="ghost" @click="openProjectBinding((row as { id: string }).id)">配置</BaseButton>
            <BaseButton
              :disabled="!canWrite"
              variant="ghost"
              @click="removeProjectById((row as { id: string }).id)"
            >
              移除
            </BaseButton>
          </div>
        </template>
      </ResourceConfigTable>
    </section>

    <BaseModal :open="form.open" class="project-binding-modal" @close="closeProjectBinding">
      <template #title>
        <div class="modal-header">
          <h3 class="modal-title">项目绑定配置</h3>
          <p class="modal-subtitle">{{ form.projectName }}</p>
        </div>
      </template>

      <div class="binding-form">
        <section class="binding-intro">
          <p class="intro-title">绑定说明</p>
          <p class="intro-description">ProjectConfig 仅定义默认绑定；Conversation 可覆盖但不反写项目配置。</p>
        </section>

        <section class="default-model-panel">
          <div class="default-model-header">
            <h4>默认模型</h4>
            <span>{{ form.modelIds.length }} / {{ modelOptions.length }} 已选</span>
          </div>
          <BaseSelect
            v-model="form.defaultModelId"
            :options="[{ value: '', label: '不设置默认模型' }, ...defaultModelOptions]"
            :disabled="form.modelIds.length === 0"
          />
          <p class="default-model-hint">仅可从“模型绑定”中选择默认模型。</p>
        </section>

        <div class="group-grid">
          <section class="group">
            <header class="group-head">
              <h4>模型绑定</h4>
              <div class="group-tools">
                <span class="group-meta">{{ form.modelIds.length }} / {{ modelOptions.length }}</span>
                <div class="group-actions">
                  <button class="mini-action" type="button" :disabled="!canWrite || modelOptions.length === 0" @click="setGroupSelection('modelIds', 'all')">
                    全选
                  </button>
                  <button class="mini-action" type="button" :disabled="!canWrite || form.modelIds.length === 0" @click="setGroupSelection('modelIds', 'none')">
                    清空
                  </button>
                </div>
              </div>
            </header>
            <div class="checkbox-list">
              <label v-for="item in modelOptions" :key="item.id" class="checkbox-item">
                <input :checked="isChecked('modelIds', item.id)" type="checkbox" @change="toggleListItem('modelIds', item.id)" />
                <span class="checkbox-label">{{ item.name }}</span>
              </label>
              <span v-if="modelOptions.length === 0" class="empty-item">暂无模型配置</span>
            </div>
          </section>

          <section class="group">
            <header class="group-head">
              <h4>规则绑定</h4>
              <div class="group-tools">
                <span class="group-meta">{{ form.ruleIds.length }} / {{ ruleOptions.length }}</span>
                <div class="group-actions">
                  <button class="mini-action" type="button" :disabled="!canWrite || ruleOptions.length === 0" @click="setGroupSelection('ruleIds', 'all')">
                    全选
                  </button>
                  <button class="mini-action" type="button" :disabled="!canWrite || form.ruleIds.length === 0" @click="setGroupSelection('ruleIds', 'none')">
                    清空
                  </button>
                </div>
              </div>
            </header>
            <div class="checkbox-list">
              <label v-for="item in ruleOptions" :key="item.id" class="checkbox-item">
                <input :checked="isChecked('ruleIds', item.id)" type="checkbox" @change="toggleListItem('ruleIds', item.id)" />
                <span class="checkbox-label">{{ item.name }}</span>
              </label>
              <span v-if="ruleOptions.length === 0" class="empty-item">暂无规则配置</span>
            </div>
          </section>

          <section class="group">
            <header class="group-head">
              <h4>技能绑定</h4>
              <div class="group-tools">
                <span class="group-meta">{{ form.skillIds.length }} / {{ skillOptions.length }}</span>
                <div class="group-actions">
                  <button class="mini-action" type="button" :disabled="!canWrite || skillOptions.length === 0" @click="setGroupSelection('skillIds', 'all')">
                    全选
                  </button>
                  <button class="mini-action" type="button" :disabled="!canWrite || form.skillIds.length === 0" @click="setGroupSelection('skillIds', 'none')">
                    清空
                  </button>
                </div>
              </div>
            </header>
            <div class="checkbox-list">
              <label v-for="item in skillOptions" :key="item.id" class="checkbox-item">
                <input :checked="isChecked('skillIds', item.id)" type="checkbox" @change="toggleListItem('skillIds', item.id)" />
                <span class="checkbox-label">{{ item.name }}</span>
              </label>
              <span v-if="skillOptions.length === 0" class="empty-item">暂无技能配置</span>
            </div>
          </section>

          <section class="group">
            <header class="group-head">
              <h4>MCP 绑定</h4>
              <div class="group-tools">
                <span class="group-meta">{{ form.mcpIds.length }} / {{ mcpOptions.length }}</span>
                <div class="group-actions">
                  <button class="mini-action" type="button" :disabled="!canWrite || mcpOptions.length === 0" @click="setGroupSelection('mcpIds', 'all')">
                    全选
                  </button>
                  <button class="mini-action" type="button" :disabled="!canWrite || form.mcpIds.length === 0" @click="setGroupSelection('mcpIds', 'none')">
                    清空
                  </button>
                </div>
              </div>
            </header>
            <div class="checkbox-list">
              <label v-for="item in mcpOptions" :key="item.id" class="checkbox-item">
                <input :checked="isChecked('mcpIds', item.id)" type="checkbox" @change="toggleListItem('mcpIds', item.id)" />
                <span class="checkbox-label">{{ item.name }}</span>
              </label>
              <span v-if="mcpOptions.length === 0" class="empty-item">暂无 MCP 配置</span>
            </div>
          </section>
        </div>

        <p v-if="form.message !== ''" class="message">{{ form.message }}</p>
      </div>

      <template #footer>
        <div class="footer-actions">
          <p class="footer-summary">
            已绑定：模型 {{ form.modelIds.length }} · 规则 {{ form.ruleIds.length }} · 技能 {{ form.skillIds.length }} · MCP {{ form.mcpIds.length }}
          </p>
          <div class="footer-buttons">
            <BaseButton variant="ghost" @click="closeProjectBinding">取消</BaseButton>
            <BaseButton :disabled="!canWrite" variant="primary" @click="saveProjectBinding">保存</BaseButton>
          </div>
        </div>
      </template>
    </BaseModal>
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import ResourceConfigTable from "@/modules/resource/components/ResourceConfigTable.vue";
import { useWorkspaceProjectConfigView } from "@/modules/resource/views/useWorkspaceProjectConfigView";
import { isRuntimeCapabilitySupported } from "@/shared/runtime";
import { pickDirectoryPath } from "@/shared/services/directoryPicker";
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import BaseButton from "@/shared/ui/BaseButton.vue";
import BaseModal from "@/shared/ui/BaseModal.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";

const {
  canWrite,
  closeProjectBinding,
  columns,
  defaultModelOptions,
  form,
  importDirectoryProject,
  isChecked,
  mcpOptions,
  modelOptions,
  openProjectBinding,
  projectRows,
  tableEmptyText,
  removeProjectById,
  ruleOptions,
  saveProjectBinding,
  skillOptions,
  toggleListItem
} = useWorkspaceProjectConfigView();
const supportsDirectoryImport = isRuntimeCapabilitySupported("supportsDirectoryImport");

type BindingField = "modelIds" | "ruleIds" | "skillIds" | "mcpIds";

async function addProject(): Promise<void> {
  if (!supportsDirectoryImport) {
    return;
  }
  const directoryPath = await pickDirectoryPath();
  if (!directoryPath) {
    return;
  }
  await importDirectoryProject(directoryPath);
}

function setGroupSelection(field: BindingField, mode: "all" | "none"): void {
  const nextIDs = mode === "all" ? resolveGroupOptionIDs(field) : [];
  if (field === "modelIds") {
    form.modelIds = nextIDs;
    if (form.defaultModelId !== "" && !nextIDs.includes(form.defaultModelId)) {
      form.defaultModelId = "";
    }
    if (form.defaultModelId === "" && nextIDs.length > 0) {
      form.defaultModelId = nextIDs[0] ?? "";
    }
    return;
  }
  if (field === "ruleIds") {
    form.ruleIds = nextIDs;
    return;
  }
  if (field === "skillIds") {
    form.skillIds = nextIDs;
    return;
  }
  form.mcpIds = nextIDs;
}

function resolveGroupOptionIDs(field: BindingField): string[] {
  if (field === "modelIds") {
    return modelOptions.value.map((item) => item.id);
  }
  if (field === "ruleIds") {
    return ruleOptions.value.map((item) => item.id);
  }
  if (field === "skillIds") {
    return skillOptions.value.map((item) => item.id);
  }
  return mcpOptions.value.map((item) => item.id);
}
</script>

<style scoped src="./WorkspaceProjectConfigView.css"></style>
