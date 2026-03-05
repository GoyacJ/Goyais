import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  controlConversationRunTask,
  ensureConversationRuntime,
  loadConversationRunTaskById,
  loadConversationRunTaskGraph,
  loadConversationRunTasks,
  resetConversationStore
} from "@/modules/conversation/store";
import type { Session } from "@/shared/types/api";

const conversation: Session = {
  id: "conv_run_tasks",
  workspace_id: "ws_local",
  project_id: "proj_1",
  name: "Run Tasks",
  queue_state: "running",
  default_mode: "default",
  model_config_id: "rc_model_1",
  rule_ids: [],
  skill_ids: [],
  mcp_ids: [],
  base_revision: 0,
  active_execution_id: "exec_active",
  created_at: "2026-03-02T00:00:00Z",
  updated_at: "2026-03-02T00:00:00Z"
};

describe("conversation run task actions", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    resetConversationStore();
    fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const method = init?.method ?? "GET";

      if (url.endsWith("/v1/runs/exec_active/graph") && method === "GET") {
        return jsonResponse({
          run_id: "exec_active",
          max_parallelism: 2,
          tasks: [],
          edges: []
        });
      }

      if (url.includes("/v1/runs/exec_active/tasks?") && method === "GET") {
        return jsonResponse({
          items: [],
          next_cursor: "next_cursor_1"
        });
      }

      if (url.endsWith("/v1/runs/exec_active/tasks/task_1") && method === "GET") {
        return jsonResponse({
          task_id: "task_1",
          run_id: "exec_active",
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

      if (url.endsWith("/v1/runs/exec_active/tasks/task_1/control") && method === "POST") {
        return jsonResponse({
          ok: true,
          run_id: "exec_active",
          task_id: "task_1",
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

  it("loads run task graph using active execution as run context", async () => {
    const runtime = ensureConversationRuntime(conversation, true);
    runtime.executions.push(createExecution("exec_seed", "queued", 0), createExecution("exec_active", "executing", 1));

    const graph = await loadConversationRunTaskGraph(conversation.id);

    expect(graph?.run_id).toBe("exec_active");
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/v1/runs/exec_active/graph"),
      expect.objectContaining({ method: "GET" })
    );
  });

  it("loads run task list with forwarded query options", async () => {
    const runtime = ensureConversationRuntime(conversation, true);
    runtime.executions.push(createExecution("exec_seed", "queued", 0), createExecution("exec_active", "executing", 1));

    const response = await loadConversationRunTasks(conversation.id, { state: "queued", limit: 10 });

    expect(response?.next_cursor).toBe("next_cursor_1");
    const call = fetchMock.mock.calls.find(([url]) => String(url).includes("/v1/runs/exec_active/tasks?"));
    expect(call).toBeDefined();
    const requestURL = String(call?.[0] ?? "");
    expect(requestURL).toContain("state=queued");
    expect(requestURL).toContain("limit=10");
  });

  it("loads run task detail and controls task", async () => {
    const runtime = ensureConversationRuntime(conversation, true);
    runtime.executions.push(createExecution("exec_seed", "queued", 0), createExecution("exec_active", "executing", 1));

    const task = await loadConversationRunTaskById(conversation.id, "task_1");
    const control = await controlConversationRunTask(conversation, "task_1", "cancel", "user_requested");

    expect(task?.task_id).toBe("task_1");
    expect(control?.ok).toBe(true);
    const controlCall = fetchMock.mock.calls.find(([url, init]) => {
      return String(url).endsWith("/v1/runs/exec_active/tasks/task_1/control") && (init?.method ?? "GET") === "POST";
    });
    expect(controlCall).toBeDefined();
    const payload = JSON.parse(String(controlCall?.[1]?.body ?? "{}")) as { action?: string; reason?: string };
    expect(payload).toEqual({ action: "cancel", reason: "user_requested" });
  });

  it("returns null when runtime is missing", async () => {
    const graph = await loadConversationRunTaskGraph("missing");
    const tasks = await loadConversationRunTasks("missing");
    const task = await loadConversationRunTaskById("missing", "task_1");
    const control = await controlConversationRunTask(conversation, "task_1", "cancel");

    expect(graph).toBeNull();
    expect(tasks).toBeNull();
    expect(task).toBeNull();
    expect(control).toBeNull();
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

function createExecution(id: string, state: "queued" | "executing", queueIndex: number) {
  return {
    id,
    workspace_id: "ws_local",
    conversation_id: conversation.id,
    message_id: `msg_${id}`,
    state,
    mode: "default" as const,
    model_id: "gpt-5.3",
    mode_snapshot: "default" as const,
    model_snapshot: {
      model_id: "gpt-5.3"
    },
    project_revision_snapshot: 0,
    queue_index: queueIndex,
    trace_id: `tr_${id}`,
    created_at: "2026-03-02T00:00:00Z",
    updated_at: "2026-03-02T00:00:00Z"
  };
}

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: {
      "Content-Type": "application/json"
    }
  });
}
