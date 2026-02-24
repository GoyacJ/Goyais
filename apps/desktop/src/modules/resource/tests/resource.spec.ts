import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { resetProjectStore } from "@/modules/project/store";
import { resetResourceStore } from "@/modules/resource/store";
import WorkspaceMcpView from "@/modules/resource/views/WorkspaceMcpView.vue";
import WorkspaceModelView from "@/modules/resource/views/WorkspaceModelView.vue";
import WorkspaceProjectConfigView from "@/modules/resource/views/WorkspaceProjectConfigView.vue";
import WorkspaceRulesView from "@/modules/resource/views/WorkspaceRulesView.vue";
import { resetAuthStore } from "@/shared/stores/authStore";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces } from "@/shared/stores/workspaceStore";

describe("resource module views", () => {
  beforeEach(() => {
    resetWorkspaceStore();
    resetAuthStore();
    resetProjectStore();
    resetResourceStore();

    const now = new Date().toISOString();
    setWorkspaces([
      {
        id: "ws_local",
        name: "Local Workspace",
        mode: "local",
        hub_url: null,
        is_default_local: true,
        created_at: now,
        login_disabled: true,
        auth_mode: "disabled"
      }
    ]);
    setCurrentWorkspace("ws_local");
    vi.stubGlobal("fetch", createApiFetchMock());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders model config table and modal action", async () => {
    const wrapper = mountView(WorkspaceModelView);
    await flushPromises();

    expect(wrapper.text()).toContain("模型列表");
    expect(wrapper.text()).not.toContain("测试诊断");
    expect(wrapper.text()).not.toContain("Catalog Root");
    expect(wrapper.text()).not.toContain("Catalog Source");
    expect(wrapper.text()).not.toContain("手动刷新目录");
    const addButton = wrapper.findAll("button").find((item) => item.text() === "新增模型");
    expect(addButton).toBeTruthy();
    await addButton?.trigger("click");
    expect(wrapper.text()).toContain("新增模型配置");
    expect(wrapper.text()).not.toContain("名称");
    expect(wrapper.text()).not.toContain("模型 ID（可手输）");
    expect(wrapper.text()).not.toContain("Params(JSON)");
  });

  it("renders rules page with markdown editor modal", async () => {
    const wrapper = mountView(WorkspaceRulesView);
    await flushPromises();

    expect(wrapper.text()).toContain("规则列表");
    const deleteButton = wrapper.findAll("button").find((item) => item.text() === "删除");
    expect(deleteButton?.classes()).toContain("variant-ghost");
    const addButton = wrapper.findAll("button").find((item) => item.text() === "新增规则");
    expect(addButton).toBeTruthy();
    await addButton?.trigger("click");
    expect(wrapper.text()).toContain("新增规则");
  });

  it("renders mcp cards page and export action", async () => {
    const wrapper = mountView(WorkspaceMcpView);
    await flushPromises();

    expect(wrapper.text()).toContain("新增 MCP");
    expect(wrapper.text()).toContain("MCP 配置");
    expect(wrapper.text()).toContain("最近探测");
    expect(wrapper.text()).toContain("连接详情");
    expect(wrapper.text()).toContain("单页");

    const addButton = wrapper.findAll("button").find((item) => item.text() === "新增 MCP");
    expect(addButton).toBeTruthy();
    await addButton?.trigger("click");
    expect(wrapper.text()).toContain("新增 MCP 配置");
  });

  it("renders project config table", async () => {
    const wrapper = mountView(WorkspaceProjectConfigView);
    await flushPromises();

    expect(wrapper.text()).toContain("项目导入与绑定");
    expect(wrapper.text()).toContain("添加项目");
    const removeButton = wrapper.findAll("button").find((item) => item.text() === "移除");
    expect(removeButton?.classes()).toContain("variant-ghost");
  });
});

function mountView(component: unknown) {
  return mount(component as never, {
    global: {
      stubs: {
        WorkspaceSharedShell: {
          template: "<div class='workspace-shared-shell-stub'><slot /></div>"
        }
      }
    }
  });
}

