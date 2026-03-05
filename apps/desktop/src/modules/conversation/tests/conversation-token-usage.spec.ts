import { describe, expect, it } from "vitest";

import type { SessionRuntime } from "@/modules/conversation/store/state";
import { resolveConversationUsage, summarizeExecutionTokens } from "@/modules/conversation/views/conversationTokenUsage";
import type { Run, Session } from "@/shared/types/api";

describe("conversation token usage helpers", () => {
  it("summarizes token usage from deduped execution list", () => {
    const base = createExecution({
      id: "exec_1",
      tokens_in: 10,
      tokens_out: 5,
      updated_at: "2026-03-01T00:00:00Z"
    });
    const duplicate = createExecution({
      ...base,
      state: "completed",
      tokens_in: 12,
      tokens_out: 8,
      updated_at: "2026-03-01T00:00:10Z"
    });
    const other = createExecution({
      id: "exec_2",
      tokens_in: 3,
      tokens_out: 7,
      updated_at: "2026-03-01T00:00:20Z"
    });

    const usage = summarizeExecutionTokens([base, duplicate, other]);

    expect(usage).toEqual({
      input: 15,
      output: 15,
      total: 30
    });
  });

  it("prefers runtime executions over conversation aggregate fields", () => {
    const conversation = createConversation({
      tokens_in_total: 100,
      tokens_out_total: 120,
      tokens_total: 220
    });
    const runtime = {
      executions: [createExecution({ id: "exec_runtime_1", tokens_in: 2, tokens_out: 4 })]
    } as Pick<SessionRuntime, "executions">;

    const usage = resolveConversationUsage(conversation, runtime);
    expect(usage).toEqual({
      input: 2,
      output: 4,
      total: 6
    });
  });

  it("falls back to conversation aggregate fields and then zero", () => {
    const conversation = createConversation({
      tokens_in_total: 7,
      tokens_out_total: 9,
      tokens_total: 16
    });
    expect(resolveConversationUsage(conversation)).toEqual({
      input: 7,
      output: 9,
      total: 16
    });

    expect(resolveConversationUsage(undefined)).toEqual({
      input: 0,
      output: 0,
      total: 0
    });
  });
});

function createConversation(overrides: Partial<Session> = {}): Session {
  return {
    id: "conv_1",
    workspace_id: "ws_1",
    project_id: "proj_1",
    name: "Conversation",
    queue_state: "idle",
    default_mode: "default",
    model_config_id: "rc_model_1",
    rule_ids: [],
    skill_ids: [],
    mcp_ids: [],
    base_revision: 0,
    active_execution_id: null,
    created_at: "2026-03-01T00:00:00Z",
    updated_at: "2026-03-01T00:00:00Z",
    ...overrides
  };
}

function createExecution(overrides: Partial<Run> = {}): Run {
  return {
    id: "exec_1",
    workspace_id: "ws_1",
    conversation_id: "conv_1",
    message_id: "msg_1",
    state: "executing",
    mode: "default",
    model_id: "gpt-5",
    mode_snapshot: "default",
    model_snapshot: {
      model_id: "gpt-5"
    },
    tokens_in: 0,
    tokens_out: 0,
    project_revision_snapshot: 0,
    queue_index: 1,
    trace_id: "tr_1",
    created_at: "2026-03-01T00:00:00Z",
    updated_at: "2026-03-01T00:00:00Z",
    ...overrides
  };
}
