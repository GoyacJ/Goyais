import { getControlClient } from "@/shared/services/clients";
import { connectConversationEvents } from "@/shared/services/sseClient";
import type {
  Conversation,
  DiffCapability,
  DiffItem,
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
  return getControlClient().post<ExecutionCreateResponse>(`/v1/conversations/${conversation.id}/messages`, input);
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

export async function confirmExecution(executionId: string, decision: "approve" | "deny"): Promise<void> {
  await getControlClient().post<void>(`/v1/executions/${executionId}/confirm`, {
    decision
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
