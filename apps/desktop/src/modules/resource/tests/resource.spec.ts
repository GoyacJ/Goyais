import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { resetProjectStore } from "@/modules/project/store";
import { resetResourceStore } from "@/modules/resource/store";
import WorkspaceMcpView from "@/modules/resource/views/WorkspaceMcpView.vue";
import WorkspaceAgentView from "@/modules/resource/views/WorkspaceAgentView.vue";
import WorkspaceModelView from "@/modules/resource/views/WorkspaceModelView.vue";
import WorkspaceProjectConfigView from "@/modules/resource/views/WorkspaceProjectConfigView.vue";
import WorkspaceRulesView from "@/modules/resource/views/WorkspaceRulesView.vue";
import WorkspaceSkillsView from "@/modules/resource/views/WorkspaceSkillsView.vue";
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

    const catalogReloadCall = (global.fetch as ReturnType<typeof vi.fn>).mock.calls.find(([input, init]) => {
      const urlValue = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
      const path = new URL(urlValue, "http://127.0.0.1:8787").pathname;
      const method = (init?.method ?? "GET").toUpperCase();
      return method === "POST" && path === "/v1/workspaces/ws_local/model-catalog";
    });
    expect(catalogReloadCall).toBeTruthy();
    expect(String(catalogReloadCall?.[1]?.body ?? "")).toContain(`"source":"page_open"`);

    expect(wrapper.text()).toContain("模型列表");
    expect(wrapper.text()).not.toContain("测试诊断");
    expect(wrapper.text()).not.toContain("Catalog Root");
    expect(wrapper.text()).not.toContain("Catalog Source");
    expect(wrapper.text()).not.toContain("手动刷新目录");
    const addButton = wrapper.findAll("button").find((item) => item.text() === "新增模型");
    expect(addButton).toBeTruthy();
    await addButton?.trigger("click");
    expect(wrapper.text()).toContain("新增模型配置");
    expect(wrapper.text()).toContain("Endpoint");
    expect(wrapper.text()).not.toContain("认证：http_bearer");
    expect(wrapper.text()).not.toContain("Homepage");
    expect(wrapper.text()).not.toContain("Docs");
    expect(wrapper.text()).toContain("模型名称（可选）");
    expect(wrapper.text()).not.toContain("模型 ID（可手输）");
    expect(wrapper.text()).not.toContain("Params(JSON)");
  });

  it("loads and updates workspace agent config", async () => {
    const wrapper = mountView(WorkspaceAgentView);
    await flushPromises();

    expect(wrapper.text()).toContain("Max Model Turns");
    const turnsInput = wrapper.find("input[type='number']");
    expect(turnsInput.exists()).toBe(true);
    await turnsInput.setValue("12");
    await turnsInput.trigger("change");
    await flushPromises();

    const updateCalls = findFetchCalls("PUT", "/v1/workspaces/ws_local/agent-config");
    expect(updateCalls).toHaveLength(1);
    const [, init] = updateCalls[0] ?? [];
    const payload = JSON.parse(String(init?.body ?? "{}"));
    expect(payload.execution?.max_model_turns).toBe(12);
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

  it("deletes rule skill and project without confirm popup", async () => {
    const confirmSpy = vi.fn(() => true);
    vi.stubGlobal("confirm", confirmSpy);

    const rulesWrapper = mountView(WorkspaceRulesView);
    await flushPromises();
    const ruleDeleteButton = rulesWrapper.findAll("button").find((item) => item.text() === "删除");
    expect(ruleDeleteButton).toBeTruthy();
    await ruleDeleteButton?.trigger("click");
    await flushPromises();

    const skillsWrapper = mountView(WorkspaceSkillsView);
    await flushPromises();
    const skillDeleteButton = skillsWrapper.findAll("button").find((item) => item.text() === "删除");
    expect(skillDeleteButton).toBeTruthy();
    await skillDeleteButton?.trigger("click");
    await flushPromises();

    const projectWrapper = mountView(WorkspaceProjectConfigView);
    await flushPromises();
    const projectRemoveButton = projectWrapper.findAll("button").find((item) => item.text() === "移除");
    expect(projectRemoveButton).toBeTruthy();
    await projectRemoveButton?.trigger("click");
    await flushPromises();

    expect(confirmSpy).toHaveBeenCalledTimes(0);
    expect(findFetchCalls("DELETE", "/v1/workspaces/ws_local/resource-configs/rc_rule_1")).toHaveLength(1);
    expect(findFetchCalls("DELETE", "/v1/workspaces/ws_local/resource-configs/rc_skill_1")).toHaveLength(1);
    expect(findFetchCalls("DELETE", "/v1/projects/proj_alpha")).toHaveLength(1);
  });

  it("deletes mcp config directly without remove modal", async () => {
    const wrapper = mountView(WorkspaceMcpView);
    await flushPromises();

    const deleteButton = wrapper.findAll("button").find((item) => item.text() === "删除");
    expect(deleteButton).toBeTruthy();
    await deleteButton?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).not.toContain("删除 MCP 配置");
    expect(findFetchCalls("DELETE", "/v1/workspaces/ws_local/resource-configs/rc_mcp_1")).toHaveLength(1);
  });

  it("keeps model_config_ids unchanged when project binding returns non-config identifiers", async () => {
    const confirmSpy = vi.fn(() => true);
    vi.stubGlobal("confirm", confirmSpy);
    vi.stubGlobal("fetch", createApiFetchMock({ legacyProjectConfigModelIDs: true }));
    const wrapper = mountView(WorkspaceProjectConfigView);
    await flushPromises();

    const configButton = wrapper.findAll("button").find((item) => item.text() === "配置");
    expect(configButton).toBeTruthy();
    await configButton?.trigger("click");
    await flushPromises();

    const saveButton = wrapper.findAll("button").find((item) => item.text() === "保存");
    expect(saveButton).toBeTruthy();
    await saveButton?.trigger("click");
    await flushPromises();

    expect(confirmSpy).toHaveBeenCalledTimes(1);
    const updateCalls = findFetchCalls("PUT", "/v1/projects/proj_alpha/config");
    expect(updateCalls).toHaveLength(1);
    const [, updateInit] = updateCalls[0] ?? [];
    const body = JSON.parse(String(updateInit?.body ?? "{}"));
    expect(body.model_config_ids).toEqual(["gpt-5.3"]);
    expect(body.default_model_config_id).toBe("gpt-5.3");
  });

  it("does not save project binding when confirm is cancelled", async () => {
    const confirmSpy = vi.fn(() => false);
    vi.stubGlobal("confirm", confirmSpy);

    const wrapper = mountView(WorkspaceProjectConfigView);
    await flushPromises();

    const configButton = wrapper.findAll("button").find((item) => item.text() === "配置");
    expect(configButton).toBeTruthy();
    await configButton?.trigger("click");
    await flushPromises();

    const saveButton = wrapper.findAll("button").find((item) => item.text() === "保存");
    expect(saveButton).toBeTruthy();
    await saveButton?.trigger("click");
    await flushPromises();

    expect(confirmSpy).toHaveBeenCalledTimes(1);
    const updateCalls = findFetchCalls("PUT", "/v1/projects/proj_alpha/config");
    expect(updateCalls).toHaveLength(0);
  });

  it("shows validation error when project binding update is rejected", async () => {
    const confirmSpy = vi.fn(() => true);
    vi.stubGlobal("confirm", confirmSpy);
    vi.stubGlobal("fetch", createApiFetchMock({ rejectProjectConfigUpdate: true }));

    const wrapper = mountView(WorkspaceProjectConfigView);
    await flushPromises();
    const initialProjectBindingLoads = findFetchCalls("GET", "/v1/workspaces/ws_local/project-configs").length;

    const configButton = wrapper.findAll("button").find((item) => item.text() === "配置");
    expect(configButton).toBeTruthy();
    await configButton?.trigger("click");
    await flushPromises();

    const saveButton = wrapper.findAll("button").find((item) => item.text() === "保存");
    expect(saveButton).toBeTruthy();
    await saveButton?.trigger("click");
    await flushPromises();

    expect(confirmSpy).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain("VALIDATION_ERROR");
    expect(wrapper.text()).toContain("绑定配置");
    expect(findFetchCalls("PUT", "/v1/projects/proj_alpha/config")).toHaveLength(1);
    expect(findFetchCalls("GET", "/v1/workspaces/ws_local/project-configs")).toHaveLength(initialProjectBindingLoads);
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

function createApiFetchMock(options: { legacyProjectConfigModelIDs?: boolean; rejectProjectConfigUpdate?: boolean } = {}) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const urlValue =
      typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    const requestURL = new URL(urlValue, "http://127.0.0.1:8787");
    const method = (init?.method ?? (input instanceof Request ? input.method : "GET")).toUpperCase();
    const path = requestURL.pathname;

    if ((method === "GET" || method === "POST") && path === "/v1/workspaces/ws_local/model-catalog") {
      return jsonResponse({
        workspace_id: "ws_local",
        revision: 1,
        updated_at: "2026-02-24T00:00:00Z",
        source: "embedded://models.default.json",
        vendors: [
          {
            name: "OpenAI",
            homepage: "https://openai.com/api/",
            docs: "https://developers.openai.com/api/docs/models",
            base_url: "https://api.openai.com/v1",
            base_urls: {
              global: "https://api.openai.com/v1",
              mirror: "https://mirror.openai.com/v1"
            },
            auth: {
              type: "http_bearer",
              header: "Authorization",
              scheme: "Bearer",
              api_key_env: "OPENAI_API_KEY"
            },
            models: [{ id: "gpt-5.3", label: "GPT-5.3 (Default)", enabled: true }],
            notes: ["OpenAI models are managed by model catalog."]
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
            default_model_config_id: "rc_model_1",
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
      const modelBindingID = options.legacyProjectConfigModelIDs ? "gpt-5.3" : "rc_model_1";
      return jsonResponse([
        {
          project_id: "proj_alpha",
          project_name: "Alpha",
          config: {
            project_id: "proj_alpha",
            model_config_ids: [modelBindingID],
            default_model_config_id: modelBindingID,
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

    if (method === "GET" && path === "/v1/workspaces/ws_local/agent-config") {
      return jsonResponse({
        workspace_id: "ws_local",
        execution: {
          max_model_turns: 24
        },
        display: {
          show_process_trace: true,
          trace_detail_level: "verbose"
        },
        updated_at: "2026-02-24T00:00:00Z"
      });
    }

    if (method === "PATCH") {
      return jsonResponse({ ok: true });
    }
    if (method === "PUT" && path === "/v1/workspaces/ws_local/agent-config") {
      const parsed = JSON.parse(String(init?.body ?? "{}"));
      return jsonResponse({
        workspace_id: "ws_local",
        execution: {
          max_model_turns: parsed.execution?.max_model_turns ?? 24
        },
        display: {
          show_process_trace: parsed.display?.show_process_trace ?? true,
          trace_detail_level: parsed.display?.trace_detail_level ?? "verbose"
        },
        updated_at: "2026-02-24T00:00:00Z"
      });
    }
    if (method === "PUT" && path === "/v1/projects/proj_alpha/config" && options.rejectProjectConfigUpdate) {
      return jsonResponse(
        {
          code: "VALIDATION_ERROR",
          message: "model_config_id must be included in project model_config_ids",
          trace_id: "tr_bind_invalid"
        },
        400
      );
    }
    if (method === "PUT") {
      return jsonResponse({
        project_id: "proj_alpha",
        model_config_ids: ["rc_model_1"],
        default_model_config_id: "rc_model_1",
        rule_ids: ["rc_rule_1"],
        skill_ids: ["rc_skill_1"],
        mcp_ids: ["rc_mcp_1"],
        updated_at: "2026-02-24T00:00:00Z"
      });
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

function findFetchCalls(method: string, path: string) {
  return (global.fetch as ReturnType<typeof vi.fn>).mock.calls.filter(([input, init]) => {
    const urlValue = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    const requestURL = new URL(urlValue, "http://127.0.0.1:8787");
    const requestMethod = (init?.method ?? "GET").toUpperCase();
    return requestMethod === method.toUpperCase() && requestURL.pathname === path;
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
          model_id: "gpt-5.3"
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
