<template>
  <RemoteConfigLayout
    active-key="remote_account"
    :menu-entries="menuEntries"
    :scope-hint="scopeHint"
    title="账号信息"
    subtitle="Remote Workspace / Account"
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
        <p><span>活跃 Conversation</span><strong>{{ activeConversationCount }}</strong></p>
        <p><span>排队执行</span><strong>{{ queuedExecutionCount }}</strong></p>
        <p><span>默认管理员</span><strong>remote admin（全权限）</strong></p>
      </div>
    </section>
  </RemoteConfigLayout>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";

import { refreshAdminData } from "@/modules/admin/store";
import { useRemoteConfigMenu } from "@/shared/navigation/pageMenus";
import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import RemoteConfigLayout from "@/shared/layouts/RemoteConfigLayout.vue";
import { projectStore, refreshConversationsForActiveProject } from "@/modules/project/store";
import { conversationStore } from "@/modules/conversation/store";

const menuEntries = useRemoteConfigMenu();
const scopeHint = computed(() => "Remote 视图：显示成员与角色、权限与审计，并按 RBAC+ABAC 控制");

const workspace = computed(() =>
  workspaceStore.workspaces.find((item) => item.id === workspaceStore.currentWorkspaceId)
);

const connectionLabel = computed(() => {
  if (workspaceStore.connectionState === "ready") {
    return "connected";
  }
  if (workspaceStore.connectionState === "loading") {
    return "reconnecting";
  }
  return "disconnected";
});

const connectionClass = computed(() => {
  if (connectionLabel.value === "connected") {
    return "connected";
  }
  if (connectionLabel.value === "reconnecting") {
    return "reconnecting";
  }
  return "disconnected";
});

const activeConversationCount = computed(() => {
  return Object.values(projectStore.conversationsByProjectId).reduce((count, conversations) => count + conversations.length, 0);
});

const queuedExecutionCount = computed(() => {
  return Object.values(conversationStore.byConversationId).reduce((count, runtime) => {
    return count + runtime.executions.filter((execution) => execution.state === "queued").length;
  }, 0);
});

onMounted(async () => {
  await refreshAdminData();
  await refreshConversationsForActiveProject();
});
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