function createApiFetchMock() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const urlValue =
      typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    const requestURL = new URL(urlValue, "http://127.0.0.1:8787");
    const method = (init?.method ?? (input instanceof Request ? input.method : "GET")).toUpperCase();
    const path = requestURL.pathname;

    if (method === "GET" && path === "/v1/workspaces/ws_local/model-catalog") {
      return jsonResponse({
        workspace_id: "ws_local",
        revision: 1,
        updated_at: "2026-02-24T00:00:00Z",
        source: "embedded://models.default.json",
        vendors: [
          {
            name: "OpenAI",
            base_url: "https://api.openai.com/v1",
            models: [{ id: "gpt-4.1", label: "GPT-4.1", enabled: true }]
          }
        ]
      });
    }

    if (method === "GET" && path === "/v1/workspaces/ws_local/resource-configs") {
      const configType = requestURL.searchParams.get("type");
      return jsonResponse({
        items: buildResourceConfigItems(configType),
        next_cursor: null
      });
    }

    if (method === "GET" && path === "/v1/projects") {
      return jsonResponse({
        items: [
          {
            id: "proj_alpha",
            workspace_id: "ws_local",
            name: "Alpha",
            repo_path: "/tmp/alpha",
            is_git: true,
            default_model_id: "gpt-4.1",
            default_mode: "agent",
            current_revision: 0,
            created_at: "2026-02-24T00:00:00Z",
            updated_at: "2026-02-24T00:00:00Z"
          }
        ],
        next_cursor: null
      });
    }

    if (method === "GET" && path === "/v1/workspaces/ws_local/project-configs") {
      return jsonResponse([
        {
          project_id: "proj_alpha",
          project_name: "Alpha",
          config: {
            project_id: "proj_alpha",
            model_ids: ["rc_model_1"],
            default_model_id: "rc_model_1",
            rule_ids: ["rc_rule_1"],
            skill_ids: ["rc_skill_1"],
            mcp_ids: ["rc_mcp_1"],
            updated_at: "2026-02-24T00:00:00Z"
          }
        }
      ]);
    }

    if (method === "GET" && path === "/v1/workspaces/ws_local/mcps/export") {
      return jsonResponse({ workspace_id: "ws_local", mcps: [] });
    }

    if (method === "PATCH") {
      return jsonResponse({ ok: true });
    }
    if (method === "POST") {
      return jsonResponse({ ok: true });
    }
    if (method === "DELETE") {
      return new Response(null, { status: 204 });
    }
    return jsonResponse({});
  });
}

function buildResourceConfigItems(type: string | null) {
  if (type === "model") {
    return [
      {
        id: "rc_model_1",
        workspace_id: "ws_local",
        type: "model",
        enabled: true,
        model: {
          vendor: "OpenAI",
          model_id: "gpt-4.1"
        },
        created_at: "2026-02-24T00:00:00Z",
        updated_at: "2026-02-24T00:00:00Z"
      }
    ];
  }
  if (type === "rule") {
    return [
      {
        id: "rc_rule_1",
        workspace_id: "ws_local",
        type: "rule",
        name: "Secure Rule",
        enabled: true,
        rule: { content: "rule content" },
        created_at: "2026-02-24T00:00:00Z",
        updated_at: "2026-02-24T00:00:00Z"
      }
    ];
  }
  if (type === "skill") {
    return [
      {
        id: "rc_skill_1",
        workspace_id: "ws_local",
        type: "skill",
        name: "Review Skill",
        enabled: true,
        skill: { content: "skill content" },
        created_at: "2026-02-24T00:00:00Z",
        updated_at: "2026-02-24T00:00:00Z"
      }
    ];
  }
  if (type === "mcp") {
    return [
      {
        id: "rc_mcp_1",
        workspace_id: "ws_local",
        type: "mcp",
        name: "Github MCP",
        enabled: true,
        mcp: { transport: "http_sse", endpoint: "http://127.0.0.1:8000/sse" },
        created_at: "2026-02-24T00:00:00Z",
        updated_at: "2026-02-24T00:00:00Z"
      }
    ];
  }
  return [];
}

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: {
      "Content-Type": "application/json"
    }
  });
}
