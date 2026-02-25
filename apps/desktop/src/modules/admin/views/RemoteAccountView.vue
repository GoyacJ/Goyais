<template>
  <AccountShell
    active-key="remote_account"
    title="账号信息"
    subtitle="Remote Workspace / Account"
    runtime-status-mode
    :runtime-conversation-status="workspaceStatus.conversationStatus.value"
    :runtime-connection-status="workspaceStatus.connectionStatus.value"
    :runtime-user-display-name="workspaceStatus.userDisplayName.value"
    :runtime-hub-url="workspaceStatus.hubURL.value"
  >
    <section class="card grid gap-[var(--global-space-8)] border border-[var(--semantic-border)] rounded-[var(--global-radius-12)] bg-[var(--semantic-surface)] p-[var(--global-space-12)]">
      <h3 class="m-0">当前账号</h3>
      <div class="kv-grid grid grid-cols-2 gap-[var(--global-space-8)]">
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">账号名</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ authStore.me?.display_name ?? '-' }}</strong>
        </p>
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">用户 ID</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ authStore.me?.user_id ?? '-' }}</strong>
        </p>
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">角色</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ authStore.me?.role ?? '-' }}</strong>
        </p>
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">执行控制</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ authStore.capabilities.execution_control ? 'enabled' : 'disabled' }}</strong>
        </p>
      </div>
    </section>

    <section class="card grid gap-[var(--global-space-8)] border border-[var(--semantic-border)] rounded-[var(--global-radius-12)] bg-[var(--semantic-surface)] p-[var(--global-space-12)]">
      <h3 class="m-0">当前工作区</h3>
      <div class="kv-grid grid grid-cols-2 gap-[var(--global-space-8)]">
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">workspace_id</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ workspace?.id ?? '-' }}</strong>
        </p>
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">workspace_name</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ workspace?.name ?? '-' }}</strong>
        </p>
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">mode</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ workspace?.mode ?? '-' }}</strong>
        </p>
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">hub</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ workspace?.hub_url ?? 'local://workspace' }}</strong>
        </p>
      </div>
    </section>

    <section class="card grid gap-[var(--global-space-8)] border border-[var(--semantic-border)] rounded-[var(--global-radius-12)] bg-[var(--semantic-surface)] p-[var(--global-space-12)]">
      <h3 class="m-0">连接与会话状态</h3>
      <div class="kv-grid grid grid-cols-2 gap-[var(--global-space-8)]">
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">连接状态</span>
          <strong class="text-[var(--global-font-size-12)]" :class="connectionClass">{{ connectionLabel }}</strong>
        </p>
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">当前会话状态</span>
          <strong class="text-[var(--global-font-size-12)]" :class="conversationClass">{{ conversationStatusLabel }}</strong>
        </p>
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">活跃 Conversation</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ activeConversationCount }}</strong>
        </p>
        <p class="m-0 grid gap-[var(--global-space-4)] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]">
          <span class="text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">排队执行</span>
          <strong class="text-[var(--global-font-size-12)] text-[var(--semantic-text)]">{{ queuedExecutionCount }}</strong>
        </p>
      </div>
    </section>
  </AccountShell>
</template>

<script setup lang="ts">
import { computed, watch } from "vue";

import { refreshAdminData } from "@/modules/admin/store";
import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import { projectStore, refreshConversationsForActiveProject } from "@/modules/project/store";
import { conversationStore } from "@/modules/conversation/store";
import { useWorkspaceStatusSync } from "@/shared/stores/workspaceStatusStore";
import AccountShell from "@/shared/shells/AccountShell.vue";

const workspace = computed(() =>
  workspaceStore.workspaces.find((item) => item.id === workspaceStore.currentWorkspaceId)
);

const workspaceStatus = useWorkspaceStatusSync({
  conversationId: computed(() => projectStore.activeConversationId)
});

const connectionLabel = computed(() => workspaceStatus.connectionStatus.value);
const conversationStatusLabel = computed(() => workspaceStatus.conversationStatus.value);

const connectionClass = computed(() => {
  if (connectionLabel.value === "connected") {
    return "text-[var(--semantic-success)]";
  }
  if (connectionLabel.value === "reconnecting") {
    return "text-[var(--semantic-warning)]";
  }
  return "text-[var(--semantic-danger)]";
});

const conversationClass = computed(() => {
  if (conversationStatusLabel.value === "running" || conversationStatusLabel.value === "done") {
    return "text-[var(--semantic-success)]";
  }
  if (conversationStatusLabel.value === "queued") {
    return "text-[var(--semantic-warning)]";
  }
  if (conversationStatusLabel.value === "error") {
    return "text-[var(--semantic-danger)]";
  }
  return "";
});

const activeConversationCount = computed(() => {
  return Object.values(projectStore.conversationsByProjectId).reduce((count, conversations) => count + conversations.length, 0);
});

const queuedExecutionCount = computed(() => {
  return Object.values(conversationStore.byConversationId).reduce((count, runtime) => {
    return count + runtime.executions.filter((execution) => execution.state === "queued").length;
  }, 0);
});

watch(
  () => workspaceStore.currentWorkspaceId,
  async () => {
    await refreshAdminData();
    await refreshConversationsForActiveProject();
  },
  { immediate: true }
);
</script>
