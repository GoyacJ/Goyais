import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { Conversation } from "@/shared/types/api";

const streamConversationEventsMock = vi.fn();
const applyIncomingExecutionEventMock = vi.fn();

vi.mock("@/modules/conversation/services", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/modules/conversation/services")>();
  return {
    ...actual,
    streamConversationEvents: (...args: unknown[]) => streamConversationEventsMock(...args)
  };
});

vi.mock("@/modules/conversation/store/executionActions", () => ({
  applyIncomingExecutionEvent: (...args: unknown[]) => applyIncomingExecutionEventMock(...args)
}));

import {
  attachConversationStream,
  conversationStore,
  detachConversationStream,
  ensureConversationRuntime,
  resetConversationStore
} from "@/modules/conversation/store";

const conversationA: Conversation = {
  id: "conv_stream_a",
  workspace_id: "ws_local",
  project_id: "proj_stream",
  name: "A",
  queue_state: "idle",
  default_mode: "agent",
  model_id: "gpt-5.3",
  base_revision: 0,
  active_execution_id: null,
  created_at: "2026-02-24T00:00:00Z",
  updated_at: "2026-02-24T00:00:00Z"
};

const conversationB: Conversation = {
  ...conversationA,
  id: "conv_stream_b",
  name: "B"
};

describe("conversation stream routing", () => {
  let onEvent: ((event: unknown) => void) | undefined;
  let closeHandle: ReturnType<typeof vi.fn>;
  let optionsUsed:
    | {
      initialLastEventId?: string;
      onEvent: (event: unknown) => void;
    }
    | undefined;

  beforeEach(() => {
    resetConversationStore();
    vi.stubGlobal("EventSource", class MockEventSource {});
    onEvent = undefined;
    optionsUsed = undefined;
    closeHandle = vi.fn();
    streamConversationEventsMock.mockReset();
    applyIncomingExecutionEventMock.mockReset();
    streamConversationEventsMock.mockImplementation((
      _conversationId: string,
      options: { initialLastEventId?: string; onEvent: (event: unknown) => void }
    ) => {
      optionsUsed = options;
      onEvent = options.onEvent;
      return {
        close: closeHandle,
        lastEventId: () => "evt_stream_last_handle"
      };
    });
    ensureConversationRuntime(conversationA, true);
    ensureConversationRuntime(conversationB, true);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    resetConversationStore();
  });

  it("routes mismatched stream events by event.conversation_id", () => {
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    attachConversationStream(conversationA);
    expect(typeof onEvent).toBe("function");

    onEvent?.({
      event_id: "evt_stream_1",
      execution_id: "exec_stream_1",
      conversation_id: conversationB.id,
      trace_id: "tr_stream_1",
      sequence: 1,
      queue_index: 0,
      type: "execution_started",
      timestamp: "2026-02-24T00:00:00Z",
      payload: {}
    });

    expect(applyIncomingExecutionEventMock).toHaveBeenCalledTimes(1);
    expect(applyIncomingExecutionEventMock).toHaveBeenCalledWith(
      conversationB.id,
      expect.objectContaining({ conversation_id: conversationB.id, execution_id: "exec_stream_1" })
    );
    expect(warnSpy).toHaveBeenCalledTimes(1);
    warnSpy.mockRestore();
  });

  it("passes lastEventId during attach and updates runtime lastEventId from incoming event", () => {
    const runtime = ensureConversationRuntime(conversationA, true);
    runtime.lastEventId = "evt_stream_resume_from";

    attachConversationStream(conversationA);

    expect(optionsUsed?.initialLastEventId).toBe("evt_stream_resume_from");
    onEvent?.({
      event_id: "evt_stream_new",
      execution_id: "exec_stream_2",
      conversation_id: conversationA.id,
      trace_id: "tr_stream_2",
      sequence: 2,
      queue_index: 0,
      type: "thinking_delta",
      timestamp: "2026-02-24T00:00:00Z",
      payload: {
        stage: "model_call"
      }
    });
    expect(runtime.lastEventId).toBe("evt_stream_new");
  });

  it("detaches stream handle", () => {
    attachConversationStream(conversationA);
    expect(conversationStore.streams[conversationA.id]).toBeTruthy();

    detachConversationStream(conversationA.id);
    expect(closeHandle).toHaveBeenCalledTimes(1);
    expect(conversationStore.streams[conversationA.id]).toBeUndefined();
    expect(conversationStore.byConversationId[conversationA.id]?.lastEventId).toBe("evt_stream_last_handle");
  });
});
