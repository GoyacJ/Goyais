import { beforeEach, describe, expect, it } from "vitest";

import { useConversationStore } from "@/stores/conversationStore";

describe("conversationStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useConversationStore.getState().reset();
  });

  it("keeps sessions sorted by updated_at desc", () => {
    const store = useConversationStore.getState();
    store.setSessions("p1", [
      {
        session_id: "s1",
        project_id: "p1",
        title: "older",
        updated_at: "2026-01-01T00:00:00.000Z"
      },
      {
        session_id: "s2",
        project_id: "p1",
        title: "newer",
        updated_at: "2026-01-02T00:00:00.000Z"
      }
    ]);

    expect(useConversationStore.getState().sessionsByProjectId.p1[0].session_id).toBe("s2");
  });

  it("selects first session when project changes", () => {
    const store = useConversationStore.getState();
    store.setSessions("p1", [
      {
        session_id: "s1",
        project_id: "p1",
        title: "thread",
        updated_at: "2026-01-01T00:00:00.000Z"
      }
    ]);

    store.setSelectedProject("p1");
    expect(useConversationStore.getState().selectedSessionId).toBe("s1");
  });

  it("stores execution metadata by session", () => {
    const store = useConversationStore.getState();
    store.setSessions("p1", [
      {
        session_id: "s1",
        project_id: "p1",
        title: "thread",
        updated_at: "2026-01-01T00:00:00.000Z"
      }
    ]);
    store.touchSessionExecution("s1", { last_execution_id: "exec-1", last_status: "executing" });
    expect(useConversationStore.getState().sessionsByProjectId.p1[0].last_execution_id).toBe("exec-1");
    expect(useConversationStore.getState().sessionsByProjectId.p1[0].last_status).toBe("executing");
  });
});
