import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { confirmToolCall, createRun } from "@/api/runtimeClient";
import { ContextPanel } from "@/components/domain/context/ContextPanel";
import { DiffPanel } from "@/components/domain/diff/DiffPanel";
import { CapabilityPromptDialog } from "@/components/domain/permission/CapabilityPromptDialog";
import { PermissionQueueCenter } from "@/components/domain/permission/PermissionQueueCenter";
import { RunComposerPanel } from "@/components/domain/run/RunComposerPanel";
import { TimelinePanel } from "@/components/domain/timeline/TimelinePanel";
import { ToolDetailsDrawer } from "@/components/domain/tools/ToolDetailsDrawer";
import { RemotePlaceholder } from "@/components/domain/workspace/RemotePlaceholder";
import { useToast } from "@/components/ui/toast";
import { useRunEvents } from "@/hooks/useRunEvents";
import { isEditableElement } from "@/lib/shortcuts";
import { usePermissionStore } from "@/stores/permissionStore";
import { useRunStore } from "@/stores/runStore";
import { selectCurrentWorkspaceKind, useWorkspaceStore } from "@/stores/workspaceStore";

export function RunPage() {
  const { t } = useTranslation();
  const workspaceKind = useWorkspaceStore(selectCurrentWorkspaceKind);

  const [values, setValues] = useState({
    projectId: "project-demo",
    sessionId: "session-demo",
    modelConfigId: "model-demo",
    workspacePath: "/Users/goya/Repo/Git/Goyais",
    taskInput: t("run.defaultTask")
  });
  const [selectedEventId, setSelectedEventId] = useState<string>();
  const [selectedPermissionCallId, setSelectedPermissionCallId] = useState<string>();
  const [isStarting, setIsStarting] = useState(false);

  const runId = useRunStore((state) => state.runId);
  const setRunId = useRunStore((state) => state.setRunId);
  const setContext = useRunStore((state) => state.setContext);
  const resetRun = useRunStore((state) => state.reset);
  const events = useRunStore((state) => state.events);
  const pendingConfirmations = useRunStore((state) => state.pendingConfirmations);
  const lastPatch = useRunStore((state) => state.lastPatch);
  const toolCalls = useRunStore((state) => state.toolCalls);
  const selectedToolCallId = useRunStore((state) => state.selectedToolCallId);
  const setSelectedToolCallId = useRunStore((state) => state.setSelectedToolCallId);
  const resolvePendingConfirmation = useRunStore((state) => state.resolvePendingConfirmation);

  const addDecision = usePermissionStore((state) => state.addDecision);
  const { addToast } = useToast();

  useRunEvents(runId);

  const activeConfirmation = useMemo(() => {
    if (selectedPermissionCallId) {
      return pendingConfirmations.find((item) => item.callId === selectedPermissionCallId) ?? pendingConfirmations[0];
    }
    return pendingConfirmations[0];
  }, [pendingConfirmations, selectedPermissionCallId]);

  const onStart = async (event: FormEvent) => {
    event.preventDefault();
    setIsStarting(true);
    try {
      resetRun();
      setContext({
        projectId: values.projectId,
        modelConfigId: values.modelConfigId,
        workspacePath: values.workspacePath,
        sessionId: values.sessionId
      });
      const result = await createRun({
        project_id: values.projectId,
        session_id: values.sessionId,
        input: values.taskInput,
        model_config_id: values.modelConfigId,
        workspace_path: values.workspacePath,
        options: { use_worktree: false }
      });
      setRunId(result.run_id);
      addToast({
        title: t("run.startSuccess"),
        description: result.run_id,
        variant: "info"
      });
    } catch (error) {
      addToast({
        title: t("run.startFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    } finally {
      setIsStarting(false);
    }
  };

  const onDecision = useCallback(
    async (mode: "once" | "always" | "deny") => {
      if (!activeConfirmation) return;
      const approved = mode !== "deny";
      try {
        await confirmToolCall(activeConfirmation.runId, activeConfirmation.callId, approved);
        addDecision({
          runId: activeConfirmation.runId,
          callId: activeConfirmation.callId,
          approved,
          mode,
          decidedAt: new Date().toISOString()
        });
        resolvePendingConfirmation(activeConfirmation.callId, approved);
        setSelectedPermissionCallId(undefined);
        addToast({
          title: approved ? t("run.permissionApproved") : t("run.permissionDenied"),
          description: `${activeConfirmation.toolName} · ${t(`run.mode.${mode}`)}`,
          variant: approved ? "success" : "warning"
        });
      } catch (error) {
        addToast({
          title: t("run.permissionActionFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
      }
    },
    [activeConfirmation, addDecision, resolvePendingConfirmation, addToast, t]
  );

  useEffect(() => {
    if (!activeConfirmation) return;

    const onKeydown = (event: KeyboardEvent) => {
      if (isEditableElement(event.target)) return;
      const key = event.key.toLowerCase();
      if (key === "y") {
        event.preventDefault();
        void onDecision("once");
      }
      if (key === "a") {
        event.preventDefault();
        void onDecision("always");
      }
      if (key === "n") {
        event.preventDefault();
        void onDecision("deny");
      }
    };

    window.addEventListener("keydown", onKeydown);
    return () => window.removeEventListener("keydown", onKeydown);
  }, [activeConfirmation, onDecision]);

  if (workspaceKind === "remote") {
    return <RemotePlaceholder section="run" />;
  }

  return (
    <div className="grid h-full min-h-[calc(100vh-8rem)] grid-cols-[18rem_minmax(0,1fr)_22rem] gap-panel">
      <RunComposerPanel
        values={values}
        running={isStarting}
        onSubmit={onStart}
        onChange={(next) => setValues((state) => ({ ...state, ...next }))}
      />

      <div className="grid min-h-0 grid-rows-[minmax(0,1fr)_22rem] gap-panel">
        <TimelinePanel
          events={events}
          selectedEventId={selectedEventId}
          onSelectEvent={setSelectedEventId}
          onSelectToolCall={setSelectedToolCallId}
        />
        <DiffPanel unifiedDiff={lastPatch} />
      </div>

      <section className="min-h-0 space-y-panel overflow-auto pr-1 scrollbar-subtle">
        <ContextPanel workspacePath={values.workspacePath} taskInput={values.taskInput} eventsCount={events.length} />
        <ToolDetailsDrawer toolCalls={toolCalls} selectedCallId={selectedToolCallId} onSelectCallId={setSelectedToolCallId} />
        <PermissionQueueCenter queue={pendingConfirmations} onOpen={(callId) => setSelectedPermissionCallId(callId)} />
      </section>

      <CapabilityPromptDialog
        item={activeConfirmation}
        open={Boolean(activeConfirmation)}
        onClose={() => setSelectedPermissionCallId(undefined)}
        onDecision={(mode) => void onDecision(mode)}
      />
    </div>
  );
}
