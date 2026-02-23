import { flushPromises, mount } from "@vue/test-utils";
import { createMemoryHistory, createRouter } from "vue-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import WorkspaceView from "@/modules/workspace/views/WorkspaceView.vue";
import { resetAuthStore } from "@/shared/stores/authStore";
import { resetWorkspaceStore } from "@/shared/stores/workspaceStore";

describe("workspace view", () => {
  beforeEach(() => {
    resetWorkspaceStore();
    resetAuthStore();
    vi.unstubAllGlobals();
  });

  it("renders Local Ready", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({
        items: [
          {
            id: "ws_local",
            name: "Local",
            mode: "local",
            hub_url: null,
            is_default_local: true,
            created_at: "2026-02-22T00:00:00Z",
            login_disabled: true,
            auth_mode: "disabled"
          }
        ],
        next_cursor: null
      }))
      .mockResolvedValueOnce(jsonResponse({
        user_id: "local_user",
        display_name: "Local User",
        workspace_id: "ws_local",
        role: "admin",
        capabilities: {
          admin_console: true,
          resource_write: true,
          execution_control: true
        }
      }))
      .mockResolvedValueOnce(jsonResponse(buildPermissionSnapshot("admin")));

    vi.stubGlobal("fetch", fetchMock);

    const wrapper = await mountWithRouter();

    expect(wrapper.text()).toContain("Local Ready");
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("disables remote login button when login_disabled=true", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({
        items: [
          {
            id: "ws_local",
            name: "Local",
            mode: "local",
            hub_url: null,
            is_default_local: true,
            created_at: "2026-02-22T00:00:00Z",
            login_disabled: true,
            auth_mode: "disabled"
          },
          {
            id: "ws_remote_1",
            name: "Remote",
            mode: "remote",
            hub_url: "http://127.0.0.1:8787",
            is_default_local: false,
            created_at: "2026-02-22T00:00:00Z",
            login_disabled: true,
            auth_mode: "password_or_token"
          }
        ],
        next_cursor: null
      }))
      .mockResolvedValueOnce(jsonResponse({
        user_id: "local_user",
        display_name: "Local User",
        workspace_id: "ws_local",
        role: "admin",
        capabilities: {
          admin_console: true,
          resource_write: true,
          execution_control: true
        }
      }))
      .mockResolvedValueOnce(jsonResponse(buildPermissionSnapshot("admin")));

    vi.stubGlobal("fetch", fetchMock);

    const wrapper = await mountWithRouter();
    const useButtons = wrapper.findAll("button").filter((node) => node.text() === "Use");

    await useButtons[1].trigger("click");
    await flushPromises();

    const loginButton = wrapper.findAll("button").find((node) => node.text() === "Login Remote");
    expect(loginButton).toBeTruthy();
    expect(loginButton?.attributes("disabled")).toBeDefined();
    expect(wrapper.text()).toContain("Login is disabled for this workspace.");
  });

  it("can add remote workspace", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({
        items: [
          {
            id: "ws_local",
            name: "Local",
            mode: "local",
            hub_url: null,
            is_default_local: true,
            created_at: "2026-02-22T00:00:00Z",
            login_disabled: true,
            auth_mode: "disabled"
          }
        ],
        next_cursor: null
      }))
      .mockResolvedValueOnce(jsonResponse({
        user_id: "local_user",
        display_name: "Local User",
        workspace_id: "ws_local",
        role: "admin",
        capabilities: {
          admin_console: true,
          resource_write: true,
          execution_control: true
        }
      }))
      .mockResolvedValueOnce(jsonResponse(buildPermissionSnapshot("admin")))
      .mockResolvedValueOnce(
        jsonResponse({
          id: "ws_remote_new",
          name: "Remote New",
          mode: "remote",
          hub_url: "http://10.0.0.9:9000",
          is_default_local: false,
          created_at: "2026-02-22T00:00:00Z",
          login_disabled: false,
          auth_mode: "password_or_token"
        }, 201)
      );

    vi.stubGlobal("fetch", fetchMock);

    const wrapper = await mountWithRouter();
    await wrapper.get('input[placeholder="Remote name"]').setValue("Remote New");
    await wrapper.get('input[placeholder="http://127.0.0.1:8787"]').setValue("http://10.0.0.9:9000");

    const addButton = wrapper.findAll("button").find((node) => node.text() === "Add Remote");
    expect(addButton).toBeTruthy();
    await addButton?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Remote New");

    const postCall = fetchMock.mock.calls.find((call) => String(call[0]).endsWith("/v1/workspaces") && call[1]?.method === "POST");
    expect(postCall).toBeTruthy();
  });

  it("fetches remote me and permissions from control hub after login", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = init?.method ?? "GET";
      const headers = (init?.headers ?? {}) as Record<string, string>;
      const authHeader = headers.Authorization ?? "";

      if (url.endsWith("/v1/workspaces") && method === "GET") {
        return jsonResponse({
          items: [
            {
              id: "ws_local",
              name: "Local",
              mode: "local",
              hub_url: null,
              is_default_local: true,
              created_at: "2026-02-22T00:00:00Z",
              login_disabled: true,
              auth_mode: "disabled"
            },
            {
              id: "ws_remote_2",
              name: "Remote2",
              mode: "remote",
              hub_url: "http://10.0.0.9:9000",
              is_default_local: false,
              created_at: "2026-02-22T00:00:00Z",
              login_disabled: false,
              auth_mode: "password_or_token"
            }
          ],
          next_cursor: null
        });
      }

      if (url === "http://127.0.0.1:8787/v1/me" && method === "GET") {
        if (authHeader === "Bearer at_remote") {
          return jsonResponse({
            user_id: "remote_user",
            display_name: "Remote User",
            workspace_id: "ws_remote_2",
            role: "developer",
            capabilities: {
              admin_console: false,
              resource_write: true,
              execution_control: true
            }
          });
        }
        return jsonResponse({
          user_id: "local_user",
          display_name: "Local User",
          workspace_id: "ws_local",
          role: "admin",
          capabilities: {
            admin_console: true,
            resource_write: true,
            execution_control: true
          }
        });
      }

      if (url === "http://127.0.0.1:8787/v1/me/permissions" && method === "GET") {
        if (authHeader === "Bearer at_remote") {
          return jsonResponse(buildPermissionSnapshot("developer"));
        }
        return jsonResponse(buildPermissionSnapshot("admin"));
      }

      if (url === "http://127.0.0.1:8787/v1/auth/login" && method === "POST") {
        return jsonResponse({
          access_token: "at_remote",
          token_type: "bearer"
        });
      }

      return new Response("Not Found", { status: 404 });
    });

    vi.stubGlobal("fetch", fetchMock);

    const wrapper = await mountWithRouter();
    const useButtons = wrapper.findAll("button").filter((node) => node.text() === "Use");
    await useButtons[1].trigger("click");
    await flushPromises();

    await wrapper.get('input[placeholder="username"]').setValue("dev");
    await wrapper.get('input[placeholder="password"]').setValue("pw");

    const loginButton = wrapper.findAll("button").find((node) => node.text() === "Login Remote");
    expect(loginButton).toBeTruthy();
    await loginButton?.trigger("click");
    await flushPromises();

    const controlMeCall = fetchMock.mock.calls.find(
      (call) =>
        String(call[0]) === "http://127.0.0.1:8787/v1/me" &&
        ((call[1]?.headers as Record<string, string> | undefined)?.Authorization ?? "") === "Bearer at_remote"
    );
    expect(controlMeCall).toBeTruthy();

    const permissionsCall = fetchMock.mock.calls.find(
      (call) =>
        String(call[0]) === "http://127.0.0.1:8787/v1/me/permissions" &&
        ((call[1]?.headers as Record<string, string> | undefined)?.Authorization ?? "") === "Bearer at_remote"
    );
    expect(permissionsCall).toBeTruthy();
  });
});

async function mountWithRouter() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/workspace", component: WorkspaceView }]
  });

  await router.push("/workspace");
  await router.isReady();

  const wrapper = mount(WorkspaceView, {
    global: {
      plugins: [router]
    }
  });

  await flushPromises();
  return wrapper;
}

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: {
      "Content-Type": "application/json"
    }
  });
}

function buildPermissionSnapshot(role: "admin" | "developer") {
  return {
    role,
    permissions: role === "admin" ? ["*"] : ["project.read", "project.write", "conversation.read", "conversation.write"],
    menu_visibility: {
      main: "enabled",
      remote_account: "enabled",
      remote_members_roles: role === "admin" ? "enabled" : "hidden",
      remote_permissions_audit: role === "admin" ? "enabled" : "hidden",
      workspace_project_config: "enabled",
      workspace_agent: "enabled",
      workspace_model: "enabled",
      workspace_rules: "enabled",
      workspace_skills: "enabled",
      workspace_mcp: "enabled",
      settings_theme: "enabled",
      settings_i18n: "enabled",
      settings_general: "enabled"
    },
    action_visibility: {},
    policy_version: "test",
    generated_at: "2026-02-23T00:00:00Z"
  };
}
