import { buildExecutionTraceViewModelData } from "@/modules/conversation/trace/present";
import type { ExecutionTraceStepViewModel, ExecutionTraceViewModelData, TraceLocale } from "@/modules/conversation/trace/types";
import type { Run, RunLifecycleEvent } from "@/shared/types/api";

export type RunTraceStep = ExecutionTraceStepViewModel;
export type RunTraceViewModel = ExecutionTraceViewModelData;

// Backward-compatible aliases while callers migrate to run-based naming.
export type ExecutionTraceStep = RunTraceStep;
export type ExecutionTraceViewModel = RunTraceViewModel;

export function buildRunTraceViewModels(
  events: RunLifecycleEvent[],
  executions: Run[],
  locale: TraceLocale,
  now: Date = new Date()
): RunTraceViewModel[] {
  return buildExecutionTraceViewModelData(events, executions, locale, now);
}

export function buildExecutionTraceViewModels(
  events: RunLifecycleEvent[],
  executions: Run[],
  locale: TraceLocale,
  now: Date = new Date()
): ExecutionTraceViewModel[] {
  return buildRunTraceViewModels(events, executions, locale, now);
}
