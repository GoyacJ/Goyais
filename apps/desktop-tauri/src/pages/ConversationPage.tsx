import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { type DataProject, getModelConfigsClient, getProjectsClient } from "@/api/dataSource";
import { getSessionDataSource } from "@/api/sessionDataSource";
import { ContextPanel } from "@/components/domain/context/ContextPanel";
import { ConversationTranscriptPanel } from "@/components/domain/conversation/ConversationTranscriptPanel";
import { ExecutionActions } from "@/components/domain/conversation/ExecutionActions";
import { DiffPanel } from "@/components/domain/diff/DiffPanel";
import { ExecutionComposerPanel } from "@/components/domain/execution/ExecutionComposerPanel";
import { EmptyState } from "@/components/domain/feedback/EmptyState";
import { CapabilityPromptDialog } from "@/components/domain/permission/CapabilityPromptDialog";
import { PermissionQueueCenter } from "@/components/domain/permission/PermissionQueueCenter";
import { TimelinePanel } from "@/components/domain/timeline/TimelinePanel";
import { ToolDetailsDrawer } from "@/components/domain/tools/ToolDetailsDrawer";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/components/ui/toast";
import { useExecutionEvents } from "@/hooks/useExecutionEvents";
import { useConversationStore } from "@/stores/conversationStore";
import { useExecutionStore } from "@/stores/executionStore";
import { usePermissionStore } from "@/stores/permissionStore";
import { useSettingsStore } from "@/stores/settingsStore";
import { selectCurrentProfile, useWorkspaceStore } from "@/stores/workspaceStore";
import type { EventEnvelope } from "@/types/generated";

interface SessionTurn {
  executionId: string;
  status: string;
  createdAt: string;
  userText: string;
}

function summarizeAssistantByEvents(
  events: EventEnvelope[],
  statusFallback: string,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  for (let i = events.length - 1; i >= 0; i -= 1) {
    const event = events[i];
    if (event.type === "error") {
      const error = event.payload.error as Record<string, unknown> | undefined;
      return String(error?.message ?? t("conversation.executionStatus", { status: "failed" }));
    }
    if (event.type === "plan") {
      return String(event.payload.summary ?? t("conversation.planUpdated"));
    }
    if (event.type === "patch") {
      return t("conversation.patchGenerated");
    }
    if (event.type === "done") {
      return t("conversation.executionStatus", { status: String(event.payload.status ?? statusFallback) });
    }
  }
  return t("conversation.executionStatus", { status: statusFallback });
}

export const RIGHT_PANEL_SECTION_ORDER = ["diff", "timeline", "tools", "context"] as const;
export const CONVERSATION_HEADER_FIELDS = ["project", "session", "branch"] as const;

