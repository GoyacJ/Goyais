import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  controlRun,
  controlRunTask,
  getRunTaskById,
  getRunTaskGraph,
  listRunTasks
} from "@/modules/conversation/services";

describe("conversation run task services", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = init?.method ?? "GET";

      if (url.endsWith("/v1/runs/run_1/graph") && method === "GET") {
        return jsonResponse({
          run_id: "run_1",
          max_parallelism: 2,
          tasks: [],
          edges: []
        });
      }

      if (url.includes("/v1/runs/run_1/tasks") && method === "GET") {
        if (url.includes("/v1/runs/run_1/tasks/task_1")) {
          return jsonResponse({
            task_id: "task_1",
            run_id: "run_1",
            title: "Execution task_1",
            state: "queued",
            depends_on: [],
            children: [],
            retry_count: 0,
            max_retries: 0,
            created_at: "2026-03-02T00:00:00Z",
            updated_at: "2026-03-02T00:00:00Z"
          });
        }
        return jsonResponse({
          items: [],
          next_cursor: "next_1"
        });
      }

      if (url.endsWith("/v1/runs/run_1/tasks/task_1/control") && method === "POST") {
        return jsonResponse({
          ok: true,
          run_id: "run_1",
          task_id: "task_1",
          state: "cancelled",
          previous_state: "queued"
        });
      }

      if (url.endsWith("/v1/runs/run_1/control") && method === "POST") {
        return jsonResponse({
          ok: true,
          run_id: "run_1",
          state: "cancelled",
          previous_state: "queued"
        });
      }

      return jsonResponse(
        {
          code: "ROUTE_NOT_FOUND",
          message: "Not found",
          details: {},
          trace_id: "tr_not_found"
        },
        404
      );
    });
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads run task graph", async () => {
    const graph = await getRunTaskGraph("run_1");

    expect(graph.run_id).toBe("run_1");
    expect(graph.max_parallelism).toBe(2);
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/v1/runs/run_1/graph"),
      expect.objectContaining({ method: "GET" })
    );
  });

  it("lists run tasks with query options", async () => {
    const response = await listRunTasks("run_1", {
      state: "queued",
      cursor: "cursor_1",
      limit: 25.7
    });

    expect(response.next_cursor).toBe("next_1");
    const call = fetchMock.mock.calls.find(([url]) => String(url).includes("/v1/runs/run_1/tasks?"));
    expect(call).toBeDefined();
    const requestURL = String(call?.[0] ?? "");
    expect(requestURL).toContain("state=queued");
    expect(requestURL).toContain("cursor=cursor_1");
    expect(requestURL).toContain("limit=25");
  });

  it("loads run task detail", async () => {
    const task = await getRunTaskById("run_1", "task_1");

    expect(task.task_id).toBe("task_1");
    expect(task.state).toBe("queued");
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/v1/runs/run_1/tasks/task_1"),
      expect.objectContaining({ method: "GET" })
    );
  });

  it("controls run task with optional reason", async () => {
    const response = await controlRunTask("run_1", "task_1", "cancel", "user_requested");

    expect(response.ok).toBe(true);
    const call = fetchMock.mock.calls.find(([url, init]) => {
      return String(url).endsWith("/v1/runs/run_1/tasks/task_1/control") && (init?.method ?? "GET") === "POST";
    });
    expect(call).toBeDefined();
    const payload = JSON.parse(String(call?.[1]?.body ?? "{}")) as { action?: string; reason?: string };
    expect(payload).toEqual({ action: "cancel", reason: "user_requested" });
  });

  it("controls run via session-first alias", async () => {
    const response = await controlRun("run_1", "stop");

    expect(response.ok).toBe(true);
    const call = fetchMock.mock.calls.find(([url, init]) => {
      return String(url).endsWith("/v1/runs/run_1/control") && (init?.method ?? "GET") === "POST";
    });
    expect(call).toBeDefined();
    const payload = JSON.parse(String(call?.[1]?.body ?? "{}")) as { action?: string };
    expect(payload.action).toBe("stop");
  });
});

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: {
      "Content-Type": "application/json"
    }
  });
}
