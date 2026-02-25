import { getControlClient } from "@/shared/services/clients";
import { getControlHubBaseUrl } from "@/shared/runtime";
import { connectConversationEvents } from "@/shared/services/sseClient";
import type {
  Conversation,
  ConversationStreamEvent,
  ConversationDetailResponse,
  DiffCapability,
  DiffItem,
  ExecutionCreateRequest,
  ExecutionCreateResponse
} from "@/shared/types/api";

type ConversationServiceOptions = {
  token?: string;
};

export function streamConversationEvents(
  conversationId: string,
  options: {
    token?: string;
    initialLastEventId?: string;
    onEvent: (event: ConversationStreamEvent) => void;
    onStatusChange: (status: "connected" | "reconnecting" | "disconnected") => void;
    onError: (error: Error) => void;
  }
) {
  const url = `${getHubBaseUrl()}/v1/conversations/${conversationId}/events`;
  return connectConversationEvents(url, {
    token: options.token,
    initialLastEventId: options.initialLastEventId,
    onEvent: options.onEvent,
    onStatusChange: options.onStatusChange,
    onError: options.onError
  });
}

export async function createExecution(conversation: Conversation, input: ExecutionCreateRequest): Promise<ExecutionCreateResponse> {
  return getControlClient().post<ExecutionCreateResponse>(`/v1/conversations/${conversation.id}/messages`, input);
}

export async function getConversationDetail(
  conversationId: string,
  options: ConversationServiceOptions = {}
): Promise<ConversationDetailResponse> {
  return getControlClient().get<ConversationDetailResponse>(`/v1/conversations/${conversationId}`, { token: options.token });
}

export async function cancelExecution(conversationId: string, executionId: string): Promise<void> {
  void executionId;
  await getControlClient().post<void>(`/v1/conversations/${conversationId}/stop`);
}

export async function rollbackExecution(conversationId: string, messageId: string): Promise<void> {
  await getControlClient().post<void>(`/v1/conversations/${conversationId}/rollback`, {
    message_id: messageId
  });
}

export async function commitExecution(executionId: string): Promise<void> {
  await getControlClient().post<void>(`/v1/executions/${executionId}/commit`);
}

export async function discardExecution(executionId: string): Promise<void> {
  await getControlClient().post<void>(`/v1/executions/${executionId}/discard`);
}

export async function loadExecutionDiff(executionId: string): Promise<DiffItem[]> {
  return getControlClient().get<DiffItem[]>(`/v1/executions/${executionId}/diff`);
}

export async function exportExecutionPatch(executionId: string): Promise<string> {
  return getControlClient().get<string>(`/v1/executions/${executionId}/patch`);
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
  return getControlHubBaseUrl();
}
