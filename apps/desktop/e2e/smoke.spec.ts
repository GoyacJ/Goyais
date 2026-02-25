import { expect, test, type Page, type Route } from "@playwright/test";

const fixedTimestamp = "2026-02-25T00:00:00Z";

const localWorkspace = {
  id: "ws_local",
  name: "Local Workspace",
  mode: "local",
  hub_url: null,
  is_default_local: true,
  created_at: fixedTimestamp,
  login_disabled: true,
  auth_mode: "disabled"
};

const localMe = {
  user_id: "local_user",
  display_name: "local-user",
  workspace_id: "ws_local",
  role: "admin",
  capabilities: {
    admin_console: true,
    resource_write: true,
    execution_control: true
  }
};

const permissionSnapshot = {
  role: "admin",
  permissions: ["*"],
  menu_visibility: {},
  action_visibility: {},
  policy_version: "e2e",
  generated_at: fixedTimestamp
};

const workspaceStatus = {
  workspace_id: "ws_local",
  conversation_status: "stopped",
  hub_url: "local://workspace",
  connection_status: "connected",
  user_display_name: "local-user",
  updated_at: fixedTimestamp
};

test.beforeEach(async ({ page }) => {
  await mockHubApi(page);
});

test("main screen smoke renders primary frame", async ({ page }) => {
  await page.goto("/main");
  await expect(page).toHaveURL(/\/main$/);
  await expect(page.getByText("未选择对话")).toBeVisible();
  await expect(page.locator(".workspace-btn")).toContainText("Local Workspace");
});

test("remote route redirects to main in local workspace mode", async ({ page }) => {
  await page.goto("/remote/account");
  await expect(page).toHaveURL(/\/main\?reason=remote_required/);
  await expect(page.getByText("未选择对话")).toBeVisible();
});

test("settings theme route renders controls", async ({ page }) => {
  await page.goto("/settings/theme");
  await expect(page).toHaveURL(/\/settings\/theme$/);
  await expect(page.getByTestId("theme-mode-select")).toBeVisible();
  await expect(page.getByTestId("theme-reset-button")).toBeVisible();
});

async function mockHubApi(page: Page): Promise<void> {
  await page.route("http://127.0.0.1:8787/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method().toUpperCase();

    if (method !== "GET") {
      await route.fulfill(
        jsonResponse({
          code: "METHOD_NOT_ALLOWED",
          message: `method ${method} is not mocked in e2e smoke`,
          trace_id: "tr_e2e_method"
        }, 405)
      );
      return;
    }

    const path = url.pathname;
    if (path === "/v1/workspaces") {
      await route.fulfill(jsonResponse({ items: [localWorkspace], next_cursor: null }));
      return;
    }
    if (path === "/v1/me") {
      await route.fulfill(jsonResponse(localMe));
      return;
    }
    if (path === "/v1/me/permissions") {
      await route.fulfill(jsonResponse(permissionSnapshot));
      return;
    }
    if (path === "/v1/projects") {
      await route.fulfill(jsonResponse({ items: [], next_cursor: null }));
      return;
    }
    if (path === "/v1/workspaces/ws_local/resource-configs") {
      await route.fulfill(jsonResponse({ items: [], next_cursor: null }));
      return;
    }
    if (path === "/v1/workspaces/ws_local/status") {
      await route.fulfill(jsonResponse(workspaceStatus));
      return;
    }

    await route.fulfill(
      jsonResponse(
        {
          code: "UNMOCKED_ENDPOINT",
          message: `no mock for ${method} ${path}`,
          trace_id: "tr_e2e_unmocked"
        },
        404
      )
    );
  });
}

function jsonResponse(body: Record<string, unknown>, status = 200): Parameters<Route["fulfill"]>[0] {
  return {
    status,
    contentType: "application/json",
    body: JSON.stringify(body)
  };
}