export function ConversationPage() {
  const { t } = useTranslation();
  const { addToast } = useToast();
  const currentProfile = useWorkspaceStore(selectCurrentProfile);
  const sessionDataSource = useMemo(() => getSessionDataSource(currentProfile), [currentProfile]);
  const projectsClient = useMemo(() => getProjectsClient(currentProfile), [currentProfile]);
  const modelConfigsClient = useMemo(() => getModelConfigsClient(currentProfile), [currentProfile]);

  const selectedProjectId = useConversationStore((state) => state.selectedProjectId);
  const selectedSessionId = useConversationStore((state) => state.selectedSessionId);
  const sessionsByProjectId = useConversationStore((state) => state.sessionsByProjectId);
  const setSelectedProject = useConversationStore((state) => state.setSelectedProject);
  const setSelectedSession = useConversationStore((state) => state.setSelectedSession);
  const setSessions = useConversationStore((state) => state.setSessions);
  const upsertSession = useConversationStore((state) => state.upsertSession);
  const touchSessionExecution = useConversationStore((state) => state.touchSessionExecution);

  const defaultModelConfigId = useSettingsStore((state) => state.defaultModelConfigId);
  const setDefaultModelConfigId = useSettingsStore((state) => state.setDefaultModelConfigId);

  const executionId = useExecutionStore((state) => state.executionId);
  const executionStatus = useExecutionStore((state) => state.status);
  const events = useExecutionStore((state) => state.events);
  const pendingConfirmations = useExecutionStore((state) => state.pendingConfirmations);
  const lastPatch = useExecutionStore((state) => state.lastPatch);
  const toolCalls = useExecutionStore((state) => state.toolCalls);
  const selectedToolCallId = useExecutionStore((state) => state.selectedToolCallId);
  const setSelectedToolCallId = useExecutionStore((state) => state.setSelectedToolCallId);
  const resolvePendingConfirmation = useExecutionStore((state) => state.resolvePendingConfirmation);
  const startExecution = useExecutionStore((state) => state.startExecution);
  const resetExecution = useExecutionStore((state) => state.reset);

  const addDecision = usePermissionStore((state) => state.addDecision);

  const [projects, setProjects] = useState<DataProject[]>([]);
  const [modelOptions, setModelOptions] = useState<Array<{ model_config_id: string; provider: string; model: string }>>([]);
  const [input, setInput] = useState("");
  const [modelConfigId, setModelConfigId] = useState<string>();
  const [sessionMode, setSessionMode] = useState<"plan" | "agent">("agent");
  const [sessionUseWorktree, setSessionUseWorktree] = useState(true);
  const [selectedEventId, setSelectedEventId] = useState<string>();
  const [selectedPermissionCallId, setSelectedPermissionCallId] = useState<string>();
  const [isStarting, setIsStarting] = useState(false);
  const [commitSha, setCommitSha] = useState<string>();
  const [actionsBusy, setActionsBusy] = useState(false);
  const [turnsBySession, setTurnsBySession] = useState<Record<string, SessionTurn[]>>({});

  const selectedSession = useMemo(
    () => sessionsByProjectId[selectedProjectId ?? ""]?.find((item) => item.session_id === selectedSessionId),
    [selectedProjectId, selectedSessionId, sessionsByProjectId]
  );

  useEffect(() => {
    setCommitSha(undefined);
  }, [executionId]);

  useEffect(() => {
    if (!selectedSessionId || !executionId) {
      return;
    }

    setTurnsBySession((state) => {
      const current = state[selectedSessionId] ?? [];
      const next = current.map((turn) =>
        turn.executionId === executionId
          ? {
              ...turn,
              status: executionStatus
            }
          : turn
      );
      return { ...state, [selectedSessionId]: next };
    });
  }, [executionId, executionStatus, selectedSessionId]);

  const selectedProject = useMemo(
    () => projects.find((item) => item.project_id === selectedProjectId),
    [projects, selectedProjectId]
  );

  const transcriptTurns = useMemo(() => {
    const sessionTurns = selectedSessionId ? turnsBySession[selectedSessionId] ?? [] : [];
    return [...sessionTurns]
      .reverse()
      .map((turn) => ({
        executionId: turn.executionId,
        status: turn.status,
        createdAt: turn.createdAt,
        userText: turn.userText,
        assistantText:
          turn.executionId === executionId
            ? summarizeAssistantByEvents(events, turn.status, t)
            : t("conversation.executionStatus", { status: turn.status })
      }));
  }, [events, executionId, selectedSessionId, turnsBySession, t]);

  const activeConfirmation = useMemo(() => {
    if (selectedPermissionCallId) {
      return pendingConfirmations.find((item) => item.callId === selectedPermissionCallId) ?? pendingConfirmations[0];
    }
    return pendingConfirmations[0];
  }, [pendingConfirmations, selectedPermissionCallId]);

  useExecutionEvents({
    sessionId: selectedSessionId,
    enabled: Boolean(selectedSessionId && executionId)
  });

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

        const preferred =
          defaultModelConfigId && options.some((item) => item.model_config_id === defaultModelConfigId)
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

    sessionDataSource
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
  }, [selectedProjectId, selectedSessionId, sessionDataSource, setSelectedSession, setSessions]);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();

    if (!selectedProjectId || !input.trim()) {
      return;
    }

    const activeModelId = modelConfigId ?? modelOptions[0]?.model_config_id;
    if (!activeModelId) {
      addToast({
        title: t("conversation.modelRequired"),
        variant: "warning"
      });
      return;
    }

    let sessionId = selectedSessionId;
    if (!sessionId) {
      try {
        const created = await sessionDataSource.createSession({
          project_id: selectedProjectId,
          title: t("conversation.newThread"),
          model_config_id: activeModelId,
          mode: sessionMode,
          use_worktree: sessionUseWorktree
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

    setIsStarting(true);
    try {
      resetExecution();

      const result = await sessionDataSource.executeSession(sessionId, input.trim());
      startExecution(result.execution_id, sessionId, result.trace_id);
      touchSessionExecution(sessionId, {
        last_execution_id: result.execution_id,
        last_status: "executing",
        last_input_preview: input.trim().slice(0, 160),
        updated_at: new Date().toISOString()
      });

      setTurnsBySession((state) => {
        const current = state[sessionId] ?? [];
        return {
          ...state,
          [sessionId]: [
            ...current,
            {
              executionId: result.execution_id,
              status: "executing",
              createdAt: new Date().toISOString(),
              userText: input.trim()
            }
          ]
        };
      });

      if (selectedSession?.title === t("conversation.newThread")) {
        const nextTitle = input.trim().slice(0, 40);
        const renamed = await sessionDataSource.renameSession(sessionId, nextTitle);
        upsertSession(renamed.session);
      }

      setInput("");
      addToast({
        title: t("conversation.startSuccess"),
        description: result.execution_id,
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
      if (!activeConfirmation || !executionId) return;
      const approved = mode !== "deny";
      const decision = approved ? "approved" : "denied";
      try {
        await sessionDataSource.decideConfirmation(executionId, activeConfirmation.callId, decision);
        addDecision({
          executionId,
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
    [activeConfirmation, addDecision, addToast, executionId, resolvePendingConfirmation, sessionDataSource, t]
  );

  const onCommitExecution = useCallback(
    async (message: string) => {
      if (!executionId) return;
      setActionsBusy(true);
      try {
        const payload = await sessionDataSource.commitExecution(executionId, message);
        setCommitSha(payload.commit_sha);
        addToast({
          title: t("conversation.commitSuccess"),
          description: payload.commit_sha,
          variant: "info"
        });
      } catch (error) {
        addToast({
          title: t("conversation.commitFailed"),
          description: (error as Error).message,
          diagnostic: (error as Error).message,
          variant: "error"
        });
      } finally {
        setActionsBusy(false);
      }
    },
    [addToast, executionId, sessionDataSource, t]
  );

  const onExportExecutionPatch = useCallback(async () => {
    if (!executionId) return;
    setActionsBusy(true);
    try {
      const patch = await sessionDataSource.exportExecutionPatch(executionId);
      const blob = new Blob([patch], { type: "text/plain;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `${executionId}.patch`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);

      addToast({
        title: t("conversation.patchExportSuccess"),
        description: `${executionId}.patch`,
        variant: "info"
      });
    } catch (error) {
      addToast({
        title: t("conversation.patchExportFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    } finally {
      setActionsBusy(false);
    }
  }, [addToast, executionId, sessionDataSource, t]);

  const onDiscardExecution = useCallback(async () => {
    if (!executionId) return;
    if (!window.confirm(t("conversation.discardConfirm"))) return;

    setActionsBusy(true);
    try {
      await sessionDataSource.discardExecution(executionId);
      setCommitSha(undefined);
      addToast({
        title: t("conversation.discardSuccess"),
        variant: "info"
      });
    } catch (error) {
      addToast({
        title: t("conversation.discardFailed"),
        description: (error as Error).message,
        diagnostic: (error as Error).message,
        variant: "error"
      });
    } finally {
      setActionsBusy(false);
    }
  }, [addToast, executionId, sessionDataSource, t]);

  if (!selectedProjectId) {
    return <EmptyState title={t("conversation.noProjectTitle")} description={t("conversation.noProjectDescription")} />;
  }

  const showExecutionActions = Boolean(executionId && events.some((event) => event.type === "done"));

  return (
    <div className="grid h-full min-h-0 grid-cols-[minmax(0,1fr)_22rem]">
      <section className="min-h-0 p-3">
        <div className="grid h-full min-h-0 grid-rows-[minmax(0,1fr)_auto_auto] gap-panel">
          <ConversationTranscriptPanel
            title={t("conversation.transcriptTitle")}
            emptyTitle={t("conversation.transcriptEmptyTitle")}
            emptyDescription={t("conversation.transcriptEmptyDescription")}
            turns={transcriptTurns}
          />

          {showExecutionActions && executionId ? (
            <ExecutionActions
              executionId={executionId}
              commitSha={commitSha}
              busy={actionsBusy}
              onCommit={onCommitExecution}
              onExportPatch={onExportExecutionPatch}
              onDiscard={onDiscardExecution}
            />
          ) : null}

          <ExecutionComposerPanel
            input={input}
            modelConfigId={modelConfigId}
            modelOptions={modelOptions}
            running={isStarting}
            sessionMode={sessionMode}
            sessionUseWorktree={sessionUseWorktree}
            onInputChange={setInput}
            onModelChange={(value) => {
              setModelConfigId(value);
              setDefaultModelConfigId(value);
            }}
            onSessionModeChange={setSessionMode}
            onSessionUseWorktreeChange={setSessionUseWorktree}
            onSubmit={onSubmit}
          />
        </div>
      </section>

      <section className="min-h-0 space-y-panel overflow-auto border-l border-border-subtle p-3 scrollbar-subtle">
        <DiffPanel unifiedDiff={lastPatch ?? undefined} />

        <Card className="min-h-0">
          <CardHeader className="pb-2">
            <CardTitle>{t("timeline.title")}</CardTitle>
          </CardHeader>
          <CardContent className="h-[calc(100%-4.5rem)] min-h-0">
            <TimelinePanel
              events={events}
              selectedEventId={selectedEventId}
              onSelectEvent={setSelectedEventId}
              onSelectToolCall={(callId) => {
                setSelectedToolCallId(callId ?? null);
              }}
            />
          </CardContent>
        </Card>

        <ToolDetailsDrawer
          toolCalls={toolCalls}
          selectedCallId={selectedToolCallId ?? undefined}
          onSelectCallId={(callId) => {
            setSelectedToolCallId(callId ?? null);
          }}
        />

        <ContextPanel
          workspacePath={selectedProject?.workspace_path ?? selectedProject?.root_uri ?? "/Users/goya/Repo/Git/Goyais"}
          taskInput={input}
          eventsCount={events.length}
        />

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
