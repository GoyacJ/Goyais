import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  ensureConversationRuntime,
  resetConversationStore,
  setConversationDraft,
  stopConversationExecution,
  submitConversationMessage
} from "@/modules/conversation/store";
import type { Conversation } from "@/shared/types/api";

const mockConversation: Conversation = {
  id: "conv_test",
  workspace_id: "ws_local",
  project_id: "proj_1",
  name: "Test Conversation",
  queue_state: "idle",
  default_mode: "agent",
  model_id: "gpt-4.1",
  active_execution_id: null,
  created_at: "2026-02-23T00:00:00Z",
  updated_at: "2026-02-23T00:00:00Z"
};

describe("conversation store", () => {
  beforeEach(() => {
    resetConversationStore();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps FIFO queue and drains after completion", async () => {
    ensureConversationRuntime(mockConversation, true);
    setConversationDraft(mockConversation.id, "first message");
    await submitConversationMessage(mockConversation, true);

    setConversationDraft(mockConversation.id, "second message");
    await submitConversationMessage(mockConversation, true);

    const runtime = ensureConversationRuntime(mockConversation, true);
    const queuedBefore = runtime.executions.filter((item) => item.state === "queued").length;
    expect(queuedBefore).toBe(1);

    await vi.advanceTimersByTimeAsync(2300);
    const activeAfterFirst = runtime.executions.filter((item) => item.state === "executing").length;
    expect(activeAfterFirst).toBe(1);

    await vi.advanceTimersByTimeAsync(2300);
    const completedCount = runtime.executions.filter((item) => item.state === "completed").length;
    expect(completedCount).toBe(2);
  });

  it("stop only cancels active execution and keeps queued items", async () => {
    ensureConversationRuntime(mockConversation, true);
    setConversationDraft(mockConversation.id, "active");
    await submitConversationMessage(mockConversation, true);
    setConversationDraft(mockConversation.id, "queued");
    await submitConversationMessage(mockConversation, true);

    await stopConversationExecution(mockConversation);

    const runtime = ensureConversationRuntime(mockConversation, true);
    const cancelledCount = runtime.executions.filter((item) => item.state === "cancelled").length;
    const queuedCount = runtime.executions.filter((item) => item.state === "queued").length;
    const executingCount = runtime.executions.filter((item) => item.state === "executing").length;

    expect(cancelledCount).toBe(1);
    expect(queuedCount).toBe(0);
    expect(executingCount).toBe(1);
  });
});
