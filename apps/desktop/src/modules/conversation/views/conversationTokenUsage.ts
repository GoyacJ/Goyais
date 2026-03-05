import { normalizeExecutionList } from "@/modules/conversation/store/executionMerge";
import type { SessionRuntime } from "@/modules/conversation/store/state";
import type { Run, Session } from "@/shared/types/api";

export type ConversationTokenUsage = {
  input: number;
  output: number;
  total: number;
};

export function summarizeExecutionTokens(executions: Run[]): ConversationTokenUsage {
  let input = 0;
  let output = 0;
  const normalized = normalizeExecutionList(executions);

  for (const execution of normalized) {
    input += toNonNegativeInteger(execution.tokens_in);
    output += toNonNegativeInteger(execution.tokens_out);
  }

  return {
    input,
    output,
    total: input + output
  };
}

export function resolveConversationUsage(
  conversation: Session | undefined,
  runtime?: Pick<SessionRuntime, "executions">
): ConversationTokenUsage {
  if (runtime) {
    return summarizeExecutionTokens(runtime.executions);
  }

  const input = toNonNegativeInteger(conversation?.tokens_in_total);
  const output = toNonNegativeInteger(conversation?.tokens_out_total);
  const totalFromConversation = toNonNegativeInteger(conversation?.tokens_total);
  const total = totalFromConversation > 0 ? totalFromConversation : input + output;

  return {
    input,
    output,
    total
  };
}

function toNonNegativeInteger(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }
  if (value <= 0) {
    return 0;
  }
  return Math.trunc(value);
}
