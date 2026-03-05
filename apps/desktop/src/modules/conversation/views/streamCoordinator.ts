import { getSessionDetail } from "@/modules/conversation/services";
import {
  attachSessionStream,
  sessionStore,
  detachSessionStream,
  ensureSessionRuntime,
  hasUnfinishedExecutions,
  hydrateSessionRuntime
} from "@/modules/conversation/store";
import type { Project, Session } from "@/shared/types/api";

type SessionContext = {
  session: Session;
  isGitProject: boolean;
};

type StreamCoordinatorInput = {
  projects: () => Project[];
  conversationsByProjectId: () => Record<string, Session[]>;
  activeSessionId: () => string;
  resolveToken: () => string | undefined;
};

export function createSessionStreamCoordinator(input: StreamCoordinatorInput) {
  const hydratingSessionIds = new Set<string>();
  const hydratedSessionIds = new Set<string>();

  function findSessionContextById(sessionId: string): SessionContext | undefined {
    for (const project of input.projects()) {
      const conversations = input.conversationsByProjectId()[project.id] ?? [];
      const session = conversations.find((item) => item.id === sessionId);
      if (session) {
        return { session, isGitProject: project.is_git };
      }
    }
    return undefined;
  }

  async function hydrateSessionDetail(context: SessionContext, force = false): Promise<void> {
    const sessionId = context.session.id;
    if (hydratingSessionIds.has(sessionId)) {
      return;
    }
    if (!force && hydratedSessionIds.has(sessionId)) {
      return;
    }

    const existingRuntime = sessionStore.byConversationId[sessionId];
    const hasHydratedData = Boolean(existingRuntime?.hydrated);
    if (!force && hasHydratedData) {
      return;
    }

    hydratingSessionIds.add(sessionId);
    try {
      const detail = await getSessionDetail(sessionId, { token: input.resolveToken() });
      hydrateSessionRuntime(context.session, context.isGitProject, detail);
      hydratedSessionIds.add(sessionId);
    } catch {
      ensureSessionRuntime(context.session, context.isGitProject);
    } finally {
      hydratingSessionIds.delete(sessionId);
    }
  }

  function collectTrackedSessionContexts(): SessionContext[] {
    const activeSessionId = input.activeSessionId();
    const trackedBySessionId = new Map<string, SessionContext>();

    for (const project of input.projects()) {
      const conversations = input.conversationsByProjectId()[project.id] ?? [];
      for (const session of conversations) {
        const runtime = sessionStore.byConversationId[session.id];
        const trackedByRuntime = runtime ? hasUnfinishedExecutions(runtime) : false;
        const trackedByServerState =
          session.queue_state === "running" ||
          session.queue_state === "queued" ||
          (session.active_execution_id ?? "").trim() !== "";
        if (session.id === activeSessionId || trackedByRuntime || trackedByServerState) {
          trackedBySessionId.set(session.id, {
            session,
            isGitProject: project.is_git
          });
        }
      }
    }

    return [...trackedBySessionId.values()];
  }

  function syncSessionStreams(): void {
    const trackedContexts = collectTrackedSessionContexts();
    const trackedIds = new Set(trackedContexts.map((item) => item.session.id));
    const token = input.resolveToken();

    for (const context of trackedContexts) {
      ensureSessionRuntime(context.session, context.isGitProject);
      if (!sessionStore.streams[context.session.id]) {
        attachSessionStream(context.session, token);
      }
      void hydrateSessionDetail(context, false);
    }

    for (const streamSessionId of Object.keys(sessionStore.streams)) {
      if (trackedIds.has(streamSessionId)) {
        continue;
      }
      if (streamSessionId === input.activeSessionId()) {
        continue;
      }
      const runtime = sessionStore.byConversationId[streamSessionId];
      if (runtime && hasUnfinishedExecutions(runtime)) {
        continue;
      }
      detachSessionStream(streamSessionId);
    }
  }

  function clearStreams(): void {
    for (const sessionId of Object.keys(sessionStore.streams)) {
      detachSessionStream(sessionId);
    }
    hydratedSessionIds.clear();
  }

  return {
    clearStreams,
    findSessionContextById,
    hydrateSessionDetail,
    syncSessionStreams
  };
}

export function createConversationStreamCoordinator(input: {
  projects: () => Project[];
  conversationsByProjectId: () => Record<string, Session[]>;
  activeConversationId: () => string;
  resolveToken: () => string | undefined;
}) {
  const sessionCoordinator = createSessionStreamCoordinator({
    projects: input.projects,
    conversationsByProjectId: input.conversationsByProjectId,
    activeSessionId: input.activeConversationId,
    resolveToken: input.resolveToken
  });

  function findConversationContextById(conversationId: string): { conversation: Session; isGitProject: boolean } | undefined {
    const context = sessionCoordinator.findSessionContextById(conversationId);
    if (!context) {
      return undefined;
    }
    return {
      conversation: context.session,
      isGitProject: context.isGitProject
    };
  }

  async function hydrateConversationDetail(
    context: { conversation: Session; isGitProject: boolean },
    force = false
  ): Promise<void> {
    await sessionCoordinator.hydrateSessionDetail(
      {
        session: context.conversation,
        isGitProject: context.isGitProject
      },
      force
    );
  }

  return {
    clearStreams: sessionCoordinator.clearStreams,
    findConversationContextById,
    hydrateConversationDetail,
    syncConversationStreams: sessionCoordinator.syncSessionStreams
  };
}
