import { afterEach, describe, expect, it, vi } from "vitest";

import {
  archiveSession,
  bootstrapAdmin,
  commitExecution,
  createModelConfig,
  deleteProject,
  discardExecution,
  exportExecutionPatch,
  getBootstrapStatus,
  getNavigation,
  listProjects,
  listRuntimeModelCatalog
} from "@/api/hubClient";

describe("hubClient", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("fetches bootstrap status from normalized server URL", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          setup_mode: true,
          allow_public_signup: false,
          message: "setup required"
        }),
        {
          status: 200,
          headers: {
            "content-type": "application/json"
          }
        }
      )
    );

    vi.stubGlobal("fetch", fetchMock);

    const response = await getBootstrapStatus("http://127.0.0.1:8787/");
    expect(response.setup_mode).toBe(true);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe("http://127.0.0.1:8787/v1/auth/bootstrap/status");
  });

  it("normalizes go hub bootstrap status payload", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          setup_completed: false
        }),
        {
          status: 200,
          headers: {
            "content-type": "application/json"
          }
        }
      )
    );

    vi.stubGlobal("fetch", fetchMock);

    const response = await getBootstrapStatus("http://127.0.0.1:8080");
    expect(response.setup_mode).toBe(true);
    expect(response.setup_completed).toBe(false);
  });

  it("posts bootstrap payload without bootstrap_token when omitted", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ token: "local-token" }), {
        status: 200,
        headers: {
          "content-type": "application/json"
        }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await bootstrapAdmin("http://127.0.0.1:8080", {
      email: "local-admin@goyais.local",
      password: "secret",
      display_name: "Local Admin"
    });

    const [, requestInit] = fetchMock.mock.calls[0] as [string, RequestInit];
    const body = JSON.parse(String(requestInit.body)) as Record<string, string>;
    expect(body.bootstrap_token).toBeUndefined();
    expect(body.email).toBe("local-admin@goyais.local");
    expect(body.display_name).toBe("Local Admin");
  });

  it("sends Authorization header for navigation request", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          workspace_id: "ws-1",
          menus: [],
          permissions: ["project:read"],
          feature_flags: {}
        }),
        {
          status: 200,
          headers: {
            "content-type": "application/json"
          }
        }
      )
    );

    vi.stubGlobal("fetch", fetchMock);

    await getNavigation("http://127.0.0.1:8787", "token-abc", "ws-1");

    const requestInit = fetchMock.mock.calls[0][1] as RequestInit;
    const headers = requestInit.headers as Headers;
    expect(headers.get("Authorization")).toBe("Bearer token-abc");
    expect(fetchMock.mock.calls[0][0]).toContain("workspace_id=ws-1");
  });

  it("sends workspace_id query for projects list", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ projects: [] }), {
        status: 200,
        headers: {
          "content-type": "application/json"
        }
      })
    );

    vi.stubGlobal("fetch", fetchMock);

    await listProjects("http://127.0.0.1:8787/", "token-abc", "ws-42");
    expect(fetchMock.mock.calls[0][0]).toBe("http://127.0.0.1:8787/v1/projects?workspace_id=ws-42");
  });

  it("posts model config payload to workspace endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          model_config: {
            model_config_id: "mc-1",
            workspace_id: "ws-1",
            provider: "openai",
            model: "gpt-4.1-mini",
            base_url: null,
            temperature: 0,
            max_tokens: 2048,
            secret_ref: "secret:abc",
            created_at: "2026-02-20T00:00:00.000Z",
            updated_at: "2026-02-20T00:00:00.000Z"
          }
        }),
        {
          status: 200,
          headers: {
            "content-type": "application/json"
          }
        }
      )
    );

    vi.stubGlobal("fetch", fetchMock);

    await createModelConfig("http://127.0.0.1:8787", "token-abc", "ws-1", {
      provider: "openai",
      model: "gpt-4.1-mini",
      api_key: "sk-test"
    });

    const [url, requestInit] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://127.0.0.1:8787/v1/model-configs?workspace_id=ws-1");
    expect(requestInit.method).toBe("POST");
    expect(requestInit.body).toContain("sk-test");
  });

  it("requests runtime model catalog with workspace query", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          provider: "openai",
          items: [],
          fetched_at: "2026-02-20T00:00:00.000Z",
          fallback_used: false
        }),
        {
          status: 200,
          headers: {
            "content-type": "application/json"
          }
        }
      )
    );

    vi.stubGlobal("fetch", fetchMock);

    await listRuntimeModelCatalog("http://127.0.0.1:8787", "token-abc", "ws-1", "mc-1");

    expect(fetchMock.mock.calls[0][0]).toBe(
      "http://127.0.0.1:8787/v1/runtime/model-configs/mc-1/models?workspace_id=ws-1"
    );
  });

  it("posts execution commit payload to workspace endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ commit_sha: "abc123" }), {
        status: 200,
        headers: {
          "content-type": "application/json"
        }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await commitExecution("http://127.0.0.1:8787", "token-abc", "ws-1", "exec-1", "feat: msg");
    expect(result.commit_sha).toBe("abc123");
    expect(fetchMock.mock.calls[0][0]).toBe("http://127.0.0.1:8787/v1/executions/exec-1/commit?workspace_id=ws-1");
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("POST");
    expect(String(init.body)).toContain("feat: msg");
  });

  it("downloads execution patch from workspace endpoint", async () => {
    const patchText = "--- a/README.md\n+++ b/README.md\n@@ -1 +1 @@\n-hello\n+world\n";
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(patchText, {
        status: 200,
        headers: {
          "content-type": "text/plain; charset=utf-8"
        }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await exportExecutionPatch("http://127.0.0.1:8787", "token-abc", "ws-1", "exec-1");
    expect(result).toBe(patchText);
    expect(fetchMock.mock.calls[0][0]).toBe("http://127.0.0.1:8787/v1/executions/exec-1/patch?workspace_id=ws-1");
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("GET");
  });

  it("calls execution discard endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ status: "ok" }), {
        status: 200,
        headers: {
          "content-type": "application/json"
        }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await discardExecution("http://127.0.0.1:8787", "token-abc", "ws-1", "exec-1");
    expect(fetchMock.mock.calls[0][0]).toBe("http://127.0.0.1:8787/v1/executions/exec-1/discard?workspace_id=ws-1");
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("DELETE");
  });

  it("handles 204 response for project deletion", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(null, { status: 204 })
    );
    vi.stubGlobal("fetch", fetchMock);

    await deleteProject("http://127.0.0.1:8787", "token-abc", "ws-1", "project-1");
    expect(fetchMock.mock.calls[0][0]).toBe("http://127.0.0.1:8787/v1/projects/project-1?workspace_id=ws-1");
  });

  it("handles 204 response for session deletion", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(null, { status: 204 })
    );
    vi.stubGlobal("fetch", fetchMock);

    await archiveSession("http://127.0.0.1:8787", "token-abc", "ws-1", "session-1");
    expect(fetchMock.mock.calls[0][0]).toBe("http://127.0.0.1:8787/v1/sessions/session-1?workspace_id=ws-1");
  });
});
