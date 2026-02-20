import { afterEach, describe, expect, it, vi } from "vitest";

import {
  confirmToolCall,
  createRun,
  subscribeRunEvents
} from "@/api/runtimeGatewayClient";

describe("runtimeGatewayClient", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("creates run via hub gateway with workspace_id query and bearer token", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ run_id: "run-1" }), {
        status: 200,
        headers: { "content-type": "application/json" }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await createRun("http://127.0.0.1:8787", "token-abc", "ws-1", {
      project_id: "p1",
      session_id: "s1",
      input: "hello",
      model_config_id: "mc1",
      workspace_path: "/tmp/work",
      options: { use_worktree: false }
    });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://127.0.0.1:8787/v1/runtime/runs?workspace_id=ws-1");
    const headers = new Headers(init.headers);
    expect(headers.get("Authorization")).toBe("Bearer token-abc");
  });

  it("posts tool confirmation to strict endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "content-type": "application/json" }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await confirmToolCall("http://127.0.0.1:8787", "token-abc", "ws-2", "run-1", "call-1", true);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://127.0.0.1:8787/v1/runtime/tool-confirmations?workspace_id=ws-2");
    expect(init.method).toBe("POST");
  });

  it("parses SSE stream events from hub gateway", async () => {
    const encoder = new TextEncoder();
    const stream = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(encoder.encode('data: {"event_id":"evt-1","type":"plan"}\n\n'));
        controller.enqueue(encoder.encode('data: {"event_id":"evt-2","type":"done"}\n\n'));
        controller.close();
      }
    });

    const fetchMock = vi.fn().mockResolvedValue(
      new Response(stream, {
        status: 200,
        headers: { "content-type": "text/event-stream" }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    const events: string[] = [];
    const sub = subscribeRunEvents(
      "http://127.0.0.1:8787",
      "token-abc",
      "ws-3",
      "run-1",
      (event) => {
        events.push(event.event_id);
      }
    );

    for (let i = 0; i < 20 && events.length < 2; i += 1) {
      // Allow async reader loop to consume chunks.
      await new Promise((resolve) => setTimeout(resolve, 5));
    }
    sub.close();

    expect(events).toEqual(["evt-1", "evt-2"]);
    const [url] = fetchMock.mock.calls[0] as [string];
    expect(url).toContain("/v1/runtime/runs/run-1/events?workspace_id=ws-3");
  });
});
