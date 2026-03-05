import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";

import MainInspectorPanel from "@/modules/session/components/MainInspectorPanel.vue";

describe("main inspector panel run tasks", () => {
  it("renders run task graph summary and emits refresh event", async () => {
    const wrapper = mount(MainInspectorPanel, {
      props: {
        capability: {
          can_commit: false,
          can_discard: false,
          can_export: true
        },
        queuedCount: 1,
        pendingCount: 0,
        executingCount: 1,
        modelLabel: "gpt-5.3",
        executions: [],
        events: [],
        activeTab: "run",
        runTaskStateFilter: "",
        runTaskListNextCursor: "cursor_1",
        runTaskItems: [
          {
            task_id: "task_1",
            run_id: "run_1",
            title: "Execution task_1",
            state: "running",
            depends_on: [],
            children: [],
            retry_count: 0,
            max_retries: 0,
            created_at: "2026-03-02T00:00:00Z",
            updated_at: "2026-03-02T00:00:00Z"
          },
          {
            task_id: "task_2",
            run_id: "run_1",
            title: "Execution task_2",
            state: "failed",
            depends_on: [],
            children: [],
            retry_count: 0,
            max_retries: 0,
            created_at: "2026-03-02T00:00:00Z",
            updated_at: "2026-03-02T00:00:00Z"
          }
        ],
        runTaskGraph: {
          run_id: "run_1",
          max_parallelism: 2,
          tasks: [
            {
              task_id: "task_1",
              run_id: "run_1",
              title: "Execution task_1",
              state: "running",
              depends_on: [],
              children: [],
              retry_count: 0,
              max_retries: 0,
              created_at: "2026-03-02T00:00:00Z",
              updated_at: "2026-03-02T00:00:00Z"
            },
            {
              task_id: "task_2",
              run_id: "run_1",
              title: "Execution task_2",
              state: "failed",
              depends_on: [],
              children: [],
              retry_count: 0,
              max_retries: 0,
              created_at: "2026-03-02T00:00:00Z",
              updated_at: "2026-03-02T00:00:00Z"
            }
          ],
          edges: []
        },
        selectedRunTask: {
          task_id: "task_2",
          run_id: "run_1",
          title: "Execution task_2",
          state: "failed",
          depends_on: ["task_1"],
          children: [],
          retry_count: 1,
          max_retries: 3,
          last_error: "tool timeout",
          artifact: {
            task_id: "task_2",
            kind: "summary",
            summary: "failed summary"
          },
          created_at: "2026-03-02T00:00:00Z",
          updated_at: "2026-03-02T00:00:00Z"
        }
      }
    });

    expect(wrapper.text()).toContain("Tasks: 2");
    expect(wrapper.text()).toContain("Execution task_1");
    expect(wrapper.text()).toContain("Execution task_2");

    const refreshButton = wrapper.findAll("button").find((item) => item.text().includes("Refresh tasks"));
    const loadMoreButton = wrapper.findAll("button").find((item) => item.text().includes("Load more tasks"));
    expect(refreshButton).toBeDefined();
    expect(loadMoreButton).toBeDefined();
    await refreshButton?.trigger("click");
    await loadMoreButton?.trigger("click");
    expect(wrapper.emitted("refreshRunTasks")).toHaveLength(1);
    expect(wrapper.emitted("loadMoreRunTasks")).toHaveLength(1);

    const filterSelect = wrapper.find("select");
    expect(filterSelect.exists()).toBe(true);
    await filterSelect.setValue("failed");
    expect(wrapper.emitted("changeRunTaskStateFilter")).toEqual([["failed"]]);

    const cancelButton = wrapper.findAll("button").find((item) => item.text().includes("Cancel"));
    const retryButton = wrapper.findAll("button").find((item) => item.text().includes("Retry"));
    const taskSelectButton = wrapper.findAll("button").find((item) => item.text().includes("Execution task_2"));
    expect(cancelButton).toBeDefined();
    expect(retryButton).toBeUndefined();
    expect(taskSelectButton).toBeDefined();

    await taskSelectButton?.trigger("click");

    await cancelButton?.trigger("click");
    expect(wrapper.text()).toContain("Task ID: task_2");
    expect(wrapper.text()).toContain("tool timeout");
    expect(wrapper.text()).toContain("failed summary");

    expect(wrapper.emitted("selectRunTask")).toEqual([["task_2"]]);
    expect(wrapper.emitted("controlRunTask")).toEqual([[{ taskId: "task_1", action: "cancel" }]]);
  });
});
