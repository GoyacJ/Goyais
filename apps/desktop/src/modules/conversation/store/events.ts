import { createMockId } from "@/shared/utils/id";
import type { ExecutionEvent, RunEventType } from "@/shared/types/api";

export function createExecutionEvent(
  conversationId: string,
  executionId: string,
  queueIndex: number,
  type: RunEventType,
  payload: Record<string, unknown>
): ExecutionEvent {
  return {
    event_id: createMockId("evt"),
    execution_id: executionId,
    conversation_id: conversationId,
    trace_id: createMockId("tr"),
    sequence: Date.now(),
    queue_index: queueIndex,
    type,
    timestamp: new Date().toISOString(),
    payload
  };
}
