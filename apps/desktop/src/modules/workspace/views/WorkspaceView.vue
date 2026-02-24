<template>
  <section class="workspace-manager">
    <h2>Workspace Manager</h2>

    <p v-if="adminForbidden" data-testid="admin-forbidden-message" class="notice">
      Admin access is not available in this workspace.
    </p>

    <p v-if="workspaceStore.error" class="error">{{ workspaceStore.error }}</p>
    <p v-if="authStore.error" class="error">{{ authStore.error }}</p>

    <div class="panel">
      <h3>Workspaces</h3>

      <ul class="workspace-list">
        <li v-for="workspace in workspaceStore.workspaces" :key="workspace.id" class="workspace-item">
          <div class="workspace-row">
            <strong>{{ workspace.name }}</strong>
            <span>{{ workspace.mode }}</span>
            <button type="button" @click="selectWorkspace(workspace.id)">Use</button>
          </div>

          <p v-if="workspace.mode === 'local'" data-testid="local-ready" class="status">Local Ready</p>

          <div v-if="workspace.mode === 'remote' && workspace.id === workspaceStore.currentWorkspaceId" class="remote-actions">
            <p class="status">Connection: {{ workspaceStore.connectionState }}</p>
            <p v-if="workspace.hub_url" class="status">Target Hub: {{ workspace.hub_url }}</p>

            <label>
              Username
              <input v-model="loginForm.username" type="text" placeholder="username" />
            </label>

            <label>
              Password
              <input v-model="loginForm.password" type="password" placeholder="password" />
            </label>

            <label>
              Token (optional)
              <input v-model="loginForm.token" type="text" placeholder="token" />
            </label>

            <button type="button" :disabled="workspace.login_disabled || authStore.loading" @click="submitLogin(workspace.id)">
              Login Remote
            </button>

            <p v-if="workspace.login_disabled" class="notice">Login is disabled for this workspace.</p>
          </div>
        </li>
      </ul>
    </div>

    <div class="panel">
      <h3>Add Remote Workspace</h3>
      <label>
        Name
        <input v-model="createForm.name" type="text" placeholder="Remote name" />
      </label>
      <label>
        Hub URL
        <input v-model="createForm.hubUrl" type="url" placeholder="http://127.0.0.1:8787" />
      </label>
      <button type="button" :disabled="workspaceStore.loading" @click="submitCreateRemote">Add Remote</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive } from "vue";
import { useRoute } from "vue-router";

import { createRemoteWorkspace, listWorkspaces, loginWorkspace } from "@/modules/workspace/services";
import { authStore, refreshMeForCurrentWorkspace, setWorkspaceToken } from "@/shared/stores/authStore";
import { setCurrentWorkspace, setWorkspaces, upsertWorkspace, workspaceStore } from "@/shared/stores/workspaceStore";
import { ApiError } from "@/shared/services/http";

const route = useRoute();

const createForm = reactive({
  name: "",
  hubUrl: ""
});

const loginForm = reactive({
  username: "",
  password: "",
  token: ""
});

const adminForbidden = computed(() => route.query.reason === "admin_forbidden");

onMounted(async () => {
  await loadWorkspaces();
});

async function loadWorkspaces(): Promise<void> {
  workspaceStore.loading = true;
  workspaceStore.error = "";

  try {
    const response = await listWorkspaces();
    setWorkspaces(response.items);

    if (workspaceStore.currentWorkspaceId !== "") {
      await refreshMeForCurrentWorkspace();
    }
  } catch (error) {
    workspaceStore.error = formatErrorMessage(error);
    workspaceStore.connectionState = "error";
  } finally {
    workspaceStore.loading = false;
  }
}

async function submitCreateRemote(): Promise<void> {
  workspaceStore.loading = true;
  workspaceStore.error = "";

  try {
    const created = await createRemoteWorkspace({
      name: createForm.name,
      hub_url: createForm.hubUrl
    });

    upsertWorkspace(created);
    createForm.name = "";
    createForm.hubUrl = "";
  } catch (error) {
    workspaceStore.error = formatErrorMessage(error);
  } finally {
    workspaceStore.loading = false;
  }
}

async function selectWorkspace(workspaceId: string): Promise<void> {
  setCurrentWorkspace(workspaceId);
  await refreshMeForCurrentWorkspace();
}

async function submitLogin(workspaceId: string): Promise<void> {
  authStore.error = "";

  try {
    const response = await loginWorkspace({
      workspace_id: workspaceId,
      username: loginForm.username || undefined,
      password: loginForm.password || undefined,
      token: loginForm.token || undefined
    });

    setWorkspaceToken(workspaceId, response.access_token, response.refresh_token);
    setCurrentWorkspace(workspaceId);
    await refreshMeForCurrentWorkspace();

    loginForm.username = "";
    loginForm.password = "";
    loginForm.token = "";
  } catch (error) {
    authStore.error = formatErrorMessage(error);
  }
}

function formatErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return `${error.message} (trace_id: ${error.traceId})`;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return "Unknown error";
}
</script>

<style scoped>
.workspace-manager {
  display: grid;
  gap: var(--component-space-md);
}

.panel {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-panel-bg);
  padding: var(--component-space-md);
  display: grid;
  gap: var(--component-space-sm);
}

.workspace-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: var(--component-space-sm);
}

.workspace-item {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  padding: var(--component-space-sm);
  display: grid;
  gap: var(--component-space-xs);
}

.workspace-row {
  display: flex;
  align-items: center;
  gap: var(--component-space-sm);
}

.remote-actions {
  display: grid;
  gap: var(--component-space-xs);
}

label {
  display: grid;
  gap: var(--component-space-xs);
}

input,
button {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-panel-bg);
  color: var(--component-text-main);
  padding: var(--component-space-xs) var(--component-space-sm);
  font: inherit;
}

.notice {
  margin: 0;
  color: var(--component-text-subtle);
}

.status {
  margin: 0;
  color: var(--component-text-subtle);
}

.error {
  margin: 0;
  color: var(--danger);
}
</style>
