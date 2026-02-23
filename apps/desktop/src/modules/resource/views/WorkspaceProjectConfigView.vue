<template>
  <WorkspaceSharedShell
    active-key="workspace_project_config"
    title="项目配置"
    account-subtitle="Workspace Config / Project Config"
    settings-subtitle="Local Settings / Project Config"
  >
    <p v-if="resourceStore.error" class="error">{{ resourceStore.error }}</p>

    <section class="card">
      <h3>项目导入与绑定</h3>
      <div class="import-row">
        <BaseInput v-model="form.importPath" placeholder="输入项目目录路径，例如 /Users/.../repo" />
        <button type="button" :disabled="!canWrite" @click="importDirectoryProject">目录导入</button>
      </div>

      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>项目</th>
              <th>目录</th>
              <th>模型绑定</th>
              <th>默认模型</th>
              <th>规则/技能/MCP</th>
              <th>动作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in projectRows" :key="row.id">
              <td>{{ row.name }}</td>
              <td>{{ row.repoPath }}</td>
              <td>{{ row.modelCount }}</td>
              <td>{{ row.defaultModelId }}</td>
              <td>{{ row.ruleCount }}/{{ row.skillCount }}/{{ row.mcpCount }}</td>
              <td>
                <div class="table-actions">
                  <button type="button" :disabled="!canWrite" @click="openProjectBinding(row.id)">配置</button>
                  <button type="button" class="danger" :disabled="!canWrite" @click="removeProjectById(row.id, row.name)">移除</button>
                </div>
              </td>
            </tr>
            <tr v-if="projectRows.length === 0">
              <td colspan="6" class="empty">当前工作区暂无项目</td>
            </tr>
          </tbody>
        </table>
      </div>
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
          <button type="button" @click="closeProjectBinding">取消</button>
          <button type="button" :disabled="!canWrite" @click="saveProjectBinding">保存</button>
        </div>
      </template>
    </BaseModal>
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import { useWorkspaceProjectConfigView } from "@/modules/resource/views/useWorkspaceProjectConfigView";
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import BaseInput from "@/shared/ui/BaseInput.vue";
import BaseModal from "@/shared/ui/BaseModal.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";

const {
  canWrite,
  closeProjectBinding,
  defaultModelOptions,
  form,
  importDirectoryProject,
  isChecked,
  mcpOptions,
  modelOptions,
  openProjectBinding,
  projectRows,
  removeProjectById,
  resourceStore,
  ruleOptions,
  saveProjectBinding,
  skillOptions,
  toggleListItem
} = useWorkspaceProjectConfigView();
</script>

<style scoped src="./WorkspaceProjectConfigView.css"></style>
