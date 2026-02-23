import { beforeEach, describe, expect, it } from "vitest";
import { createMemoryHistory } from "vue-router";

import { createAppRouter, resetRouterInitForTests, routes } from "@/router";
import { authStore, resetAuthStore } from "@/shared/stores/authStore";
import { resetNavigationStore } from "@/shared/stores/navigationStore";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces, workspaceStore } from "@/shared/stores/workspaceStore";

describe("desktop routes", () => {
  beforeEach(() => {
    resetRouterInitForTests();
    resetWorkspaceStore();
    resetAuthStore();
    resetNavigationStore();
  });

  it("contains required 13 routes", () => {
    const routePaths = routes.map((route) => route.path);

    expect(routePaths).toContain("/main");
    expect(routePaths).toContain("/remote/account");
    expect(routePaths).toContain("/remote/members-roles");
    expect(routePaths).toContain("/remote/permissions-audit");
    expect(routePaths).toContain("/workspace/agent");
    expect(routePaths).toContain("/workspace/model");
    expect(routePaths).toContain("/workspace/rules");
    expect(routePaths).toContain("/workspace/skills");
    expect(routePaths).toContain("/workspace/mcp");
    expect(routePaths).toContain("/settings/theme");
    expect(routePaths).toContain("/settings/i18n");
    expect(routePaths).toContain("/settings/updates-diagnostics");
    expect(routePaths).toContain("/settings/general");
  });

  it("redirects remote route to main when workspace is local", async () => {
    setWorkspaces([
      {
        id: "ws_local",
        name: "Local",
        mode: "local",
        hub_url: null,
        is_default_local: true,
        created_at: "2026-02-23T00:00:00Z",
        login_disabled: true,
        auth_mode: "disabled"
      }
    ]);
    setCurrentWorkspace("ws_local");
    resetRouterInitForTests(true);

    const router = createAppRouter(createMemoryHistory());
    await router.push("/remote/account");
    await router.isReady();

    expect(router.currentRoute.value.path).toBe("/main");
    expect(router.currentRoute.value.query.reason).toBe("remote_required");
  });

  it("auto switches to remote workspace for remote routes", async () => {
    setWorkspaces([
      {
        id: "ws_local",
        name: "Local",
        mode: "local",
        hub_url: null,
        is_default_local: true,
        created_at: "2026-02-23T00:00:00Z",
        login_disabled: true,
        auth_mode: "disabled"
      },
      {
        id: "ws_remote",
        name: "Remote",
        mode: "remote",
        hub_url: "https://hub.example.com",
        is_default_local: false,
        created_at: "2026-02-23T00:00:00Z",
        login_disabled: false,
        auth_mode: "password_or_token"
      }
    ]);
    setCurrentWorkspace("ws_local");
    resetRouterInitForTests(true);

    const router = createAppRouter(createMemoryHistory());
    await router.push("/remote/account");
    await router.isReady();

    expect(router.currentRoute.value.path).toBe("/remote/account");
    expect(workspaceStore.currentWorkspaceId).toBe("ws_remote");
    expect(workspaceStore.mode).toBe("remote");
  });

  it("redirects admin route to account when admin capability is missing", async () => {
    setWorkspaces([
      {
        id: "ws_remote",
        name: "Remote",
        mode: "remote",
        hub_url: "https://hub.example.com",
        is_default_local: false,
        created_at: "2026-02-23T00:00:00Z",
        login_disabled: false,
        auth_mode: "password_or_token"
      }
    ]);
    setCurrentWorkspace("ws_remote");
    authStore.capabilities = {
      admin_console: false,
      resource_write: true,
      execution_control: true
    };
    resetRouterInitForTests(true);

    const router = createAppRouter(createMemoryHistory());
    await router.push("/remote/members-roles");
    await router.isReady();

    expect(router.currentRoute.value.path).toBe("/remote/account");
    expect(router.currentRoute.value.query.reason).toBe("admin_forbidden");
  });

  it("switches to local workspace when entering local settings routes", async () => {
    setWorkspaces([
      {
        id: "ws_local",
        name: "Local",
        mode: "local",
        hub_url: null,
        is_default_local: true,
        created_at: "2026-02-23T00:00:00Z",
        login_disabled: true,
        auth_mode: "disabled"
      },
      {
        id: "ws_remote",
        name: "Remote",
        mode: "remote",
        hub_url: "https://hub.example.com",
        is_default_local: false,
        created_at: "2026-02-23T00:00:00Z",
        login_disabled: false,
        auth_mode: "password_or_token"
      }
    ]);
    setCurrentWorkspace("ws_remote");
    resetRouterInitForTests(true);

    const router = createAppRouter(createMemoryHistory());
    await router.push("/settings/theme");
    await router.isReady();

    expect(router.currentRoute.value.path).toBe("/settings/theme");
    expect(workspaceStore.mode).toBe("local");
    expect(workspaceStore.currentWorkspaceId).toBe("ws_local");
  });
});
