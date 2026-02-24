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
    <section class="card">
      <h3>当前账号</h3>
      <div class="kv-grid">
        <p><span>账号名</span><strong>{{ authStore.me?.display_name ?? '-' }}</strong></p>
        <p><span>用户 ID</span><strong>{{ authStore.me?.user_id ?? '-' }}</strong></p>
        <p><span>角色</span><strong>{{ authStore.me?.role ?? '-' }}</strong></p>
        <p><span>执行控制</span><strong>{{ authStore.capabilities.execution_control ? 'enabled' : 'disabled' }}</strong></p>
      </div>
    </section>

    <section class="card">
      <h3>当前工作区</h3>
      <div class="kv-grid">
        <p><span>workspace_id</span><strong>{{ workspace?.id ?? '-' }}</strong></p>
        <p><span>workspace_name</span><strong>{{ workspace?.name ?? '-' }}</strong></p>
        <p><span>mode</span><strong>{{ workspace?.mode ?? '-' }}</strong></p>
        <p><span>hub</span><strong>{{ workspace?.hub_url ?? 'local://workspace' }}</strong></p>
      </div>
    </section>

    <section class="card">
      <h3>连接与会话状态</h3>
      <div class="kv-grid">
        <p><span>连接状态</span><strong :class="connectionClass">{{ connectionLabel }}</strong></p>
        <p><span>当前会话状态</span><strong :class="conversationClass">{{ conversationStatusLabel }}</strong></p>
        <p><span>活跃 Conversation</span><strong>{{ activeConversationCount }}</strong></p>
        <p><span>排队执行</span><strong>{{ queuedExecutionCount }}</strong></p>
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
    return "connected";
  }
  if (connectionLabel.value === "reconnecting") {
    return "reconnecting";
  }
  return "disconnected";
});

const conversationClass = computed(() => {
  if (conversationStatusLabel.value === "running" || conversationStatusLabel.value === "done") {
    return "connected";
  }
  if (conversationStatusLabel.value === "queued") {
    return "reconnecting";
  }
  if (conversationStatusLabel.value === "error") {
    return "disconnected";
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

.kv-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--global-space-8);
}

.kv-grid p {
  margin: 0;
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-4);
}

.kv-grid span {
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}

.kv-grid strong {
  color: var(--semantic-text);
  font-size: var(--global-font-size-12);
}

.connected {
  color: var(--semantic-success);
}

.reconnecting {
  color: var(--semantic-warning);
}

.disconnected {
  color: var(--semantic-danger);
}
</style>
