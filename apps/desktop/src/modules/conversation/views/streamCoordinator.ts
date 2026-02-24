import { getConversationDetail } from "@/modules/conversation/services";
import {
  attachConversationStream,
  conversationStore,
  detachConversationStream,
  ensureConversationRuntime,
  hasUnfinishedExecutions,
  hydrateConversationRuntime
} from "@/modules/conversation/store";
import type { Conversation, Project } from "@/shared/types/api";

type ConversationContext = {
  conversation: Conversation;
  isGitProject: boolean;
};

type StreamCoordinatorInput = {
  projects: () => Project[];
  conversationsByProjectId: () => Record<string, Conversation[]>;
  activeConversationId: () => string;
  resolveToken: () => string | undefined;
};

export function createConversationStreamCoordinator(input: StreamCoordinatorInput) {
  const hydratingConversationIds = new Set<string>();
  const hydratedConversationIds = new Set<string>();

  function findConversationContextById(conversationId: string): ConversationContext | undefined {
    for (const project of input.projects()) {
      const conversations = input.conversationsByProjectId()[project.id] ?? [];
      const conversation = conversations.find((item) => item.id === conversationId);
      if (conversation) {
        return { conversation, isGitProject: project.is_git };
      }
    }
    return undefined;
  }

  async function hydrateConversationDetail(context: ConversationContext, force = false): Promise<void> {
    const conversationId = context.conversation.id;
    if (hydratingConversationIds.has(conversationId)) {
      return;
    }
    if (!force && hydratedConversationIds.has(conversationId)) {
      return;
    }

    const existingRuntime = conversationStore.byConversationId[conversationId];
    const hasHydratedData = Boolean(existingRuntime?.hydrated);
    if (!force && hasHydratedData) {
      return;
    }

    hydratingConversationIds.add(conversationId);
    try {
      const detail = await getConversationDetail(conversationId, { token: input.resolveToken() });
      hydrateConversationRuntime(context.conversation, context.isGitProject, detail);
      hydratedConversationIds.add(conversationId);
    } catch {
      ensureConversationRuntime(context.conversation, context.isGitProject);
    } finally {
      hydratingConversationIds.delete(conversationId);
    }
  }

  function collectTrackedConversationContexts(): ConversationContext[] {
    const activeConversationId = input.activeConversationId();
    const trackedByConversationId = new Map<string, ConversationContext>();

    for (const project of input.projects()) {
      const conversations = input.conversationsByProjectId()[project.id] ?? [];
      for (const conversation of conversations) {
        const runtime = conversationStore.byConversationId[conversation.id];
        const trackedByRuntime = runtime ? hasUnfinishedExecutions(runtime) : false;
        const trackedByServerState =
          conversation.queue_state === "running" ||
          conversation.queue_state === "queued" ||
          (conversation.active_execution_id ?? "").trim() !== "";
        if (conversation.id === activeConversationId || trackedByRuntime || trackedByServerState) {
          trackedByConversationId.set(conversation.id, {
            conversation,
            isGitProject: project.is_git
          });
        }
      }
    }

    return [...trackedByConversationId.values()];
  }

  function syncConversationStreams(): void {
    const trackedContexts = collectTrackedConversationContexts();
    const trackedIds = new Set(trackedContexts.map((item) => item.conversation.id));
    const token = input.resolveToken();

    for (const context of trackedContexts) {
      ensureConversationRuntime(context.conversation, context.isGitProject);
      if (!conversationStore.streams[context.conversation.id]) {
        attachConversationStream(context.conversation, token);
      }
      void hydrateConversationDetail(context, false);
    }

    for (const streamConversationId of Object.keys(conversationStore.streams)) {
      if (trackedIds.has(streamConversationId)) {
        continue;
      }
      if (streamConversationId === input.activeConversationId()) {
        continue;
      }
      const runtime = conversationStore.byConversationId[streamConversationId];
      if (runtime && hasUnfinishedExecutions(runtime)) {
        continue;
      }
      detachConversationStream(streamConversationId);
    }
  }

  function clearStreams(): void {
    for (const conversationId of Object.keys(conversationStore.streams)) {
      detachConversationStream(conversationId);
    }
    hydratedConversationIds.clear();
  }

  return {
    clearStreams,
    findConversationContextById,
    hydrateConversationDetail,
    syncConversationStreams
  };
}
