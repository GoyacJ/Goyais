import { afterEach, describe, expect, it, vi } from "vitest";

import { createModelConfig, getBootstrapStatus, getNavigation, listProjects } from "@/api/hubClient";

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
});
