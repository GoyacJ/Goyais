import { FormEvent, useMemo, useState } from "react";

import { confirmToolCall, createRun } from "../api/runtimeClient";
import { DiffPanel } from "../components/DiffPanel";
import { EventTimeline } from "../components/EventTimeline";
import { PermissionCenter } from "../components/PermissionCenter";
import { PermissionModal } from "../components/PermissionModal";
import { useRunEvents } from "../hooks/useRunEvents";
import { usePermissionStore } from "../stores/permissionStore";
import { useRunStore } from "../stores/runStore";

export function RunPage() {
  const [projectId, setProjectId] = useState("project-demo");
  const [sessionId, setSessionId] = useState("session-demo");
  const [modelConfigId, setModelConfigId] = useState("model-demo");
  const [workspacePath, setWorkspacePath] = useState("/Users/goya/Repo/Git/Goyais");
  const [taskInput, setTaskInput] = useState("把 README 的标题改成 MVP-1 Demo");

  const runId = useRunStore((state) => state.runId);
  const setRunId = useRunStore((state) => state.setRunId);
  const resetRun = useRunStore((state) => state.reset);
  const events = useRunStore((state) => state.events);
  const pendingConfirmations = useRunStore((state) => state.pendingConfirmations);
  const lastPatch = useRunStore((state) => state.lastPatch);

  const addDecision = usePermissionStore((state) => state.addDecision);

  useRunEvents(runId);

  const activeConfirmation = useMemo(() => pendingConfirmations[0], [pendingConfirmations]);

  const onStart = async (event: FormEvent) => {
    event.preventDefault();
    resetRun();
    const result = await createRun({
      project_id: projectId,
      session_id: sessionId,
      input: taskInput,
      model_config_id: modelConfigId,
      workspace_path: workspacePath,
      options: { use_worktree: false }
    });
    setRunId(result.run_id);
  };

  const onDecision = async (approved: boolean) => {
    if (!activeConfirmation) return;
    await confirmToolCall(activeConfirmation.runId, activeConfirmation.callId, approved);
    addDecision({
      runId: activeConfirmation.runId,
      callId: activeConfirmation.callId,
      approved,
      decidedAt: new Date().toISOString()
    });
    useRunStore.setState((state) => ({ pendingConfirmations: state.pendingConfirmations.slice(1) }));
  };

  return (
    <div className="grid two-columns">
      <section className="panel">
        <h2>Run</h2>
        <form onSubmit={onStart} className="form-grid">
          <label>
            Project ID
            <input value={projectId} onChange={(e) => setProjectId(e.target.value)} />
          </label>
          <label>
            Session ID
            <input value={sessionId} onChange={(e) => setSessionId(e.target.value)} />
          </label>
          <label>
            Model Config ID
            <input value={modelConfigId} onChange={(e) => setModelConfigId(e.target.value)} />
          </label>
          <label>
            Workspace Path
            <input value={workspacePath} onChange={(e) => setWorkspacePath(e.target.value)} />
          </label>
          <label>
            Task
            <textarea value={taskInput} onChange={(e) => setTaskInput(e.target.value)} rows={4} />
          </label>
          <button type="submit">Start run</button>
        </form>
      </section>

      <PermissionCenter />
      <EventTimeline events={events} />
      <DiffPanel unifiedDiff={lastPatch} />

      {activeConfirmation && (
        <PermissionModal
          toolName={activeConfirmation.toolName}
          args={activeConfirmation.args}
          onApprove={() => void onDecision(true)}
          onDeny={() => void onDecision(false)}
        />
      )}
    </div>
  );
}
