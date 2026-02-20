import { afterEach, describe, expect, it, vi } from "vitest";

import { getBootstrapStatus, getNavigation } from "@/api/hubClient";

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
});
