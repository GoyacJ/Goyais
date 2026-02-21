import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { type DataProject,getModelConfigsClient, getProjectsClient } from "@/api/dataSource";
import { getRunDataSource } from "@/api/runDataSource";
import { ContextPanel } from "@/components/domain/context/ContextPanel";
import { DiffPanel } from "@/components/domain/diff/DiffPanel";
import { EmptyState } from "@/components/domain/feedback/EmptyState";
import { CapabilityPromptDialog } from "@/components/domain/permission/CapabilityPromptDialog";
import { PermissionQueueCenter } from "@/components/domain/permission/PermissionQueueCenter";
import { RunComposerPanel } from "@/components/domain/run/RunComposerPanel";
import { TimelinePanel } from "@/components/domain/timeline/TimelinePanel";
import { ToolDetailsDrawer } from "@/components/domain/tools/ToolDetailsDrawer";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import { useReplay } from "@/hooks/useReplay";
import { useRunEvents } from "@/hooks/useRunEvents";
import {
  type ConversationRunSummary,
  useConversationStore
} from "@/stores/conversationStore";
import { usePermissionStore } from "@/stores/permissionStore";
import { useRunStore } from "@/stores/runStore";
import { useSettingsStore } from "@/stores/settingsStore";
import { selectCurrentProfile, useWorkspaceStore } from "@/stores/workspaceStore";
import type { EventEnvelope } from "@/types/generated";
import type { ToolCallView } from "@/types/ui";

function deriveToolCalls(events: EventEnvelope[]): Record<string, ToolCallView> {
  const toolCalls: Record<string, ToolCallView> = {};

  for (const event of events) {
    if (event.type === "tool_call") {
      const callId = String(event.payload.call_id ?? "");
      if (!callId) continue;
      toolCalls[callId] = {
        callId,
        toolName: String(event.payload.tool_name ?? "unknown"),
        args: (event.payload.args ?? {}) as Record<string, unknown>,
        requiresConfirmation: Boolean(event.payload.requires_confirmation),
        status: event.payload.requires_confirmation ? "waiting" : "completed",
        createdAt: event.ts
      };
    }

    if (event.type === "tool_result") {
      const callId = String(event.payload.call_id ?? "");
      const current = toolCalls[callId];
      if (!current) continue;
      const output = event.payload.ok === true ? event.payload.output : event.payload.error;
      toolCalls[callId] = {
        ...current,
        output,
        finishedAt: event.ts,
        status: event.payload.ok === true ? "completed" : "failed"
      };
    }
  }

  return toolCalls;
}

function extractLastPatch(events: EventEnvelope[]): string | undefined {
  for (let i = events.length - 1; i >= 0; i -= 1) {
    const event = events[i];
    if (event.type === "patch") {
      return String(event.payload.unified_diff ?? "");
    }
  }
  return undefined;
}

function toRunSummary(items: Array<Record<string, string>>): ConversationRunSummary[] {
  return items.map((item) => ({
    run_id: String(item.run_id ?? ""),
    trace_id: item.trace_id,
    status: String(item.status ?? "unknown"),
    created_at: String(item.created_at ?? ""),
    input: item.input
  }));
}

