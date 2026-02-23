<template>
  <WorkspaceSharedShell
    active-key="workspace_project_config"
    title="项目配置"
    account-subtitle="Workspace Config / Project Config"
    settings-subtitle="Local Settings / Project Config"
  >
    <section class="project-config-page">
      <div class="import-row">
        <BaseButton :disabled="!canWrite" variant="secondary" @click="addProject">添加项目</BaseButton>
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
              @click="removeProjectById((row as { id: string; name: string }).id, (row as { id: string; name: string }).name)"
            >
              移除
            </BaseButton>
          </div>
        </template>
      </ResourceConfigTable>
    </section>

    <BaseModal :open="form.open">
      <template #title>
        <h3 class="modal-title">项目绑定配置：{{ form.projectName }}</h3>
      </template>

      <div class="binding-form">
        <p>ProjectConfig 仅定义默认绑定；Conversation 可覆盖但不反写项目配置。</p>

        <section class="group">
          <h4>模型绑定（多选）</h4>
          <div class="checkbox-list">
            <label v-for="item in modelOptions" :key="item.id" class="checkbox-item">
              <input :checked="isChecked('modelIds', item.id)" type="checkbox" @change="toggleListItem('modelIds', item.id)" />
              {{ item.name }}
            </label>
            <span v-if="modelOptions.length === 0">暂无模型配置</span>
          </div>

          <BaseSelect
            v-model="form.defaultModelId"
            :options="[{ value: '', label: '不设置默认模型' }, ...defaultModelOptions]"
            :disabled="form.modelIds.length === 0"
          />
        </section>

        <section class="group">
          <h4>规则绑定</h4>
          <div class="checkbox-list">
            <label v-for="item in ruleOptions" :key="item.id" class="checkbox-item">
              <input :checked="isChecked('ruleIds', item.id)" type="checkbox" @change="toggleListItem('ruleIds', item.id)" />
              {{ item.name }}
            </label>
            <span v-if="ruleOptions.length === 0">暂无规则配置</span>
          </div>
        </section>

        <section class="group">
          <h4>技能绑定</h4>
          <div class="checkbox-list">
            <label v-for="item in skillOptions" :key="item.id" class="checkbox-item">
              <input :checked="isChecked('skillIds', item.id)" type="checkbox" @change="toggleListItem('skillIds', item.id)" />
              {{ item.name }}
            </label>
            <span v-if="skillOptions.length === 0">暂无技能配置</span>
          </div>
        </section>

        <section class="group">
          <h4>MCP 绑定</h4>
          <div class="checkbox-list">
            <label v-for="item in mcpOptions" :key="item.id" class="checkbox-item">
              <input :checked="isChecked('mcpIds', item.id)" type="checkbox" @change="toggleListItem('mcpIds', item.id)" />
              {{ item.name }}
            </label>
            <span v-if="mcpOptions.length === 0">暂无 MCP 配置</span>
          </div>
        </section>

        <p v-if="form.message !== ''" class="message">{{ form.message }}</p>
      </div>

      <template #footer>
        <div class="footer-actions">
          <BaseButton variant="ghost" @click="closeProjectBinding">取消</BaseButton>
          <BaseButton :disabled="!canWrite" variant="primary" @click="saveProjectBinding">保存</BaseButton>
        </div>
      </template>
    </BaseModal>
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import ResourceConfigTable from "@/modules/resource/components/ResourceConfigTable.vue";
import { useWorkspaceProjectConfigView } from "@/modules/resource/views/useWorkspaceProjectConfigView";
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

async function addProject(): Promise<void> {
  const directoryPath = await pickDirectoryPath();
  if (!directoryPath) {
    return;
  }
  await importDirectoryProject(directoryPath);
}
</script>

<style scoped src="./WorkspaceProjectConfigView.css"></style>
