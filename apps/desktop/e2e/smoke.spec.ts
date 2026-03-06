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
  await page.addInitScript(() => {
    window.localStorage.setItem("goyais.locale", "zh-CN");
    Object.defineProperty(window, "EventSource", {
      configurable: true,
      value: undefined
    });
  });
});

test("main screen smoke renders primary frame", async ({ page }) => {
  await mockHubApi(page);
  await page.goto("/main");
  await expect(page).toHaveURL(/\/main$/);
  await expect(page.getByText("未选择会话")).toBeVisible();
  await expect(page.locator(".workspace-btn")).toContainText("Local Workspace");
});

test("remote route redirects to main in local workspace mode", async ({ page }) => {
  await mockHubApi(page);
  await page.goto("/remote/account");
  await expect(page).toHaveURL(/\/main\?reason=remote_required/);
  await expect(page.getByText("未选择会话")).toBeVisible();
});

test("settings theme route renders controls", async ({ page }) => {
  await mockHubApi(page);
  await page.goto("/settings/theme");
  await expect(page).toHaveURL(/\/settings\/theme$/);
  await expect(page.getByTestId("theme-mode-select")).toBeVisible();
  await expect(page.getByTestId("theme-reset-button")).toBeVisible();
});

test("main screen inspector run tab uses latest run context and preserves tasks on refresh failure", async ({ page }) => {
  await mockInspectorHubApi(page);
  await page.goto("/main");

  await expect(page.getByRole("button", { name: "Inspector Conversation" })).toBeVisible();
  await page.getByRole("button", { name: "Inspector Conversation" }).click();
  await page.getByRole("button", { name: "运行" }).click();

  await expect(page.getByText("任务: 1")).toBeVisible();
  await expect(page.getByRole("button", { name: "Latest task" })).toBeVisible();
  await page.getByRole("button", { name: "刷新任务" }).click();

  await expect(page.getByRole("button", { name: "Latest task" })).toBeVisible();
  await expect(page.getByText("TASK_LIST_UNAVAILABLE: task list temporarily unavailable (trace_id: tr_task_list_2)")).toBeVisible();
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

async function mockInspectorHubApi(page: Page): Promise<void> {
  const project = {
    id: "proj_1",
    workspace_id: "ws_local",
    name: "Inspector Project",
    repo_path: "/tmp/inspector-project",
    is_git: true,
    default_model_config_id: "rc_model_1",
    default_mode: "default",
    current_revision: 1,
    created_at: fixedTimestamp,
    updated_at: fixedTimestamp
  };
  const conversation = {
    id: "conv_1",
    workspace_id: "ws_local",
    project_id: "proj_1",
    name: "Inspector Conversation",
    queue_state: "idle",
    default_mode: "default",
    model_config_id: "rc_model_1",
    rule_ids: [],
    skill_ids: [],
    mcp_ids: [],
    base_revision: 1,
    active_run_id: null,
    created_at: fixedTimestamp,
    updated_at: fixedTimestamp
  };
  const runOld = {
    id: "run_old",
    workspace_id: "ws_local",
    session_id: "conv_1",
    message_id: "msg_1",
    state: "completed",
    mode: "default",
    model_id: "rc_model_1",
    mode_snapshot: "default",
    model_snapshot: {
      model_id: "rc_model_1"
    },
    project_revision_snapshot: 1,
    queue_index: 0,
    trace_id: "tr_old",
    created_at: "2026-02-24T00:00:00Z",
    updated_at: "2026-02-24T00:01:00Z"
  };
  const runLatest = {
    id: "run_latest",
    workspace_id: "ws_local",
    session_id: "conv_1",
    message_id: "msg_1",
    state: "failed",
    mode: "default",
    model_id: "rc_model_1",
    mode_snapshot: "default",
    model_snapshot: {
      model_id: "rc_model_1"
    },
    project_revision_snapshot: 1,
    queue_index: 1,
    trace_id: "tr_latest",
    created_at: "2026-02-25T00:00:00Z",
    updated_at: "2026-02-25T00:03:00Z"
  };
  const task = {
    task_id: "task_latest",
    run_id: "run_latest",
    title: "Latest task",
    state: "failed",
    depends_on: [],
    children: [],
    retry_count: 1,
    max_retries: 3,
    last_error: "tool timeout",
    artifact: {
      task_id: "task_latest",
      kind: "summary",
      summary: "latest failed summary"
    },
    created_at: "2026-02-25T00:00:00Z",
    updated_at: "2026-02-25T00:03:00Z"
  };
  let listRequests = 0;

  await page.route("http://127.0.0.1:8787/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method().toUpperCase();

    if (method !== "GET") {
      await route.fulfill(
        jsonResponse({
          code: "METHOD_NOT_ALLOWED",
          message: `method ${method} is not mocked in inspector smoke`,
          trace_id: "tr_inspector_method"
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
      await route.fulfill(jsonResponse({ items: [project], next_cursor: null }));
      return;
    }
    if (path === "/v1/projects/proj_1/sessions") {
      await route.fulfill(jsonResponse({ items: [conversation], next_cursor: null }));
      return;
    }
    if (path === "/v1/workspaces/ws_local/project-configs") {
      await route.fulfill(jsonResponse([]));
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
    if (path === "/v1/sessions/conv_1/input/catalog") {
      await route.fulfill(jsonResponse({ revision: "rev_1", commands: [], capabilities: [] }));
      return;
    }
    if (path === "/v1/sessions/conv_1") {
      await route.fulfill(jsonResponse({
        session: conversation,
        messages: [
          {
            id: "msg_1",
            session_id: "conv_1",
            role: "user",
            content: "Review inspector",
            queue_index: 0,
            created_at: fixedTimestamp
          }
        ],
        runs: [runOld, runLatest],
        snapshots: []
      }));
      return;
    }
    if (path === "/v1/sessions/conv_1/changeset") {
      await route.fulfill(jsonResponse({
        change_set_id: "cs_1",
        conversation_id: "conv_1",
        project_kind: "git",
        file_count: 0,
        entries: [],
        capability: {
          can_commit: true,
          can_discard: true,
          can_export: true,
          can_export_patch: true
        },
        suggested_message: {
          message: "chore: inspector"
        }
      }));
      return;
    }
    if (path === "/v1/runs/run_latest/graph") {
      await route.fulfill(jsonResponse({
        run_id: "run_latest",
        max_parallelism: 1,
        tasks: [task],
        edges: []
      }));
      return;
    }
    if (path === "/v1/runs/run_latest/tasks") {
      listRequests += 1;
      if (listRequests === 1) {
        await route.fulfill(jsonResponse({
          items: [task],
          next_cursor: null
        }));
        return;
      }
      await route.fulfill(jsonResponse({
        code: "TASK_LIST_UNAVAILABLE",
        message: "task list temporarily unavailable",
        trace_id: "tr_task_list_2"
      }, 500));
      return;
    }
    if (path === "/v1/runs/run_latest/tasks/task_latest") {
      await route.fulfill(jsonResponse(task));
      return;
    }

    await route.fulfill(
      jsonResponse(
        {
          code: "UNMOCKED_ENDPOINT",
          message: `no mock for ${method} ${path}`,
          trace_id: "tr_inspector_unmocked"
        },
        404
      )
    );
  });
}

function jsonResponse(body: unknown, status = 200): Parameters<Route["fulfill"]>[0] {
  return {
    status,
    contentType: "application/json",
    body: JSON.stringify(body)
  };
}