export function ConversationPage() {
  const { t } = useTranslation();
  const { addToast } = useToast();
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const runDataSource = useMemo(() => getRunDataSource(currentProfile), [currentProfile]);
  const projectsClient = useMemo(() => getProjectsClient(currentProfile), [currentProfile]);
  const modelConfigsClient = useMemo(() => getModelConfigsClient(currentProfile), [currentProfile]);

  const selectedProjectId = useConversationStore((state) => state.selectedProjectId);
  const selectedSessionId = useConversationStore((state) => state.selectedSessionId);
  const sessionsByProjectId = useConversationStore((state) => state.sessionsByProjectId);
  const detailBySessionId = useConversationStore((state) => state.detailBySessionId);
  const setSelectedProject = useConversationStore((state) => state.setSelectedProject);
  const setSelectedSession = useConversationStore((state) => state.setSelectedSession);
  const setSessions = useConversationStore((state) => state.setSessions);
  const setSelectedRunId = useConversationStore((state) => state.setSelectedRunId);
  const upsertSession = useConversationStore((state) => state.upsertSession);
  const touchSessionRun = useConversationStore((state) => state.touchSessionRun);

  const defaultModelConfigId = useSettingsStore((state) => state.defaultModelConfigId);
  const setDefaultModelConfigId = useSettingsStore((state) => state.setDefaultModelConfigId);

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

  const [projects, setProjects] = useState<DataProject[]>([]);
  const [runs, setRuns] = useState<ConversationRunSummary[]>([]);
  const [modelOptions, setModelOptions] = useState<Array<{ model_config_id: string; provider: string; model: string }>>([]);
  const [input, setInput] = useState("");
  const [modelConfigId, setModelConfigId] = useState<string>();
  const [selectedEventId, setSelectedEventId] = useState<string>();
  const [selectedPermissionCallId, setSelectedPermissionCallId] = useState<string>();
  const [isStarting, setIsStarting] = useState(false);
  const [titleEditing, setTitleEditing] = useState(false);
  const [titleDraft, setTitleDraft] = useState("");
  const [replaySelectedToolCallId, setReplaySelectedToolCallId] = useState<string>();

  const selectedSession = useMemo(
    () => sessionsByProjectId[selectedProjectId ?? ""]?.find((item) => item.session_id === selectedSessionId),
    [selectedProjectId, selectedSessionId, sessionsByProjectId]
  );

  useEffect(() => {
    setTitleDraft(selectedSession?.title ?? "");
  }, [selectedSession?.title]);

  const selectedProject = useMemo(
    () => projects.find((item) => item.project_id === selectedProjectId),
    [projects, selectedProjectId]
  );

  const selectedRunId = selectedSessionId ? detailBySessionId[selectedSessionId]?.selectedRunId : undefined;
  const replayRunId = selectedRunId && selectedRunId !== runId ? selectedRunId : undefined;
  const { events: replayEvents, loading: replayLoading } = useReplay(replayRunId);

  const timelineEvents = replayRunId ? replayEvents : events;
  const viewToolCalls = useMemo(
    () => (replayRunId ? deriveToolCalls(timelineEvents) : toolCalls),
    [replayRunId, timelineEvents, toolCalls]
  );
  const viewLastPatch = replayRunId ? extractLastPatch(timelineEvents) : lastPatch;
  const viewSelectedToolCallId = replayRunId ? replaySelectedToolCallId : selectedToolCallId;

  const activeConfirmation = useMemo(() => {
    if (replayRunId) return undefined;
    if (selectedPermissionCallId) {
      return pendingConfirmations.find((item) => item.callId === selectedPermissionCallId) ?? pendingConfirmations[0];
    }
    return pendingConfirmations[0];
  }, [pendingConfirmations, replayRunId, selectedPermissionCallId]);

  useRunEvents(runId);

  const refreshProjects = useCallback(async () => {
    try {
      const list = await projectsClient.list();
      setProjects(list);
      if (!selectedProjectId || !list.some((item) => item.project_id === selectedProjectId)) {
        setSelectedProject(list[0]?.project_id);
      }
    } catch (error) {
      addToast({
        title: t("projects.loadFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  }, [addToast, projectsClient, selectedProjectId, setSelectedProject, t]);

  const refreshRuns = useCallback(
    async (sessionId: string) => {
      try {
        const payload = await runDataSource.listRuns(sessionId);
        const mapped = toRunSummary(payload.runs).filter((item) => item.run_id);
        setRuns(mapped);
        const preferred = detailBySessionId[sessionId]?.selectedRunId;
        const nextRun = preferred && mapped.some((item) => item.run_id === preferred) ? preferred : mapped[0]?.run_id;
        if (nextRun) {
          setSelectedRunId(sessionId, nextRun);
        }
      } catch (error) {
        addToast({
          title: t("replay.loadFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
      }
    },
    [addToast, detailBySessionId, runDataSource, setSelectedRunId, t]
  );

  useEffect(() => {
    void refreshProjects();
  }, [refreshProjects]);

  useEffect(() => {
    modelConfigsClient
      .list()
      .then((items) => {
        const options = items.map((item) => ({
          model_config_id: item.model_config_id,
          provider: item.provider,
          model: item.model
        }));
        setModelOptions(options);

        const preferred = defaultModelConfigId && options.some((item) => item.model_config_id === defaultModelConfigId)
          ? defaultModelConfigId
          : options[0]?.model_config_id;

        if (preferred) {
          setModelConfigId(preferred);
          setDefaultModelConfigId(preferred);
        }
      })
      .catch(() => {
        setModelOptions([]);
      });
  }, [defaultModelConfigId, modelConfigsClient, setDefaultModelConfigId]);

  useEffect(() => {
    if (!selectedProjectId) return;

    runDataSource
      .listSessions(selectedProjectId)
      .then((payload) => {
        setSessions(selectedProjectId, payload.sessions);
        if (!selectedSessionId && payload.sessions[0]) {
          setSelectedSession(selectedProjectId, payload.sessions[0].session_id);
        }
      })
      .catch(() => {
        // Keep local fallback from persisted conversation store.
      });
  }, [runDataSource, selectedProjectId, selectedSessionId, setSelectedSession, setSessions]);

  useEffect(() => {
    if (!selectedSessionId) {
      setRuns([]);
      return;
    }
    void refreshRuns(selectedSessionId);
  }, [refreshRuns, selectedSessionId]);

  useEffect(() => {
    if (!runId || !selectedSessionId) return;
    void refreshRuns(selectedSessionId);
  }, [refreshRuns, runId, selectedSessionId]);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();

    if (!selectedProjectId || !input.trim()) {
      return;
    }

    let sessionId = selectedSessionId;
    if (!sessionId) {
      try {
        const created = await runDataSource.createSession({
          project_id: selectedProjectId,
          title: t("conversation.newThread")
        });
        upsertSession(created.session);
        sessionId = created.session.session_id;
        setSelectedSession(selectedProjectId, sessionId);
      } catch (error) {
        addToast({
          title: t("conversation.startFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
        return;
      }
    }

    const activeModelId = modelConfigId ?? modelOptions[0]?.model_config_id;
    if (!activeModelId) {
      addToast({
        title: t("conversation.modelRequired"),
        variant: "warning"
      });
      return;
    }

    setIsStarting(true);
    try {
      const workspacePath = selectedProject?.workspace_path ?? selectedProject?.root_uri ?? "/Users/goya/Repo/Git/Goyais";

      resetRun();
      setContext({
        projectId: selectedProjectId,
        modelConfigId: activeModelId,
        workspacePath,
        sessionId
      });

      const result = await runDataSource.createRun({
        project_id: selectedProjectId,
        session_id: sessionId,
        input: input.trim(),
        model_config_id: activeModelId,
        workspace_path: workspacePath,
        options: { use_worktree: false }
      });

      setRunId(result.run_id);
      setSelectedRunId(sessionId, result.run_id);
      touchSessionRun(sessionId, {
        last_run_id: result.run_id,
        last_status: "running",
        last_input_preview: input.trim().slice(0, 160),
        updated_at: new Date().toISOString()
      });

      if (selectedSession?.title === t("conversation.newThread")) {
        const nextTitle = input.trim().slice(0, 40);
        const renamed = await runDataSource.renameSession(sessionId, nextTitle);
        upsertSession(renamed.session);
      }

      setInput("");
      addToast({
        title: t("conversation.startSuccess"),
        description: result.run_id,
        variant: "info"
      });
    } catch (error) {
      addToast({
        title: t("conversation.startFailed"),
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
        await runDataSource.confirmToolCall(activeConfirmation.runId, activeConfirmation.callId, approved);
        addDecision({
          runId: activeConfirmation.runId,
          callId: activeConfirmation.callId,
          approved,
          mode,
          decidedAt: new Date().toISOString()
        });
        resolvePendingConfirmation(activeConfirmation.callId, approved);
        setSelectedPermissionCallId(undefined);
      } catch (error) {
        addToast({
          title: t("run.permissionActionFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
      }
    },
    [activeConfirmation, addDecision, addToast, resolvePendingConfirmation, runDataSource, t]
  );

  const onRenameSession = async () => {
    if (!selectedSessionId || !titleDraft.trim()) return;
    try {
      const payload = await runDataSource.renameSession(selectedSessionId, titleDraft.trim());
      upsertSession(payload.session);
      setTitleEditing(false);
    } catch (error) {
      addToast({
        title: t("conversation.renameFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    }
  };

  if (!selectedProjectId) {
    return <EmptyState title={t("conversation.noProjectTitle")} description={t("conversation.noProjectDescription")} />;
  }

  const sessionTitle = selectedSession?.title ?? t("conversation.newThread");

  return (
    <div className="grid h-full min-h-[calc(100vh-8rem)] grid-cols-[minmax(0,1fr)_22rem] gap-panel">
      <section className="flex min-h-0 flex-col gap-panel">
        <Card>
          <CardContent className="flex items-center justify-between gap-3 p-3">
            {titleEditing ? (
              <Input value={titleDraft} onChange={(event) => setTitleDraft(event.target.value)} />
            ) : (
              <h1 className="truncate text-h2 text-foreground">{sessionTitle}</h1>
            )}
            <div className="flex items-center gap-2">
              {titleEditing ? (
                <>
                  <button
                    type="button"
                    className="rounded-control border border-border-subtle px-2 py-1 text-small text-muted-foreground hover:bg-muted"
                    onClick={() => setTitleEditing(false)}
                  >
                    {t("workspace.cancel")}
                  </button>
                  <button
                    type="button"
                    className="rounded-control border border-border-subtle px-2 py-1 text-small text-foreground hover:bg-muted"
                    onClick={() => {
                      void onRenameSession();
                    }}
                  >
                    {t("conversation.rename")}
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  className="rounded-control border border-border-subtle px-2 py-1 text-small text-muted-foreground hover:bg-muted"
                  onClick={() => setTitleEditing(true)}
                >
                  {t("conversation.rename")}
                </button>
              )}
            </div>
          </CardContent>
        </Card>

        <div className="grid min-h-0 flex-1 grid-rows-[minmax(0,1fr)_auto] gap-panel">
          <Card className="min-h-0">
            <CardHeader className="pb-2">
              <CardTitle>{replayLoading ? t("replay.loading") : t("timeline.title")}</CardTitle>
            </CardHeader>
            <CardContent className="h-[calc(100%-4.5rem)] min-h-0">
              <TimelinePanel
                events={timelineEvents}
                selectedEventId={selectedEventId}
                onSelectEvent={setSelectedEventId}
                onSelectToolCall={(callId) => {
                  if (replayRunId) {
                    setReplaySelectedToolCallId(callId);
                    return;
                  }
                  setSelectedToolCallId(callId);
                }}
              />
            </CardContent>
          </Card>

          <RunComposerPanel
            input={input}
            running={isStarting}
            modelConfigId={modelConfigId}
            modelOptions={modelOptions}
            onInputChange={setInput}
            onModelConfigIdChange={(value) => {
              setModelConfigId(value);
              setDefaultModelConfigId(value);
            }}
            onSubmit={onSubmit}
          />
        </div>
      </section>

      <section className="min-h-0 space-y-panel overflow-auto pr-1 scrollbar-subtle">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle>{t("replay.title")}</CardTitle>
          </CardHeader>
          <CardContent>
            <select
              className="h-9 w-full rounded-control border border-border bg-background px-2 text-small text-foreground"
              value={selectedRunId ?? ""}
              onChange={(event) => {
                if (!selectedSessionId) return;
                setSelectedRunId(selectedSessionId, event.target.value || undefined);
              }}
            >
              <option value="">{t("replay.selectRun")}</option>
              {runs.map((run) => (
                <option key={run.run_id} value={run.run_id}>
                  {run.run_id} · {run.status}
                </option>
              ))}
            </select>
          </CardContent>
        </Card>

        <DiffPanel unifiedDiff={viewLastPatch} />

        <ToolDetailsDrawer
          toolCalls={viewToolCalls}
          selectedCallId={viewSelectedToolCallId}
          onSelectCallId={(callId) => {
            if (replayRunId) {
              setReplaySelectedToolCallId(callId);
              return;
            }
            setSelectedToolCallId(callId);
          }}
        />

        <ContextPanel
          workspacePath={selectedProject?.workspace_path ?? selectedProject?.root_uri ?? "/Users/goya/Repo/Git/Goyais"}
          taskInput={input}
          eventsCount={timelineEvents.length}
        />

        {!replayRunId ? (
          <PermissionQueueCenter queue={pendingConfirmations} onOpen={(callId) => setSelectedPermissionCallId(callId)} />
        ) : null}
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
