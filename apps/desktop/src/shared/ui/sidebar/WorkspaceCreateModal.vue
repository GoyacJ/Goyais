<template>
  <div v-if="open" class="modal-mask" data-testid="workspace-create-modal" @click.self="closeModal">
    <div class="modal">
      <h4>新增工作区</h4>
      <label>
        Hub 地址
        <input v-model="workspaceForm.hub_url" type="url" placeholder="https://hub.example.com" />
      </label>
      <label>
        用户名
        <input v-model="workspaceForm.username" type="text" placeholder="admin" />
      </label>
      <label>
        密码
        <input v-model="workspaceForm.password" type="password" placeholder="******" />
      </label>
      <div class="modal-actions">
        <button type="button" @click="closeModal">取消</button>
        <button type="button" @click="submitWorkspaceCreate">创建</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from "vue";

const props = defineProps<{
  open: boolean;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "submit", payload: { hub_url: string; username: string; password: string }): void;
}>();

const workspaceForm = reactive({
  hub_url: "",
  username: "",
  password: ""
});

watch(
  () => props.open,
  (open) => {
    if (!open) {
      resetForm();
    }
  }
);

function closeModal(): void {
  resetForm();
  emit("close");
}

function submitWorkspaceCreate(): void {
  if (workspaceForm.hub_url.trim() === "" || workspaceForm.username.trim() === "" || workspaceForm.password.trim() === "") {
    return;
  }

  emit("submit", {
    hub_url: workspaceForm.hub_url.trim(),
    username: workspaceForm.username.trim(),
    password: workspaceForm.password
  });
  closeModal();
}

function resetForm(): void {
  workspaceForm.hub_url = "";
  workspaceForm.username = "";
  workspaceForm.password = "";
}
</script>

<style scoped>
.modal-mask {
  position: fixed;
  inset: 0;
  background: var(--component-modal-mask-bg);
  display: grid;
  place-items: center;
  z-index: 20;
}

.modal {
  width: 360px;
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  border: 1px solid var(--semantic-border);
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-8);
}

.modal h4 {
  margin: 0;
  font-size: var(--global-font-size-14);
}

.modal label {
  display: grid;
  gap: var(--global-space-4);
  font-size: var(--global-font-size-11);
  color: var(--semantic-text-muted);
}

.modal input {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  padding: var(--global-space-8);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--global-space-8);
}

.modal-actions button {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  padding: var(--global-space-8) var(--global-space-12);
}
</style>
