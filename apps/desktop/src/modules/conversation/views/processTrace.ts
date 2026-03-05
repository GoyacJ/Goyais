import { buildExecutionTraceViewModelData } from "@/modules/conversation/trace/present";
import type { ExecutionTraceStepViewModel, ExecutionTraceViewModelData, TraceLocale } from "@/modules/conversation/trace/types";
import type { Run, RunLifecycleEvent } from "@/shared/types/api";

export type ExecutionTraceStep = ExecutionTraceStepViewModel;
export type ExecutionTraceViewModel = ExecutionTraceViewModelData;

export function buildExecutionTraceViewModels(
  events: RunLifecycleEvent[],
  executions: Run[],
  locale: TraceLocale,
  now: Date = new Date()
): ExecutionTraceViewModel[] {
  return buildExecutionTraceViewModelData(events, executions, locale, now);
}
