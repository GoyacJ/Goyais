<template>
  <BaseModal
    :open="open"
    aria-label="新增工作区"
    panel-class="!w-[360px]"
    close-on-backdrop
    initial-focus-selector="input[type='url']"
    data-testid="workspace-create-modal"
    @close="closeModal"
  >
    <template #title>
      <h4 class="m-0 text-[var(--global-font-size-14)]">新增工作区</h4>
    </template>

    <div class="grid gap-[var(--global-space-8)]">
      <label class="grid gap-[var(--global-space-4)] text-[var(--global-font-size-11)] text-[var(--semantic-text-muted)]">
        Hub 地址
        <input
          v-model="workspaceForm.hub_url"
          type="url"
          placeholder="https://hub.example.com"
          class="border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)] text-[var(--semantic-text)]"
        />
      </label>
      <label class="grid gap-[var(--global-space-4)] text-[var(--global-font-size-11)] text-[var(--semantic-text-muted)]">
        用户名
        <input
          v-model="workspaceForm.username"
          type="text"
          placeholder="admin"
          class="border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)] text-[var(--semantic-text)]"
        />
      </label>
      <label class="grid gap-[var(--global-space-4)] text-[var(--global-font-size-11)] text-[var(--semantic-text-muted)]">
        密码
        <input
          v-model="workspaceForm.password"
          type="password"
          placeholder="******"
          class="border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)] text-[var(--semantic-text)]"
        />
      </label>
    </div>

    <template #footer>
      <div class="flex justify-end gap-[var(--global-space-8)]">
        <button
          type="button"
          class="border-0 rounded-[var(--global-radius-8)] bg-[var(--semantic-surface-2)] px-[var(--global-space-12)] py-[var(--global-space-8)] text-[var(--semantic-text)]"
          @click="closeModal"
        >
          取消
        </button>
        <button
          type="button"
          class="border-0 rounded-[var(--global-radius-8)] bg-[var(--semantic-surface-2)] px-[var(--global-space-12)] py-[var(--global-space-8)] text-[var(--semantic-text)]"
          @click="submitWorkspaceCreate"
        >
          创建
        </button>
      </div>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { reactive, watch } from "vue";

import BaseModal from "@/shared/ui/BaseModal.vue";

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
