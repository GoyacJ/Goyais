import { getControlClient } from "@/shared/services/clients";
import { connectConversationEvents } from "@/shared/services/sseClient";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId } from "@/shared/services/mockData";
import type {
  Conversation,
  DiffCapability,
  DiffItem,
  Execution,
  ExecutionCreateRequest,
  ExecutionCreateResponse
} from "@/shared/types/api";

export function streamConversationEvents(
  conversationId: string,
  options: {
    token?: string;
    onEvent: (event: unknown) => void;
    onStatusChange: (status: "connected" | "reconnecting" | "disconnected") => void;
    onError: (error: Error) => void;
  }
) {
  const url = `${getHubBaseUrl()}/v1/conversations/${conversationId}/events`;
  return connectConversationEvents(url, {
    token: options.token,
    onEvent: options.onEvent as never,
    onStatusChange: options.onStatusChange,
    onError: options.onError
  });
}

export async function createExecution(conversation: Conversation, input: ExecutionCreateRequest): Promise<ExecutionCreateResponse> {
  return withApiFallback(
    "conversation.createExecution",
    () => getControlClient().post<ExecutionCreateResponse>(`/v1/conversations/${conversation.id}/messages`, input),
    () => ({
      execution: {
        id: createMockId("exec"),
        workspace_id: conversation.workspace_id,
        conversation_id: conversation.id,
        message_id: createMockId("msg"),
        state: "queued",
        mode: input.mode,
        model_id: input.model_id,
        queue_index: 0,
        trace_id: createMockId("tr"),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
    })
  );
}

export async function cancelExecution(conversationId: string, executionId: string): Promise<void> {
  void executionId;
  return withApiFallback(
    "conversation.cancelExecution",
    async () => {
      await getControlClient().post<void>(`/v1/conversations/${conversationId}/stop`);
    },
    () => undefined
  );
}

export async function rollbackExecution(conversationId: string, messageId: string): Promise<void> {
  return withApiFallback(
    "conversation.rollback",
    async () => {
      await getControlClient().post<void>(`/v1/conversations/${conversationId}/rollback`, {
        message_id: messageId
      });
    },
    () => undefined
  );
}

export async function commitExecution(executionId: string): Promise<void> {
  return withApiFallback(
    "conversation.commitExecution",
    async () => {
      await getControlClient().post<void>(`/v1/executions/${executionId}/commit`);
    },
    () => undefined
  );
}

export async function discardExecution(executionId: string): Promise<void> {
  return withApiFallback(
    "conversation.discardExecution",
    async () => {
      await getControlClient().post<void>(`/v1/executions/${executionId}/discard`);
    },
    () => undefined
  );
}

export async function loadExecutionDiff(executionId: string): Promise<DiffItem[]> {
  return withApiFallback(
    "conversation.loadExecutionDiff",
    () => getControlClient().get<DiffItem[]>(`/v1/executions/${executionId}/diff`),
    () => [
      {
        id: createMockId("diff"),
        path: "src/modules/conversation/views/MainScreenView.vue",
        change_type: "modified",
        summary: "Refine composer layout and queue indicator"
      },
      {
        id: createMockId("diff"),
        path: "src/shared/ui/BaseButton.vue",
        change_type: "added",
        summary: "Introduce icon-only action style"
      }
    ]
  );
}

export function resolveDiffCapability(isGitProject: boolean): DiffCapability {
  if (isGitProject) {
    return {
      can_commit: true,
      can_discard: true,
      can_export_patch: true
    };
  }

  return {
    can_commit: false,
    can_discard: false,
    can_export_patch: true,
    reason: "Non-Git project: commit and discard are disabled"
  };
}

function getHubBaseUrl(): string {
  return import.meta.env.VITE_HUB_BASE_URL ?? "http://127.0.0.1:8787";
}
