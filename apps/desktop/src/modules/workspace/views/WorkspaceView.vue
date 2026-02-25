<template>
  <section class="workspace-manager grid gap-[var(--component-space-md)]">
    <h2>Workspace Manager</h2>

    <p v-if="adminForbidden" data-testid="admin-forbidden-message" class="notice m-0 text-[var(--component-text-subtle)]">
      Admin access is not available in this workspace.
    </p>

    <p v-if="workspaceStore.error" class="error m-0 text-[var(--danger)]">{{ workspaceStore.error }}</p>
    <p v-if="authStore.error" class="error m-0 text-[var(--danger)]">{{ authStore.error }}</p>

    <div
      class="panel grid gap-[var(--component-space-sm)] border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] p-[var(--component-space-md)]"
    >
      <h3>Workspaces</h3>

      <ul class="workspace-list m-0 grid list-none gap-[var(--component-space-sm)] p-0">
        <li
          v-for="workspace in workspaceStore.workspaces"
          :key="workspace.id"
          class="workspace-item grid gap-[var(--component-space-xs)] border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] p-[var(--component-space-sm)]"
        >
          <div class="workspace-row flex items-center gap-[var(--component-space-sm)]">
            <strong>{{ workspace.name }}</strong>
            <span>{{ workspace.mode }}</span>
            <button
              type="button"
              class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)] [font:inherit]"
              @click="selectWorkspace(workspace.id)"
            >
              Use
            </button>
          </div>

          <p v-if="workspace.mode === 'local'" data-testid="local-ready" class="status m-0 text-[var(--component-text-subtle)]">Local Ready</p>

          <div
            v-if="workspace.mode === 'remote' && workspace.id === workspaceStore.currentWorkspaceId"
            class="remote-actions grid gap-[var(--component-space-xs)]"
          >
            <p class="status m-0 text-[var(--component-text-subtle)]">Connection: {{ workspaceStore.connectionState }}</p>
            <p v-if="workspace.hub_url" class="status m-0 text-[var(--component-text-subtle)]">Target Hub: {{ workspace.hub_url }}</p>

            <label class="grid gap-[var(--component-space-xs)]">
              Username
              <input
                v-model="loginForm.username"
                type="text"
                placeholder="username"
                class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)] [font:inherit]"
              />
            </label>

            <label class="grid gap-[var(--component-space-xs)]">
              Password
              <input
                v-model="loginForm.password"
                type="password"
                placeholder="password"
                class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)] [font:inherit]"
              />
            </label>

            <label class="grid gap-[var(--component-space-xs)]">
              Token (optional)
              <input
                v-model="loginForm.token"
                type="text"
                placeholder="token"
                class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)] [font:inherit]"
              />
            </label>

            <button
              type="button"
              :disabled="workspace.login_disabled || authStore.loading"
              class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)] [font:inherit]"
              @click="submitLogin(workspace.id)"
            >
              Login Remote
            </button>

            <p v-if="workspace.login_disabled" class="notice m-0 text-[var(--component-text-subtle)]">Login is disabled for this workspace.</p>
          </div>
        </li>
      </ul>
    </div>

    <div
      class="panel grid gap-[var(--component-space-sm)] border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] p-[var(--component-space-md)]"
    >
      <h3>Add Remote Workspace</h3>
      <label class="grid gap-[var(--component-space-xs)]">
        Name
        <input
          v-model="createForm.name"
          type="text"
          placeholder="Remote name"
          class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)] [font:inherit]"
        />
      </label>
      <label class="grid gap-[var(--component-space-xs)]">
        Hub URL
        <input
          v-model="createForm.hubUrl"
          type="url"
          placeholder="http://127.0.0.1:8787"
          class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)] [font:inherit]"
        />
      </label>
      <button
        type="button"
        :disabled="workspaceStore.loading"
        class="border border-[var(--component-panel-border)] rounded-[var(--component-radius-sm)] bg-[var(--component-panel-bg)] px-[var(--component-space-sm)] py-[var(--component-space-xs)] text-[var(--component-text-main)] [font:inherit]"
        @click="submitCreateRemote"
      >
        Add Remote
      </button>
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
